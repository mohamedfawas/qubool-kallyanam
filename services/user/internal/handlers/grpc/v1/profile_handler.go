package v1

// import (
// 	"context"

// 	"github.com/google/uuid"
// 	"google.golang.org/grpc/codes"
// 	"google.golang.org/grpc/metadata"
// 	"google.golang.org/grpc/status"

// 	userpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/user/v1"
// 	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
// 	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/models"
// 	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/services"
// )

// // UserHandler implements the user gRPC service
// type UserHandler struct {
// 	userpb.UnimplementedUserServiceServer
// 	profileService *services.ProfileService
// 	logger         logging.Logger
// }

// // NewUserHandler creates a new user handler
// func NewUserHandler(
// 	profileService *services.ProfileService,
// 	logger logging.Logger,
// ) *UserHandler {
// 	return &UserHandler{
// 		profileService: profileService,
// 		logger:         logger,
// 	}
// }

// // CreateProfile handles user profile creation
// func (h *UserHandler) CreateProfile(ctx context.Context, req *userpb.CreateProfileRequest) (*userpb.CreateProfileResponse, error) {
// 	// In a real implementation, extract user ID from JWT token in the context
// 	// For now, we'll use a placeholder implementation
// 	userID, err := extractUserIDFromContext(ctx)
// 	if err != nil {
// 		h.logger.Error("Failed to extract user ID from context", "error", err)
// 		return nil, status.Error(codes.Unauthenticated, "Invalid authentication")
// 	}

// 	// Create profile creation model from request
// 	profileCreation := &models.ProfileCreation{
// 		Name:          req.Name,
// 		Age:           int(req.Age),
// 		Gender:        req.Gender,
// 		Religion:      req.Religion,
// 		Caste:         req.Caste,
// 		MotherTongue:  req.MotherTongue,
// 		MaritalStatus: req.MaritalStatus,
// 		HeightCm:      int(req.HeightCm),
// 		Education:     req.Education,
// 		Occupation:    req.Occupation,
// 		AnnualIncome:  req.AnnualIncome,
// 		City:          req.Location.City,
// 		State:         req.Location.State,
// 		Country:       req.Location.Country,
// 		About:         req.About,
// 	}

// 	// Process profile creation
// 	profile, err := h.profileService.CreateProfile(ctx, userID, profileCreation)
// 	if err != nil {
// 		h.logger.Error("Profile creation failed", "error", err)

// 		// Map domain errors to gRPC status codes
// 		switch {
// 		case err == services.ErrProfileExists:
// 			return nil, status.Error(codes.AlreadyExists, "Profile already exists")
// 		case err == services.ErrInvalidInput:
// 			return nil, status.Error(codes.InvalidArgument, err.Error())
// 		default:
// 			return nil, status.Error(codes.Internal, "Profile creation failed")
// 		}
// 	}

// 	h.logger.Info("Profile created successfully", "profileId", profile.ID)

// 	// Return successful response
// 	return &userpb.CreateProfileResponse{
// 		Success: true,
// 		Id:      profile.ID.String(),
// 		Message: "Profile created successfully",
// 	}, nil
// }
