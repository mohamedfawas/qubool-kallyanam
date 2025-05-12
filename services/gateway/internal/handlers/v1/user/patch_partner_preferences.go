package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/validation"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/middleware"
)

type PatchPartnerPreferencesRequest struct {
	MinAgeYears                   *int     `json:"min_age_years,omitempty"`
	MaxAgeYears                   *int     `json:"max_age_years,omitempty"`
	MinHeightCM                   *int     `json:"min_height_cm,omitempty"`
	MaxHeightCM                   *int     `json:"max_height_cm,omitempty"`
	AcceptPhysicallyChallenged    *bool    `json:"accept_physically_challenged,omitempty"`
	PreferredCommunities          []string `json:"preferred_communities,omitempty"`
	PreferredMaritalStatus        []string `json:"preferred_marital_status,omitempty"`
	PreferredProfessions          []string `json:"preferred_professions,omitempty"`
	PreferredProfessionTypes      []string `json:"preferred_profession_types,omitempty"`
	PreferredEducationLevels      []string `json:"preferred_education_levels,omitempty"`
	PreferredHomeDistricts        []string `json:"preferred_home_districts,omitempty"`
	ClearPreferredCommunities     bool     `json:"clear_preferred_communities,omitempty"`
	ClearPreferredMaritalStatus   bool     `json:"clear_preferred_marital_status,omitempty"`
	ClearPreferredProfessions     bool     `json:"clear_preferred_professions,omitempty"`
	ClearPreferredProfessionTypes bool     `json:"clear_preferred_profession_types,omitempty"`
	ClearPreferredEducationLevels bool     `json:"clear_preferred_education_levels,omitempty"`
	ClearPreferredHomeDistricts   bool     `json:"clear_preferred_home_districts,omitempty"`
}

func (h *Handler) PatchPartnerPreferences(c *gin.Context) {
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	var req PatchPartnerPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", "error", err)
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid request format", err))
		return
	}

	// Validate if values are provided
	if req.MinAgeYears != nil || req.MaxAgeYears != nil {
		if err := validateAgeRange(req.MinAgeYears, req.MaxAgeYears); err != nil {
			pkghttp.Error(c, pkghttp.NewBadRequest(err.Error(), nil))
			return
		}
	}

	if req.MinHeightCM != nil || req.MaxHeightCM != nil {
		if err := validateHeightRange(req.MinHeightCM, req.MaxHeightCM); err != nil {
			pkghttp.Error(c, pkghttp.NewBadRequest(err.Error(), nil))
			return
		}
	}

	// Validate array elements if provided
	if len(req.PreferredCommunities) > 0 {
		for _, community := range req.PreferredCommunities {
			if err := validation.ValidateCommunity(community); err != nil {
				pkghttp.Error(c, pkghttp.NewBadRequest("Invalid community: "+community, err))
				return
			}
		}
	}

	if len(req.PreferredMaritalStatus) > 0 {
		for _, status := range req.PreferredMaritalStatus {
			if err := validation.ValidateMaritalStatus(status); err != nil {
				pkghttp.Error(c, pkghttp.NewBadRequest("Invalid marital status: "+status, err))
				return
			}
		}
	}

	if len(req.PreferredProfessions) > 0 {
		for _, profession := range req.PreferredProfessions {
			if err := validation.ValidateProfession(profession); err != nil {
				pkghttp.Error(c, pkghttp.NewBadRequest("Invalid profession: "+profession, err))
				return
			}
		}
	}

	if len(req.PreferredProfessionTypes) > 0 {
		for _, profType := range req.PreferredProfessionTypes {
			if err := validation.ValidateProfessionType(profType); err != nil {
				pkghttp.Error(c, pkghttp.NewBadRequest("Invalid profession type: "+profType, err))
				return
			}
		}
	}

	if len(req.PreferredEducationLevels) > 0 {
		for _, level := range req.PreferredEducationLevels {
			if err := validation.ValidateEducationLevel(level); err != nil {
				pkghttp.Error(c, pkghttp.NewBadRequest("Invalid education level: "+level, err))
				return
			}
		}
	}

	if len(req.PreferredHomeDistricts) > 0 {
		for _, district := range req.PreferredHomeDistricts {
			if err := validation.ValidateHomeDistrict(district); err != nil {
				pkghttp.Error(c, pkghttp.NewBadRequest("Invalid home district: "+district, err))
				return
			}
		}
	}

	success, message, err := h.userClient.PatchPartnerPreferences(
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
		req.ClearPreferredCommunities,
		req.ClearPreferredMaritalStatus,
		req.ClearPreferredProfessions,
		req.ClearPreferredProfessionTypes,
		req.ClearPreferredEducationLevels,
		req.ClearPreferredHomeDistricts,
	)

	if err != nil {
		h.logger.Error("Failed to patch partner preferences", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	pkghttp.Success(c, http.StatusOK, message, gin.H{
		"success": success,
	})
}
