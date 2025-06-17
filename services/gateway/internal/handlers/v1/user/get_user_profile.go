package user

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/middleware"
)

func (h *Handler) GetProfile(c *gin.Context) {
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	success, message, profileData, err := h.userClient.GetProfile(c.Request.Context(), userID.(string))
	if err != nil {
		h.logger.Error("Failed to get profile", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	if !success {
		pkghttp.Error(c, pkghttp.NewBadRequest(message, nil))
		return
	}

	// Format date of birth to show only the date part
	var dateOfBirth string
	if profileData.DateOfBirth != "" {
		dateOfBirth = profileData.DateOfBirth
	}

	// Create an ordered map for the profile data
	profile := map[string]interface{}{
		"id":                      profileData.Id,
		"full_name":               profileData.FullName,
		"is_bride":                profileData.IsBride,
		"marital_status":          profileData.MaritalStatus,
		"profession":              profileData.Profession,
		"profession_type":         profileData.ProfessionType,
		"height_cm":               profileData.HeightCm,
		"date_of_birth":           dateOfBirth,
		"community":               profileData.Community,
		"home_district":           profileData.HomeDistrict,
		"last_login":              profileData.LastLogin.AsTime(),
		"phone":                   profileData.Phone,
		"physically_challenged":   profileData.PhysicallyChallenged,
		"highest_education_level": profileData.HighestEducationLevel,
		"created_at":              profileData.CreatedAt.AsTime(),
	}

	// Add profile picture URL if it exists
	if profileData.ProfilePictureUrl != "" {
		profile["profile_picture_url"] = profileData.ProfilePictureUrl
	}

	// Create a custom response struct to maintain order
	type OrderedProfile struct {
		ID                    uint64    `json:"id"`
		FullName              string    `json:"full_name"`
		IsBride               bool      `json:"is_bride"`
		MaritalStatus         string    `json:"marital_status"`
		Profession            string    `json:"profession"`
		ProfessionType        string    `json:"profession_type"`
		HeightCM              int32     `json:"height_cm"`
		DateOfBirth           string    `json:"date_of_birth,omitempty"`
		Community             string    `json:"community"`
		HomeDistrict          string    `json:"home_district"`
		LastLogin             time.Time `json:"last_login"`
		Phone                 string    `json:"phone"`
		PhysicallyChallenged  bool      `json:"physically_challenged"`
		HighestEducationLevel string    `json:"highest_education_level"`
		CreatedAt             time.Time `json:"created_at"`
		ProfilePictureURL     string    `json:"profile_picture_url,omitempty"`
	}

	orderedProfile := OrderedProfile{
		ID:                    profileData.Id,
		FullName:              profileData.FullName,
		IsBride:               profileData.IsBride,
		MaritalStatus:         profileData.MaritalStatus,
		Profession:            profileData.Profession,
		ProfessionType:        profileData.ProfessionType,
		HeightCM:              profileData.HeightCm,
		DateOfBirth:           dateOfBirth,
		Community:             profileData.Community,
		HomeDistrict:          profileData.HomeDistrict,
		LastLogin:             profileData.LastLogin.AsTime(),
		Phone:                 profileData.Phone,
		PhysicallyChallenged:  profileData.PhysicallyChallenged,
		HighestEducationLevel: profileData.HighestEducationLevel,
		CreatedAt:             profileData.CreatedAt.AsTime(),
		ProfilePictureURL:     profileData.ProfilePictureUrl,
	}

	pkghttp.Success(c, http.StatusOK, message, gin.H{
		"profile": orderedProfile,
	})
}
