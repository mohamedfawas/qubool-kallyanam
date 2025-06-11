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
	"github.com/redis/go-redis/v9"
)

const (
	otpPrefix = "otp:" // Define the OTP key prefix in the service layer
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
	otpRepo          repositories.OTPRepository
	otpGenerator     *otp.Generator
	otpExpiryTime    time.Duration
	emailClient      *email.Client
	logger           logging.Logger
}

// NewRegistrationService creates a new registration service
func NewRegistrationService(
	repo repositories.RegistrationRepository,
	otpRepo repositories.OTPRepository,
	otpGenerator *otp.Generator,
	otpExpiryTime time.Duration,
	emailClient *email.Client,
	logger logging.Logger,
) *RegistrationService {
	return &RegistrationService{
		registrationRepo: repo,
		otpRepo:          otpRepo,
		otpGenerator:     otpGenerator,
		otpExpiryTime:    otpExpiryTime,
		emailClient:      emailClient,
		logger:           logger,
	}
}

// getOTPKey formats the OTP key by prefixing with "otp:"
// Example: if identifier is "user@example.com", this returns "otp:user@example.com"
func (s *RegistrationService) getOTPKey(identifier string) string {
	return otpPrefix + identifier
}

// storeOTP generates a new OTP and stores it in Redis with an expiry
func (s *RegistrationService) storeOTP(ctx context.Context, identifier string) (string, error) {
	// Step 1: Generate a random OTP (e.g., "123456")
	otp, err := s.otpGenerator.Generate()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrOTPGenerationFailed, err)
	}

	// Step 2: Create a Redis key like "otp:user@example.com"
	key := s.getOTPKey(identifier)

	// Step 3: Store the OTP in Redis with an expiry time (e.g., 5 minutes)
	if err := s.otpRepo.StoreOTP(ctx, key, otp, s.otpExpiryTime); err != nil {
		return "", fmt.Errorf("failed to store OTP: %w", err)
	}

	return otp, nil
}

// validateOTP checks if an OTP is valid and deletes it if it is
func (s *RegistrationService) validateOTP(ctx context.Context, identifier, inputOTP string) (bool, error) {
	key := s.getOTPKey(identifier)

	// Step 1: Retrieve the stored OTP from Redis
	storedOTP, err := s.otpRepo.GetOTP(ctx, key)
	if err != nil {
		// If key doesn't exist in Redis (OTP expired), return invalid
		if err == redis.Nil {
			return false, ErrInvalidOTP
		}
		return false, fmt.Errorf("failed to retrieve OTP: %w", err)
	}

	// Step 2: Compare the retrieved OTP with the user input
	if storedOTP != inputOTP {
		return false, nil
	}

	// Step 3: Delete the OTP after successful validation (one-time use)
	if err := s.otpRepo.DeleteOTP(ctx, key); err != nil {
		// Log the error but don't fail the validation
		s.logger.Error("Failed to delete OTP after validation", "key", key, "error", err)
	}

	return true, nil
}

// RegisterUser handles the registration process
func (s *RegistrationService) RegisterUser(ctx context.Context, reg *models.Registration) error {
	// Step 1: Validate email, phone, and password formats
	if !validation.ValidateEmail(reg.Email) {
		return fmt.Errorf("%w: invalid email format", ErrInvalidInput)
	}

	if !validation.ValidatePhone(reg.Phone) {
		return fmt.Errorf("%w: invalid phone format", ErrInvalidInput)
	}

	if !validation.ValidatePassword(reg.Password, validation.DefaultPasswordPolicy()) {
		return fmt.Errorf("%w: password does not meet requirements", ErrInvalidInput)
	}

	// Step 2: Check if email already exists in the "users" table
	existsEmail, err := s.registrationRepo.IsRegistered(ctx, "email", reg.Email)
	if err != nil {
		return fmt.Errorf("failed to check email: %w", err)
	}
	if existsEmail {
		return ErrEmailAlreadyExists
	}

	// Step 3: Check if phone already exists in the "users" table
	existsPhone, err := s.registrationRepo.IsRegistered(ctx, "phone", reg.Phone)
	if err != nil {
		return fmt.Errorf("failed to check phone: %w", err)
	}
	if existsPhone {
		return ErrPhoneAlreadyExists
	}

	// Step 4: Clean up old pending registrations using same email
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

	// Step 4: Clean up old pending registrations using same phone
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

	// Step 5: Hash the user's password before storing
	hashedPassword, err := encryption.HashPassword(reg.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Step 6: Set registration timestamps using Indian Standard Time
	now := indianstandardtime.Now()

	// Step 7: Create a new pending registration entry
	pendingReg := &models.PendingRegistration{
		Email:        reg.Email,
		Phone:        reg.Phone,
		PasswordHash: hashedPassword,
		CreatedAt:    now,
		ExpiresAt:    now.Add(1 * time.Hour), // 1 hour TTL
	}

	// Step 8: Store pending registration in DB
	if err := s.registrationRepo.CreatePendingRegistration(ctx, pendingReg); err != nil {
		return fmt.Errorf("%w: %v", ErrRegistrationFailed, err)
	}

	// Step 9: Generate and store OTP for the email
	otp, err := s.storeOTP(ctx, reg.Email)
	if err != nil {
		return err
	}

	// Step 10: Send OTP to user via email
	if err := s.emailClient.SendOTPEmail(reg.Email, otp); err != nil {
		return fmt.Errorf("failed to send OTP email: %w", err)
	}

	return nil
}

// VerifyRegistration verifies a user's email with OTP
func (s *RegistrationService) VerifyRegistration(ctx context.Context, email, otp string) error {
	// Step 1: Validate email format
	if !validation.ValidateEmail(email) {
		s.logger.Debug("Invalid email format", "email", email)
		return fmt.Errorf("%w: invalid email format", ErrInvalidInput)
	}

	// Step 2: Retrieve pending registration from DB using email
	pendingReg, err := s.registrationRepo.GetPendingRegistration(ctx, "email", email)
	if err != nil {
		s.logger.Error("Failed to get pending registration", "email", email, "error", err)
		return fmt.Errorf("failed to get pending registration: %w", err)
	}

	if pendingReg == nil {
		s.logger.Debug("No pending registration found", "email", email)
		return fmt.Errorf("no pending registration found for email: %s", email)
	}

	// Step 3: Validate OTP using stored value
	valid, err := s.validateOTP(ctx, email, otp)
	if err != nil {
		s.logger.Error("OTP validation error", "email", email, "error", err)
		return fmt.Errorf("OTP verification error: %w", err)
	}

	if !valid {
		s.logger.Debug("Invalid OTP provided", "email", email)
		return ErrInvalidOTP
	}

	s.logger.Info("OTP verified successfully", "email", email)

	// Step 4: Create a fully verified "user" entry from pending registration
	user := &models.User{
		Email:        pendingReg.Email,
		Phone:        pendingReg.Phone,
		PasswordHash: pendingReg.PasswordHash,
		Verified:     true,
		CreatedAt:    indianstandardtime.Now(),
		UpdatedAt:    indianstandardtime.Now(),
	}

	// Step 5: Store the verified user data in the DB (in "users" table)
	if err := s.registrationRepo.CreateUser(ctx, user); err != nil {
		s.logger.Error("Failed to create user", "email", email, "error", err)
		return fmt.Errorf("%w: %v", ErrVerificationFailed, err)
	}

	s.logger.Info("User created successfully", "email", email, "id", user.ID)

	// Step 6: Clean up the pending registration
	if err := s.registrationRepo.DeletePendingRegistration(ctx, pendingReg.ID); err != nil {
		s.logger.Error("Failed to delete pending registration", "id", pendingReg.ID, "error", err)
		// User is already created, so we don’t stop the flow
	} else {
		s.logger.Debug("Pending registration deleted", "id", pendingReg.ID)
	}

	return nil
}
