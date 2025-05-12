package user

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/validation"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/middleware"
)

// UpdatePartnerPreferencesRequest defines the request body for updating partner preferences
type UpdatePartnerPreferencesRequest struct {
	MinAgeYears                *int     `json:"min_age_years,omitempty"`
	MaxAgeYears                *int     `json:"max_age_years,omitempty"`
	MinHeightCM                *int     `json:"min_height_cm,omitempty"`
	MaxHeightCM                *int     `json:"max_height_cm,omitempty"`
	AcceptPhysicallyChallenged bool     `json:"accept_physically_challenged"`
	PreferredCommunities       []string `json:"preferred_communities"`
	PreferredMaritalStatus     []string `json:"preferred_marital_status"`
	PreferredProfessions       []string `json:"preferred_professions"`
	PreferredProfessionTypes   []string `json:"preferred_profession_types"`
	PreferredEducationLevels   []string `json:"preferred_education_levels"`
	PreferredHomeDistricts     []string `json:"preferred_home_districts"`
}

var (
	ErrInvalidAgeRange    = errors.New("invalid age range: min age must be at least 18, max age must be at most 80, and min age must not exceed max age")
	ErrInvalidHeightRange = errors.New("invalid height range: min height must be at least 130cm, max height must be at most 220cm, and min height must not exceed max height")
)

// validateAgeRange validates that the age range is valid
func validateAgeRange(minAge, maxAge *int) error {
	if minAge != nil && maxAge != nil {
		if *minAge < 18 || *maxAge > 80 || *minAge > *maxAge {
			return ErrInvalidAgeRange
		}
	} else if minAge != nil && *minAge < 18 {
		return ErrInvalidAgeRange
	} else if maxAge != nil && *maxAge > 80 {
		return ErrInvalidAgeRange
	}
	return nil
}

// validateHeightRange validates that the height range is valid
func validateHeightRange(minHeight, maxHeight *int) error {
	if minHeight != nil {
		if err := validation.ValidateHeight(minHeight); err != nil {
			return ErrInvalidHeightRange
		}
	}
	if maxHeight != nil {
		if err := validation.ValidateHeight(maxHeight); err != nil {
			return ErrInvalidHeightRange
		}
	}
	if minHeight != nil && maxHeight != nil && *minHeight > *maxHeight {
		return ErrInvalidHeightRange
	}
	return nil
}

// UpdatePartnerPreferences handles the partner preferences update request
func (h *Handler) UpdatePartnerPreferences(c *gin.Context) {
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	var req UpdatePartnerPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", "error", err)
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid request format", err))
		return
	}

	// Validate age and height ranges
	if err := validateAgeRange(req.MinAgeYears, req.MaxAgeYears); err != nil {
		pkghttp.Error(c, pkghttp.NewBadRequest(err.Error(), nil))
		return
	}

	if err := validateHeightRange(req.MinHeightCM, req.MaxHeightCM); err != nil {
		pkghttp.Error(c, pkghttp.NewBadRequest(err.Error(), nil))
		return
	}

	// Validate preferred communities
	for _, community := range req.PreferredCommunities {
		if err := validation.ValidateCommunity(community); err != nil {
			pkghttp.Error(c, pkghttp.NewBadRequest("Invalid community: "+community, err))
			return
		}
	}

	// Validate preferred marital statuses
	for _, status := range req.PreferredMaritalStatus {
		if err := validation.ValidateMaritalStatus(status); err != nil {
			pkghttp.Error(c, pkghttp.NewBadRequest("Invalid marital status: "+status, err))
			return
		}
	}

	// Validate preferred professions
	for _, profession := range req.PreferredProfessions {
		if err := validation.ValidateProfession(profession); err != nil {
			pkghttp.Error(c, pkghttp.NewBadRequest("Invalid profession: "+profession, err))
			return
		}
	}

	// Validate preferred profession types
	for _, profType := range req.PreferredProfessionTypes {
		if err := validation.ValidateProfessionType(profType); err != nil {
			pkghttp.Error(c, pkghttp.NewBadRequest("Invalid profession type: "+profType, err))
			return
		}
	}

	// Validate preferred education levels
	for _, level := range req.PreferredEducationLevels {
		if err := validation.ValidateEducationLevel(level); err != nil {
			pkghttp.Error(c, pkghttp.NewBadRequest("Invalid education level: "+level, err))
			return
		}
	}

	// Validate preferred home districts
	for _, district := range req.PreferredHomeDistricts {
		if err := validation.ValidateHomeDistrict(district); err != nil {
			pkghttp.Error(c, pkghttp.NewBadRequest("Invalid home district: "+district, err))
			return
		}
	}

	// Initialize empty arrays if they're nil to avoid null in JSON
	if req.PreferredCommunities == nil {
		req.PreferredCommunities = []string{}
	}
	if req.PreferredMaritalStatus == nil {
		req.PreferredMaritalStatus = []string{}
	}
	if req.PreferredProfessions == nil {
		req.PreferredProfessions = []string{}
	}
	if req.PreferredProfessionTypes == nil {
		req.PreferredProfessionTypes = []string{}
	}
	if req.PreferredEducationLevels == nil {
		req.PreferredEducationLevels = []string{}
	}
	if req.PreferredHomeDistricts == nil {
		req.PreferredHomeDistricts = []string{}
	}

	// Call the user service
	success, message, err := h.userClient.UpdatePartnerPreferences(
		c.Request.Context(),
		userID.(string),
		req.MinAgeYears,
		req.MaxAgeYears,
		req.MinHeightCM,
		req.MaxHeightCM,
		req.AcceptPhysicallyChallenged,
		req.PreferredCommunities,
		req.PreferredMaritalStatus,
		req.PreferredProfessions,
		req.PreferredProfessionTypes,
		req.PreferredEducationLevels,
		req.PreferredHomeDistricts,
	)

	if err != nil {
		h.logger.Error("Failed to update partner preferences", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	pkghttp.Success(c, http.StatusOK, message, gin.H{
		"success": success,
	})
}
