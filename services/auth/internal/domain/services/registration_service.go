package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/notifications/email"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/security/encryption"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/security/otp"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/indianstandardtime"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/validation"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/repositories"
)

var (
	ErrInvalidInput        = errors.New("invalid input parameters")
	ErrEmailAlreadyExists  = errors.New("email already registered")
	ErrPhoneAlreadyExists  = errors.New("phone number already registered")
	ErrRegistrationFailed  = errors.New("registration failed")
	ErrOTPGenerationFailed = errors.New("failed to generate OTP")
	ErrInvalidOTP          = errors.New("invalid or expired OTP")
	ErrVerificationFailed  = errors.New("verification failed")
)

// RegistrationService handles user registration logic
type RegistrationService struct {
	registrationRepo repositories.RegistrationRepository
	otpGenerator     *otp.Generator
	otpStore         *otp.Store
	emailClient      *email.Client
	logger           logging.Logger
}

// NewRegistrationService creates a new registration service
func NewRegistrationService(
	repo repositories.RegistrationRepository,
	otpGenerator *otp.Generator,
	otpStore *otp.Store,
	emailClient *email.Client,
	logger logging.Logger,
) *RegistrationService {
	return &RegistrationService{
		registrationRepo: repo,
		otpGenerator:     otpGenerator,
		otpStore:         otpStore,
		emailClient:      emailClient,
		logger:           logger,
	}
}

// RegisterUser handles the registration process
func (s *RegistrationService) RegisterUser(ctx context.Context, reg *models.Registration) error {
	// Validate input
	if !validation.ValidateEmail(reg.Email) {
		return fmt.Errorf("%w: invalid email format", ErrInvalidInput)
	}

	if !validation.ValidatePhone(reg.Phone) {
		return fmt.Errorf("%w: invalid phone format", ErrInvalidInput)
	}

	if !validation.ValidatePassword(reg.Password, validation.DefaultPasswordPolicy()) {
		return fmt.Errorf("%w: password does not meet requirements", ErrInvalidInput)
	}

	// Check if email is already registered
	existsEmail, err := s.registrationRepo.IsRegistered(ctx, "email", reg.Email)
	if err != nil {
		return fmt.Errorf("failed to check email: %w", err)
	}
	if existsEmail {
		return ErrEmailAlreadyExists
	}

	// Check if phone is already registered
	existsPhone, err := s.registrationRepo.IsRegistered(ctx, "phone", reg.Phone)
	if err != nil {
		return fmt.Errorf("failed to check phone: %w", err)
	}
	if existsPhone {
		return ErrPhoneAlreadyExists
	}

	// Check if there's a pending registration with this email
	pendingEmail, err := s.registrationRepo.GetPendingRegistration(ctx, "email", reg.Email)
	if err != nil {
		return fmt.Errorf("failed to check pending registration: %w", err)
	}
	if pendingEmail != nil {
		// If pending registration exists, delete it
		err = s.registrationRepo.DeletePendingRegistration(ctx, pendingEmail.ID)
		if err != nil {
			return fmt.Errorf("failed to delete existing pending registration: %w", err)
		}
	}

	// Check if there's a pending registration with this phone
	pendingPhone, err := s.registrationRepo.GetPendingRegistration(ctx, "phone", reg.Phone)
	if err != nil {
		return fmt.Errorf("failed to check pending registration: %w", err)
	}
	if pendingPhone != nil && (pendingEmail == nil || pendingEmail.ID != pendingPhone.ID) {
		// If different pending registration exists with this phone, delete it
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

	// Set the current time using Indian Standard Time
	now := indianstandardtime.Now()

	// Create pending registration
	pendingReg := &models.PendingRegistration{
		Email:        reg.Email,
		Phone:        reg.Phone,
		PasswordHash: hashedPassword,
		CreatedAt:    now,
		ExpiresAt:    now.Add(1 * time.Hour), // 1 hour TTL
	}

	// Store pending registration
	if err := s.registrationRepo.CreatePendingRegistration(ctx, pendingReg); err != nil {
		return fmt.Errorf("%w: %v", ErrRegistrationFailed, err)
	}

	// Generate OTP
	otp, err := s.otpGenerator.Generate()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrOTPGenerationFailed, err)
	}

	// Store OTP in Redis
	if err := s.otpStore.StoreOTP(ctx, reg.Email, otp); err != nil {
		return fmt.Errorf("failed to store OTP: %w", err)
	}

	// Send OTP via email
	if err := s.emailClient.SendOTPEmail(reg.Email, otp); err != nil {
		return fmt.Errorf("failed to send OTP email: %w", err)
	}

	return nil
}

// VerifyRegistration verifies a user's email with OTP
func (s *RegistrationService) VerifyRegistration(ctx context.Context, email, otp string) error {
	// Validate email format
	if !validation.ValidateEmail(email) {
		s.logger.Debug("Invalid email format", "email", email)
		return fmt.Errorf("%w: invalid email format", ErrInvalidInput)
	}

	// Get pending registration
	pendingReg, err := s.registrationRepo.GetPendingRegistration(ctx, "email", email)
	if err != nil {
		s.logger.Error("Failed to get pending registration", "email", email, "error", err)
		return fmt.Errorf("failed to get pending registration: %w", err)
	}

	if pendingReg == nil {
		s.logger.Debug("No pending registration found", "email", email)
		return fmt.Errorf("no pending registration found for email: %s", email)
	}

	// Verify OTP
	valid, err := s.otpStore.ValidateOTP(ctx, email, otp)
	if err != nil {
		s.logger.Error("OTP validation error", "email", email, "error", err)
		return fmt.Errorf("OTP verification error: %w", err)
	}

	if !valid {
		s.logger.Debug("Invalid OTP provided", "email", email)
		return ErrInvalidOTP
	}

	s.logger.Info("OTP verified successfully", "email", email)

	// Create new user from pending registration
	user := &models.User{
		Email:        pendingReg.Email,
		Phone:        pendingReg.Phone,
		PasswordHash: pendingReg.PasswordHash,
		Verified:     true,
		CreatedAt:    indianstandardtime.Now(),
		UpdatedAt:    indianstandardtime.Now(),
	}

	// Create user in database
	if err := s.registrationRepo.CreateUser(ctx, user); err != nil {
		s.logger.Error("Failed to create user", "email", email, "error", err)
		return fmt.Errorf("%w: %v", ErrVerificationFailed, err)
	}

	s.logger.Info("User created successfully", "email", email, "id", user.ID)

	// Delete pending registration
	if err := s.registrationRepo.DeletePendingRegistration(ctx, pendingReg.ID); err != nil {
		s.logger.Error("Failed to delete pending registration", "id", pendingReg.ID, "error", err)
		// We continue despite this error since user is already created
	} else {
		s.logger.Debug("Pending registration deleted", "id", pendingReg.ID)
	}

	return nil
}
