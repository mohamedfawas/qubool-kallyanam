package services

import (
	"context"
	"fmt"

	adminpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/admin/v1"
	authpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/auth/v1"
	userpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/user/v1"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/admin/internal/clients/auth"
	"github.com/mohamedfawas/qubool-kallyanam/services/admin/internal/clients/user"
)

type AdminService struct {
	authClient *auth.Client
	userClient *user.Client
	logger     logging.Logger
}

func NewAdminService(
	authClient *auth.Client,
	userClient *user.Client,
	logger logging.Logger,
) *AdminService {
	return &AdminService{
		authClient: authClient,
		userClient: userClient,
		logger:     logger,
	}
}

// GetUsers handles user listing by calling auth service only
func (s *AdminService) GetUsers(ctx context.Context, req *adminpb.GetUsersRequest) (*adminpb.GetUsersResponse, error) {
	s.logger.Info("Admin service getting users", "limit", req.Limit, "offset", req.Offset)

	// Call auth service to get users list - provides all needed data for listing
	authResp, err := s.authClient.GetUsersList(ctx, &authpb.GetUsersListRequest{
		Limit:           req.Limit,
		Offset:          req.Offset,
		SortBy:          req.SortBy,
		SortOrder:       req.SortOrder,
		IsActive:        req.IsActive,
		VerifiedOnly:    req.VerifiedOnly,
		PremiumOnly:     req.PremiumOnly,
		CreatedAfter:    req.CreatedAfter,
		CreatedBefore:   req.CreatedBefore,
		LastLoginAfter:  req.LastLoginAfter,
		LastLoginBefore: req.LastLoginBefore,
	})
	if err != nil {
		s.logger.Error("Failed to get users from auth service", "error", err)
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	// Convert auth service response to admin response format
	userSummaries := make([]*adminpb.UserSummary, len(authResp.Users))
	for i, authUser := range authResp.Users {
		userSummaries[i] = &adminpb.UserSummary{
			Id:           authUser.Id,
			Email:        authUser.Email,
			Phone:        authUser.Phone,
			Verified:     authUser.Verified,
			IsActive:     authUser.IsActive,
			IsPremium:    authUser.IsPremium,
			PremiumUntil: authUser.PremiumUntil,
			LastLoginAt:  authUser.LastLoginAt,
			CreatedAt:    authUser.CreatedAt,
			UpdatedAt:    authUser.UpdatedAt,
		}
	}

	return &adminpb.GetUsersResponse{
		Success: authResp.Success,
		Message: authResp.Message,
		Users:   userSummaries,
		Pagination: &adminpb.Pagination{
			Total:       authResp.Pagination.Total,
			Limit:       authResp.Pagination.Limit,
			Offset:      authResp.Pagination.Offset,
			HasMore:     authResp.Pagination.HasMore,
			TotalPages:  authResp.Pagination.TotalPages,
			CurrentPage: authResp.Pagination.CurrentPage,
		},
	}, nil
}

// GetUser handles getting detailed user info by aggregating auth + user services
func (s *AdminService) GetUser(ctx context.Context, req *adminpb.GetUserRequest) (*adminpb.GetUserResponse, error) {
	s.logger.Info("Admin service getting user details", "identifier", req.Identifier)

	// 1. Get auth data first
	authResp, err := s.authClient.GetUser(ctx, &authpb.GetUserRequest{
		Identifier:     req.Identifier,
		IdentifierType: req.IdentifierType,
	})
	if err != nil {
		s.logger.Error("Failed to get user from auth service", "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// 2. Get detailed profile data from user service
	var userProfileData *userpb.DetailedProfileData
	profileResp, err := s.userClient.GetProfileForAdmin(ctx, &userpb.GetProfileForAdminRequest{
		UserUuid: authResp.User.Id,
	})
	if err != nil {
		s.logger.Warn("Failed to get profile data", "error", err, "userUUID", authResp.User.Id)
		// Continue without profile data - user might not have created profile yet
	} else if profileResp.Success {
		userProfileData = profileResp.Profile
	}

	// 3. Build detailed response
	userDetails := &adminpb.UserDetails{
		Auth: &adminpb.AuthData{
			Id:           authResp.User.Id,
			Email:        authResp.User.Email,
			Phone:        authResp.User.Phone,
			Verified:     authResp.User.Verified,
			IsActive:     authResp.User.IsActive,
			IsPremium:    authResp.User.IsPremium,
			PremiumUntil: authResp.User.PremiumUntil,
			LastLoginAt:  authResp.User.LastLoginAt,
			CreatedAt:    authResp.User.CreatedAt,
			UpdatedAt:    authResp.User.UpdatedAt,
		},
	}

	// 4. Convert user service profile data to admin proto format if available
	if userProfileData != nil {
		// Convert userpb.DetailedProfileData to adminpb.DetailedProfileData
		adminProfileData := &adminpb.DetailedProfileData{
			Id:                    userProfileData.Id,
			IsBride:               userProfileData.IsBride,
			FullName:              userProfileData.FullName,
			DateOfBirth:           userProfileData.DateOfBirth,
			HeightCm:              userProfileData.HeightCm,
			PhysicallyChallenged:  userProfileData.PhysicallyChallenged,
			Community:             userProfileData.Community,
			MaritalStatus:         userProfileData.MaritalStatus,
			Profession:            userProfileData.Profession,
			ProfessionType:        userProfileData.ProfessionType,
			HighestEducationLevel: userProfileData.HighestEducationLevel,
			HomeDistrict:          userProfileData.HomeDistrict,
			ProfilePictureUrl:     userProfileData.ProfilePictureUrl,
			LastLogin:             userProfileData.LastLogin,
			Age:                   userProfileData.Age,
		}

		// Convert partner preferences if available
		if userProfileData.PartnerPreferences != nil {
			adminProfileData.PartnerPreferences = &adminpb.PartnerPreferencesData{
				MinAgeYears:                userProfileData.PartnerPreferences.MinAgeYears,
				MaxAgeYears:                userProfileData.PartnerPreferences.MaxAgeYears,
				MinHeightCm:                userProfileData.PartnerPreferences.MinHeightCm,
				MaxHeightCm:                userProfileData.PartnerPreferences.MaxHeightCm,
				AcceptPhysicallyChallenged: userProfileData.PartnerPreferences.AcceptPhysicallyChallenged,
				PreferredCommunities:       userProfileData.PartnerPreferences.PreferredCommunities,
				PreferredMaritalStatus:     userProfileData.PartnerPreferences.PreferredMaritalStatus,
				PreferredProfessions:       userProfileData.PartnerPreferences.PreferredProfessions,
				PreferredProfessionTypes:   userProfileData.PartnerPreferences.PreferredProfessionTypes,
				PreferredEducationLevels:   userProfileData.PartnerPreferences.PreferredEducationLevels,
				PreferredHomeDistricts:     userProfileData.PartnerPreferences.PreferredHomeDistricts,
			}
		}

		// Convert additional photos if available
		if len(userProfileData.AdditionalPhotos) > 0 {
			adminProfileData.AdditionalPhotos = make([]*adminpb.UserPhotoData, len(userProfileData.AdditionalPhotos))
			for i, photo := range userProfileData.AdditionalPhotos {
				adminProfileData.AdditionalPhotos[i] = &adminpb.UserPhotoData{
					PhotoUrl:     photo.PhotoUrl,
					DisplayOrder: photo.DisplayOrder,
					CreatedAt:    photo.CreatedAt,
				}
			}
		}

		// Convert intro video if available
		if userProfileData.IntroVideo != nil {
			adminProfileData.IntroVideo = &adminpb.UserVideoData{
				VideoUrl:        userProfileData.IntroVideo.VideoUrl,
				FileName:        userProfileData.IntroVideo.FileName,
				FileSize:        userProfileData.IntroVideo.FileSize,
				DurationSeconds: userProfileData.IntroVideo.DurationSeconds,
				CreatedAt:       userProfileData.IntroVideo.CreatedAt,
			}
		}

		userDetails.Profile = adminProfileData
	}

	return &adminpb.GetUserResponse{
		Success: true,
		Message: "User details retrieved successfully",
		User:    userDetails,
	}, nil
}
