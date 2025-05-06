package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/security/encryption"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/indianstandardtime"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/validation"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/repositories"
)

var (
	ErrInvalidAdminInput = errors.New("invalid admin input parameters")
)

type AdminService struct {
	adminRepo repositories.AdminRepository
	logger    logging.Logger
}

func NewAdminService(
	adminRepo repositories.AdminRepository,
	logger logging.Logger,
) *AdminService {
	return &AdminService{
		adminRepo: adminRepo,
		logger:    logger,
	}
}

// InitializeDefaultAdmin checks if any admin exists and creates a default one if not
func (s *AdminService) InitializeDefaultAdmin(ctx context.Context, defaultEmail, defaultPassword string) error {
	// Check if any admin exists
	exists, err := s.adminRepo.CheckAdminExists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if admin exists: %w", err)
	}

	// If admin already exists, nothing to do
	if exists {
		s.logger.Info("Admin already exists, skipping default admin creation")
		return nil
	}

	// Validate admin input
	if !validation.ValidateEmail(defaultEmail) {
		return fmt.Errorf("%w: invalid admin email format", ErrInvalidAdminInput)
	}

	if !validation.ValidatePassword(defaultPassword, validation.DefaultPasswordPolicy()) {
		return fmt.Errorf("%w: admin password does not meet requirements", ErrInvalidAdminInput)
	}

	// Hash password
	hashedPassword, err := encryption.HashPassword(defaultPassword)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	// Create admin
	now := indianstandardtime.Now()
	admin := &models.Admin{
		Email:        defaultEmail,
		PasswordHash: hashedPassword,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.adminRepo.CreateAdmin(ctx, admin); err != nil {
		return fmt.Errorf("failed to create default admin: %w", err)
	}

	s.logger.Info("Default admin account created successfully", "email", defaultEmail)
	return nil
}
