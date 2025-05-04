package services

import (
	"context"
	"errors"
	"fmt"
	"time"

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
)

// RegistrationService handles user registration logic
type RegistrationService struct {
	registrationRepo repositories.RegistrationRepository
	otpGenerator     *otp.Generator
	otpStore         *otp.Store
	emailClient      *email.Client
}

// NewRegistrationService creates a new registration service
func NewRegistrationService(
	repo repositories.RegistrationRepository,
	otpGenerator *otp.Generator,
	otpStore *otp.Store,
	emailClient *email.Client,
) *RegistrationService {
	return &RegistrationService{
		registrationRepo: repo,
		otpGenerator:     otpGenerator,
		otpStore:         otpStore,
		emailClient:      emailClient,
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
