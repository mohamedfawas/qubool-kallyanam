// gateway/internal/handlers/v1/user/user_profile.go
package user

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/validation"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/clients/user"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/middleware"
)

type Handler struct {
	userClient *user.Client
	logger     logging.Logger
}

func NewHandler(userClient *user.Client, logger logging.Logger) *Handler {
	return &Handler{
		userClient: userClient,
		logger:     logger,
	}
}

type UpdateProfileRequest struct {
	IsBride               bool       `json:"is_bride"`
	FullName              string     `json:"full_name"`
	DateOfBirth           *time.Time `json:"date_of_birth"`
	HeightCM              int        `json:"height_cm"`
	PhysicallyChallenged  bool       `json:"physically_challenged"`
	Community             string     `json:"community"`
	MaritalStatus         string     `json:"marital_status"`
	Profession            string     `json:"profession"`
	ProfessionType        string     `json:"profession_type"`
	HighestEducationLevel string     `json:"highest_education_level"`
	HomeDistrict          string     `json:"home_district"`
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	// Get user ID from context (previously set by auth middleware)
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	// Parse request body
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", "error", err)
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid request format", err))
		return
	}

	// Validate input fields
	if req.HeightCM != 0 {
		if err := validation.ValidateHeight(&req.HeightCM); err != nil {
			pkghttp.Error(c, pkghttp.NewBadRequest(err.Error(), nil))
			return
		}
	}

	if req.Community != "" {
		if err := validation.ValidateCommunity(req.Community); err != nil {
			pkghttp.Error(c, pkghttp.NewBadRequest(err.Error(), nil))
			return
		}
	}

	if req.MaritalStatus != "" {
		if err := validation.ValidateMaritalStatus(req.MaritalStatus); err != nil {
			pkghttp.Error(c, pkghttp.NewBadRequest(err.Error(), nil))
			return
		}
	}

	if req.Profession != "" {
		if err := validation.ValidateProfession(req.Profession); err != nil {
			pkghttp.Error(c, pkghttp.NewBadRequest(err.Error(), nil))
			return
		}
	}

	if req.ProfessionType != "" {
		if err := validation.ValidateProfessionType(req.ProfessionType); err != nil {
			pkghttp.Error(c, pkghttp.NewBadRequest(err.Error(), nil))
			return
		}
	}

	if req.HighestEducationLevel != "" {
		if err := validation.ValidateEducationLevel(req.HighestEducationLevel); err != nil {
			pkghttp.Error(c, pkghttp.NewBadRequest(err.Error(), nil))
			return
		}
	}

	if req.HomeDistrict != "" {
		if err := validation.ValidateHomeDistrict(req.HomeDistrict); err != nil {
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
	success, message, err := h.userClient.UpdateProfile(
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
	)

	if err != nil {
		h.logger.Error("Failed to update profile", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	pkghttp.Success(c, http.StatusOK, message, gin.H{
		"success": success,
	})
}
