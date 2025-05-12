package user

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/validation"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/middleware"
)

type PatchProfileRequest struct {
	IsBride               *bool      `json:"is_bride,omitempty"`
	FullName              *string    `json:"full_name,omitempty"`
	DateOfBirth           *time.Time `json:"date_of_birth,omitempty"`
	HeightCM              *int       `json:"height_cm,omitempty"`
	PhysicallyChallenged  *bool      `json:"physically_challenged,omitempty"`
	Community             *string    `json:"community,omitempty"`
	MaritalStatus         *string    `json:"marital_status,omitempty"`
	Profession            *string    `json:"profession,omitempty"`
	ProfessionType        *string    `json:"profession_type,omitempty"`
	HighestEducationLevel *string    `json:"highest_education_level,omitempty"`
	HomeDistrict          *string    `json:"home_district,omitempty"`
	ClearDateOfBirth      bool       `json:"clear_date_of_birth,omitempty"`
	ClearHeightCM         bool       `json:"clear_height_cm,omitempty"`
}

func (h *Handler) PatchProfile(c *gin.Context) {
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	var req PatchProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", "error", err)
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid request format", err))
		return
	}

	// Validate fields if present
	if req.HeightCM != nil {
		if err := validation.ValidateHeight(req.HeightCM); err != nil {
			pkghttp.Error(c, pkghttp.NewBadRequest(err.Error(), nil))
			return
		}
	}

	if req.Community != nil {
		if err := validation.ValidateCommunity(*req.Community); err != nil {
			pkghttp.Error(c, pkghttp.NewBadRequest(err.Error(), nil))
			return
		}
	}

	if req.MaritalStatus != nil {
		if err := validation.ValidateMaritalStatus(*req.MaritalStatus); err != nil {
			pkghttp.Error(c, pkghttp.NewBadRequest(err.Error(), nil))
			return
		}
	}

	if req.Profession != nil {
		if err := validation.ValidateProfession(*req.Profession); err != nil {
			pkghttp.Error(c, pkghttp.NewBadRequest(err.Error(), nil))
			return
		}
	}

	if req.ProfessionType != nil {
		if err := validation.ValidateProfessionType(*req.ProfessionType); err != nil {
			pkghttp.Error(c, pkghttp.NewBadRequest(err.Error(), nil))
			return
		}
	}

	if req.HighestEducationLevel != nil {
		if err := validation.ValidateEducationLevel(*req.HighestEducationLevel); err != nil {
			pkghttp.Error(c, pkghttp.NewBadRequest(err.Error(), nil))
			return
		}
	}

	if req.HomeDistrict != nil {
		if err := validation.ValidateHomeDistrict(*req.HomeDistrict); err != nil {
			pkghttp.Error(c, pkghttp.NewBadRequest(err.Error(), nil))
			return
		}
	}

	if req.DateOfBirth != nil {
		if err := validation.ValidateDateOfBirth(req.DateOfBirth); err != nil {
			pkghttp.Error(c, pkghttp.NewBadRequest(err.Error(), nil))
			return
		}
	}

	// Call the user service with user ID from context
	success, message, err := h.userClient.PatchProfile(
		c.Request.Context(),
		userID.(string),
		req.IsBride,
		req.FullName,
		req.DateOfBirth,
		req.HeightCM,
		req.PhysicallyChallenged,
		req.Community,
		req.MaritalStatus,
		req.Profession,
		req.ProfessionType,
		req.HighestEducationLevel,
		req.HomeDistrict,
		req.ClearDateOfBirth,
		req.ClearHeightCM,
	)

	if err != nil {
		h.logger.Error("Failed to patch profile", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	pkghttp.Success(c, http.StatusOK, message, gin.H{
		"success": success,
	})
}
