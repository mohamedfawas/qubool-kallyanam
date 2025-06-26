package v1

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"
	authpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/auth/v1"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/repositories"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/services"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/helpers"
)

type AuthHandler struct {
	authpb.UnimplementedAuthServiceServer
	registrationService *services.RegistrationService
	authService         *services.AuthService
	logger              logging.Logger
}

func NewAuthHandler(
	registrationService *services.RegistrationService,
	authService *services.AuthService,
	logger logging.Logger,
) *AuthHandler {
	return &AuthHandler{
		registrationService: registrationService,
		authService:         authService,
		logger:              logger,
	}
}

func (h *AuthHandler) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.RegisterResponse, error) {
	h.logger.Info("Received registration request", "email", req.Email, "phone", req.Phone)

	registration := &models.Registration{
		Email:    req.Email,
		Phone:    req.Phone,
		Password: req.Password,
	}

	err := h.registrationService.RegisterUser(ctx, registration)
	if err != nil {
		h.logger.Error("Registration failed", "error", err)
		return nil, helpers.MapErrorToGRPCStatus(err)
	}

	h.logger.Info("Registration successful, OTP sent", "email", req.Email)

	return &authpb.RegisterResponse{
		Success: true,
		Message: "OTP sent to registered email",
	}, nil
}

func (h *AuthHandler) Verify(ctx context.Context, req *authpb.VerifyRequest) (*authpb.VerifyResponse, error) {
	h.logger.Info("Received verification request", "email", req.Email)

	err := h.registrationService.VerifyRegistration(ctx, req.Email, req.Otp)
	if err != nil {
		h.logger.Error("Verification failed", "error", err)

		// Handle specific case for pending registration not found
		if strings.Contains(err.Error(), "no pending registration found") {
			return nil, status.Error(codes.NotFound, "No pending registration found")
		}

		return nil, helpers.MapErrorToGRPCStatus(err)
	}

	h.logger.Info("Verification successful", "email", req.Email)

	return &authpb.VerifyResponse{
		Success: true,
		Message: "Account verified successfully",
	}, nil
}

func (h *AuthHandler) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	h.logger.Info("Received login request", "email", req.Email)

	// Validate input using helper
	if err := helpers.ValidateLoginInput(req.Email, req.Password); err != nil {
		h.logger.Debug("Invalid login request - missing required fields")
		return nil, helpers.MapErrorToGRPCStatus(err)
	}

	tokenPair, err := h.authService.Login(ctx, req.Email, req.Password)
	if err != nil {
		h.logger.Error("Login failed", "email", req.Email, "error", err)
		return nil, helpers.MapErrorToGRPCStatus(err)
	}

	h.logger.Info("Login successful", "email", req.Email)

	return &authpb.LoginResponse{
		Success:      true,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		Message:      "Login successful",
	}, nil
}

func (h *AuthHandler) Logout(ctx context.Context, req *authpb.LogoutRequest) (*authpb.LogoutResponse, error) {
	h.logger.Info("Received logout request")

	if req.AccessToken == "" {
		h.logger.Debug("Invalid logout request - missing token")
		return nil, status.Error(codes.InvalidArgument, "Access token is required")
	}

	err := h.authService.Logout(ctx, req.AccessToken)
	if err != nil {
		h.logger.Error("Logout failed", "error", err)
		return nil, helpers.MapErrorToGRPCStatus(err)
	}

	h.logger.Info("Logout successful")

	return &authpb.LogoutResponse{
		Success: true,
		Message: "Logout successful",
	}, nil
}

func (h *AuthHandler) RefreshToken(ctx context.Context, req *authpb.RefreshTokenRequest) (*authpb.RefreshTokenResponse, error) {
	h.logger.Info("Received token refresh request")

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		h.logger.Debug("Metadata missing from context")
		return nil, status.Error(codes.Unauthenticated, "Missing authorization")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		h.logger.Debug("Authorization header missing")
		return nil, status.Error(codes.Unauthenticated, "Missing authorization header")
	}

	authHeader := values[0]
	if !strings.HasPrefix(authHeader, "Bearer ") {
		h.logger.Debug("Invalid authorization format")
		return nil, status.Error(codes.Unauthenticated, "Invalid authorization format")
	}

	refreshToken := strings.TrimPrefix(authHeader, "Bearer ")

	tokenPair, err := h.authService.RefreshToken(ctx, refreshToken)
	if err != nil {
		h.logger.Error("Refresh failed", "error", err)
		return nil, helpers.MapErrorToGRPCStatus(err)
	}

	h.logger.Info("Token refresh successful")

	return &authpb.RefreshTokenResponse{
		Success:      true,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		Message:      "Token refresh successful",
	}, nil
}

func (h *AuthHandler) AdminLogin(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	h.logger.Info("Received admin login request", "email", req.Email)

	// Validate input using helper
	if err := helpers.ValidateLoginInput(req.Email, req.Password); err != nil {
		h.logger.Debug("Invalid admin login request - missing required fields")
		return nil, helpers.MapErrorToGRPCStatus(err)
	}

	tokenPair, err := h.authService.AdminLogin(ctx, req.Email, req.Password)
	if err != nil {
		h.logger.Error("Admin login failed", "email", req.Email, "error", err)
		return nil, helpers.MapErrorToGRPCStatus(err)
	}

	h.logger.Info("Admin login successful", "email", req.Email)

	return &authpb.LoginResponse{
		Success:      true,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		Message:      "Admin login successful",
	}, nil
}

func (h *AuthHandler) Delete(ctx context.Context, req *authpb.DeleteRequest) (*authpb.DeleteResponse, error) {
	h.logger.Info("Received delete account request")

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		h.logger.Debug("Metadata missing from context")
		return nil, status.Error(codes.Unauthenticated, "Missing authorization")
	}

	userIDs := md.Get("user-id")
	if len(userIDs) == 0 {
		h.logger.Debug("User ID missing from metadata")
		return nil, status.Error(codes.Unauthenticated, "Authentication required")
	}

	userID := userIDs[0]

	if req.Password == "" {
		h.logger.Debug("Invalid delete request - missing password")
		return nil, status.Error(codes.InvalidArgument, "Password is required")
	}

	err := h.authService.Delete(ctx, userID, req.Password)
	if err != nil {
		h.logger.Error("Delete failed", "userID", userID, "error", err)
		return nil, helpers.MapErrorToGRPCStatus(err)
	}

	h.logger.Info("User account deleted successfully", "userID", userID)

	return &authpb.DeleteResponse{
		Success: true,
		Message: "Account deleted successfully",
	}, nil
}

// GetUsersList handles admin request to list users
func (h *AuthHandler) GetUsersList(ctx context.Context, req *authpb.GetUsersListRequest) (*authpb.GetUsersListResponse, error) {
	h.logger.Info("Admin get users list request", "limit", req.Limit, "offset", req.Offset)

	// Set defaults (same pattern as existing methods)
	limit := int(req.Limit)
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	offset := int(req.Offset)
	if offset < 0 {
		offset = 0
	}

	// Build parameters using existing repository pattern
	params := repositories.GetUsersParams{
		Limit:     limit,
		Offset:    offset,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		// Remove Status field - it doesn't exist in current proto
	}

	// Handle optional boolean filters correctly
	if req.IsActive != nil {
		// Convert IsActive to Status string for repository
		if *req.IsActive {
			params.Status = "active"
		} else {
			params.Status = "inactive"
		}
	}

	if req.VerifiedOnly != nil && *req.VerifiedOnly {
		verified := true
		params.VerifiedOnly = &verified
	}

	if req.PremiumOnly != nil && *req.PremiumOnly {
		premium := true
		params.PremiumOnly = &premium
	}

	// Handle timestamp filters
	if req.CreatedAfter != nil {
		createdAfter := req.CreatedAfter.AsTime()
		params.CreatedAfter = &createdAfter
	}
	if req.CreatedBefore != nil {
		createdBefore := req.CreatedBefore.AsTime()
		params.CreatedBefore = &createdBefore
	}
	if req.LastLoginAfter != nil {
		lastLoginAfter := req.LastLoginAfter.AsTime()
		params.LastLoginAfter = &lastLoginAfter
	}
	if req.LastLoginBefore != nil {
		lastLoginBefore := req.LastLoginBefore.AsTime()
		params.LastLoginBefore = &lastLoginBefore
	}

	// Call repository (reuse existing pattern)
	users, total, err := h.authService.GetUsersList(ctx, params)
	if err != nil {
		h.logger.Error("Failed to get users list", "error", err)
		return nil, status.Error(codes.Internal, "Failed to retrieve users")
	}

	// Convert to protobuf (reuse conversion pattern)
	userDataList := make([]*authpb.UserData, len(users))
	for i, user := range users {
		userDataList[i] = h.convertUserToProtobuf(user)
	}

	// Calculate pagination
	hasMore := int64(offset+limit) < total
	totalPages := int32((total + int64(limit) - 1) / int64(limit))
	currentPage := int32((offset / limit) + 1)

	return &authpb.GetUsersListResponse{
		Success: true,
		Message: "Users retrieved successfully",
		Users:   userDataList,
		Pagination: &authpb.PaginationData{
			Total:       int32(total),
			Limit:       int32(limit),
			Offset:      int32(offset),
			HasMore:     hasMore,
			TotalPages:  totalPages,
			CurrentPage: currentPage,
		},
	}, nil
}

// GetUser handles admin request to get single user by UUID, email, or phone
func (h *AuthHandler) GetUser(ctx context.Context, req *authpb.GetUserRequest) (*authpb.GetUserResponse, error) {
	h.logger.Info("Admin get user request", "identifier", req.Identifier, "type", req.IdentifierType)

	if req.Identifier == "" {
		return nil, status.Error(codes.InvalidArgument, "Identifier cannot be empty")
	}

	// Determine field to search (reuse existing GetUser pattern)
	var field string
	switch req.IdentifierType {
	case "uuid":
		field = "id"
	case "email":
		field = "email"
	case "phone":
		field = "phone"
	default:
		// Auto-detect based on format
		if isValidUUID(req.Identifier) {
			field = "id"
		} else if isValidEmail(req.Identifier) {
			field = "email"
		} else {
			field = "phone"
		}
	}

	// Call existing repository method (reuse existing pattern)
	user, err := h.authService.GetUserByIdentifier(ctx, field, req.Identifier)
	if err != nil {
		h.logger.Error("Failed to get user", "error", err, "identifier", req.Identifier)
		return nil, status.Error(codes.Internal, "Failed to retrieve user")
	}

	if user == nil {
		return nil, status.Error(codes.NotFound, "User not found")
	}

	return &authpb.GetUserResponse{
		Success: true,
		Message: "User retrieved successfully",
		User:    h.convertUserToProtobuf(user),
	}, nil
}

// Helper method to convert user model to protobuf (reuse existing conversion pattern)
func (h *AuthHandler) convertUserToProtobuf(user *models.User) *authpb.UserData {
	userData := &authpb.UserData{
		Id:        user.ID.String(),
		Email:     user.Email,
		Phone:     user.Phone,
		Verified:  user.Verified,
		IsActive:  user.IsActive,
		IsPremium: user.IsPremium(),
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}

	if user.PremiumUntil != nil {
		userData.PremiumUntil = timestamppb.New(*user.PremiumUntil)
	}

	if user.LastLoginAt != nil {
		userData.LastLoginAt = timestamppb.New(*user.LastLoginAt)
	}

	return userData
}

// Helper validation functions
func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

func isValidEmail(s string) bool {
	return strings.Contains(s, "@")
}
