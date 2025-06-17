package user

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
)

func (h *Handler) GetDetailedProfile(c *gin.Context) {
	// Extract profile ID from path parameter
	profileIDStr := c.Param("id")
	if profileIDStr == "" {
		h.logger.Debug("Missing profile ID in path")
		pkghttp.Error(c, pkghttp.NewBadRequest("Profile ID is required", nil))
		return
	}

	profileID, err := strconv.ParseUint(profileIDStr, 10, 64)
	if err != nil {
		h.logger.Debug("Invalid profile ID format", "profileID", profileIDStr)
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid profile ID format", err))
		return
	}

	if profileID == 0 {
		h.logger.Debug("Profile ID cannot be zero")
		pkghttp.Error(c, pkghttp.NewBadRequest("Profile ID must be greater than 0", nil))
		return
	}

	// Call user service
	success, message, profileData, err := h.userClient.GetDetailedProfile(
		c.Request.Context(),
		profileID,
	)

	if err != nil {
		h.logger.Error("Failed to get detailed profile", "error", err, "profileID", profileID)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	if !success {
		pkghttp.Error(c, pkghttp.NewBadRequest(message, nil))
		return
	}

	// Format the response
	response := gin.H{
		"id":                      profileData.Id,
		"full_name":               profileData.FullName,
		"is_bride":                profileData.IsBride,
		"age":                     profileData.Age,
		"physically_challenged":   profileData.PhysicallyChallenged,
		"community":               profileData.Community,
		"marital_status":          profileData.MaritalStatus,
		"profession":              profileData.Profession,
		"profession_type":         profileData.ProfessionType,
		"highest_education_level": profileData.HighestEducationLevel,
		"home_district":           profileData.HomeDistrict,
		"last_login":              profileData.LastLogin.AsTime(),
	}

	// Add optional fields
	if profileData.DateOfBirth != "" {
		response["date_of_birth"] = profileData.DateOfBirth
	}

	if profileData.HeightCm > 0 {
		response["height_cm"] = profileData.HeightCm
	}

	if profileData.ProfilePictureUrl != "" {
		response["profile_picture_url"] = profileData.ProfilePictureUrl
	}

	// Add partner preferences if available
	if profileData.PartnerPreferences != nil {
		prefs := profileData.PartnerPreferences
		response["partner_preferences"] = gin.H{
			"min_age_years":                prefs.MinAgeYears,
			"max_age_years":                prefs.MaxAgeYears,
			"min_height_cm":                prefs.MinHeightCm,
			"max_height_cm":                prefs.MaxHeightCm,
			"accept_physically_challenged": prefs.AcceptPhysicallyChallenged,
			"preferred_communities":        prefs.PreferredCommunities,
			"preferred_marital_status":     prefs.PreferredMaritalStatus,
			"preferred_professions":        prefs.PreferredProfessions,
			"preferred_profession_types":   prefs.PreferredProfessionTypes,
			"preferred_education_levels":   prefs.PreferredEducationLevels,
			"preferred_home_districts":     prefs.PreferredHomeDistricts,
		}
	}

	// Add additional photos if available
	if len(profileData.AdditionalPhotos) > 0 {
		photos := make([]gin.H, len(profileData.AdditionalPhotos))
		for i, photo := range profileData.AdditionalPhotos {
			photos[i] = gin.H{
				"photo_url":     photo.PhotoUrl,
				"display_order": photo.DisplayOrder,
				"created_at":    photo.CreatedAt.AsTime(),
			}
		}
		response["additional_photos"] = photos
	}

	// Add intro video if available
	if profileData.IntroVideo != nil {
		video := profileData.IntroVideo
		response["intro_video"] = gin.H{
			"video_url":        video.VideoUrl,
			"file_name":        video.FileName,
			"file_size":        video.FileSize,
			"duration_seconds": video.DurationSeconds,
			"created_at":       video.CreatedAt.AsTime(),
		}
	}

	pkghttp.Success(c, http.StatusOK, message, gin.H{
		"profile": response,
	})
}
