package services

import (
	"context"
	"fmt"
	"time"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/notifications/email"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/security/encryption"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/security/otp"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/indianstandardtime"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/validation"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/constants"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/repositories"
	autherrors "github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/errors"
	"github.com/redis/go-redis/v9"
)

type RegistrationService struct {
	registrationRepo repositories.RegistrationRepository
	userRepo         repositories.UserRepository
	otpRepo          repositories.OTPRepository
	otpGenerator     *otp.Generator
	otpExpiryTime    time.Duration
	emailClient      *email.Client
	logger           logging.Logger
}

func NewRegistrationService(
	registrationRepo repositories.RegistrationRepository,
	userRepo repositories.UserRepository,
	otpRepo repositories.OTPRepository,
	otpGenerator *otp.Generator,
	otpExpiryTime time.Duration,
	emailClient *email.Client,
	logger logging.Logger,
) *RegistrationService {
	return &RegistrationService{
		registrationRepo: registrationRepo,
		userRepo:         userRepo,
		otpRepo:          otpRepo,
		otpGenerator:     otpGenerator,
		otpExpiryTime:    otpExpiryTime,
		emailClient:      emailClient,
		logger:           logger,
	}
}

// Example: if identifier is "user@example.com", this returns "otp:user@example.com"
func (s *RegistrationService) getOTPKey(identifier string) string {
	return constants.OTPPrefix + identifier
}

// storeOTP generates a new OTP and stores it in Redis with an expiry
func (s *RegistrationService) storeOTP(ctx context.Context, identifier string) (string, error) {
	otp, err := s.otpGenerator.Generate()
	if err != nil {
		return "", fmt.Errorf("%w: %v", autherrors.ErrOTPGenerationFailed, err)
	}

	key := s.getOTPKey(identifier)
	if err := s.otpRepo.StoreOTP(ctx, key, otp, s.otpExpiryTime); err != nil {
		return "", fmt.Errorf("failed to store OTP: %w", err)
	}

	return otp, nil
}

// validateOTP checks if an OTP is valid and deletes it if it is
func (s *RegistrationService) validateOTP(ctx context.Context, identifier, inputOTP string) (bool, error) {
	key := s.getOTPKey(identifier)

	storedOTP, err := s.otpRepo.GetOTP(ctx, key)
	if err != nil {
		if err == redis.Nil {
			return false, autherrors.ErrInvalidOTP
		}
		return false, fmt.Errorf("failed to retrieve OTP: %w", err)
	}

	if storedOTP != inputOTP {
		return false, nil
	}

	// Delete the OTP after successful validation
	if err := s.otpRepo.DeleteOTP(ctx, key); err != nil {
		s.logger.Error("Failed to delete OTP after validation", "key", key, "error", err)
	}

	return true, nil
}

func (s *RegistrationService) RegisterUser(ctx context.Context, reg *models.Registration) error {
	if !validation.ValidateEmail(reg.Email) {
		return fmt.Errorf("%w: invalid email format", autherrors.ErrInvalidInput)
	}

	if !validation.ValidatePhone(reg.Phone) {
		return fmt.Errorf("%w: invalid phone format", autherrors.ErrInvalidInput)
	}

	if !validation.ValidatePassword(reg.Password, validation.DefaultPasswordPolicy()) {
		return fmt.Errorf("%w: password does not meet requirements", autherrors.ErrInvalidInput)
	}

	existsEmail, err := s.userRepo.IsRegistered(ctx, "email", reg.Email)
	if err != nil {
		return fmt.Errorf("failed to check email: %w", err)
	}
	if existsEmail {
		return autherrors.ErrEmailAlreadyExists
	}

	existsPhone, err := s.userRepo.IsRegistered(ctx, "phone", reg.Phone)
	if err != nil {
		return fmt.Errorf("failed to check phone: %w", err)
	}
	if existsPhone {
		return autherrors.ErrPhoneAlreadyExists
	}

	// Clean up old pending registrations using same email
	pendingEmail, err := s.registrationRepo.GetPendingRegistration(ctx, "email", reg.Email)
	if err != nil {
		return fmt.Errorf("failed to check pending registration: %w", err)
	}
	if pendingEmail != nil {
		err = s.registrationRepo.DeletePendingRegistration(ctx, pendingEmail.ID)
		if err != nil {
			return fmt.Errorf("failed to delete existing pending registration: %w", err)
		}
	}

	// Clean up old pending registrations using same phone
	pendingPhone, err := s.registrationRepo.GetPendingRegistration(ctx, "phone", reg.Phone)
	if err != nil {
		return fmt.Errorf("failed to check pending registration: %w", err)
	}
	if pendingPhone != nil && (pendingEmail == nil || pendingEmail.ID != pendingPhone.ID) {
		err = s.registrationRepo.DeletePendingRegistration(ctx, pendingPhone.ID)
		if err != nil {
			return fmt.Errorf("failed to delete existing pending registration: %w", err)
		}
	}

	// Hash password
	hashedPassword, err := encryption.HashPassword(reg.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Create pending registration
	now := indianstandardtime.Now()
	pendingReg := &models.PendingRegistration{
		Email:        reg.Email,
		Phone:        reg.Phone,
		PasswordHash: hashedPassword,
		CreatedAt:    now,
		ExpiresAt:    now.Add(1 * time.Hour),
	}

	if err := s.registrationRepo.CreatePendingRegistration(ctx, pendingReg); err != nil {
		return fmt.Errorf("%w: %v", autherrors.ErrRegistrationFailed, err)
	}

	otp, err := s.storeOTP(ctx, reg.Email)
	if err != nil {
		return err
	}

	if err := s.emailClient.SendOTPEmail(reg.Email, otp); err != nil {
		return fmt.Errorf("failed to send OTP email: %w", err)
	}

	return nil
}

func (s *RegistrationService) VerifyRegistration(ctx context.Context, email, otp string) error {
	if !validation.ValidateEmail(email) {
		s.logger.Debug("Invalid email format", "email", email)
		return fmt.Errorf("%w: invalid email format", autherrors.ErrInvalidInput)
	}

	pendingReg, err := s.registrationRepo.GetPendingRegistration(ctx, "email", email)
	if err != nil {
		s.logger.Error("Failed to get pending registration", "email", email, "error", err)
		return fmt.Errorf("failed to get pending registration: %w", err)
	}
	if pendingReg == nil {
		s.logger.Debug("No pending registration found", "email", email)
		return fmt.Errorf("no pending registration found for email: %s", email)
	}

	valid, err := s.validateOTP(ctx, email, otp)
	if err != nil {
		s.logger.Error("OTP validation error", "email", email, "error", err)
		return fmt.Errorf("OTP verification error: %w", err)
	}
	if !valid {
		s.logger.Debug("Invalid OTP provided", "email", email)
		return autherrors.ErrInvalidOTP
	}

	s.logger.Info("OTP verified successfully", "email", email)

	now := indianstandardtime.Now()
	user := &models.User{
		Email:        pendingReg.Email,
		Phone:        pendingReg.Phone,
		PasswordHash: pendingReg.PasswordHash,
		Verified:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		s.logger.Error("Failed to create user", "email", email, "error", err)
		return fmt.Errorf("%w: %v", autherrors.ErrVerificationFailed, err)
	}

	s.logger.Info("User created successfully", "email", email, "id", user.ID)

	if err := s.registrationRepo.DeletePendingRegistration(ctx, pendingReg.ID); err != nil {
		s.logger.Error("Failed to delete pending registration", "id", pendingReg.ID, "error", err)
	} else {
		s.logger.Debug("Pending registration deleted", "id", pendingReg.ID)
	}

	return nil
}
