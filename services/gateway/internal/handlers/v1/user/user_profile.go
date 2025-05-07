package user

// import (
// 	"net/http"

// 	"github.com/gin-gonic/gin"

// 	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
// 	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
// 	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/clients/user"
// )

// // Handler handles user-related HTTP requests
// type Handler struct {
// 	userClient *user.Client
// 	logger     logging.Logger
// }

// // NewHandler creates a new user handler
// func NewHandler(userClient *user.Client, logger logging.Logger) *Handler {
// 	return &Handler{
// 		userClient: userClient,
// 		logger:     logger,
// 	}
// }

// // CreateProfileRequest defines the request body for profile creation
// type CreateProfileRequest struct {
// 	Name          string    `json:"name" binding:"required"`
// 	Age           int32     `json:"age" binding:"required,min=18,max=100"`
// 	Gender        string    `json:"gender" binding:"required"`
// 	Religion      string    `json:"religion"`
// 	Caste         string    `json:"caste"`
// 	MotherTongue  string    `json:"mother_tongue"`
// 	MaritalStatus string    `json:"marital_status" binding:"required"`
// 	HeightCm      int32     `json:"height_cm"`
// 	Education     string    `json:"education"`
// 	Occupation    string    `json:"occupation"`
// 	AnnualIncome  int64     `json:"annual_income"`
// 	Location      Location  `json:"location" binding:"required"`
// 	About         string    `json:"about"`
// }

// type Location struct {
// 	City    string `json:"city"`
// 	State   string `json:"state"`
// 	Country string `json:"country" binding:"required"`
// }

// // CreateProfileResponse defines the response for profile creation
// type CreateProfileResponse struct {
// 	Success bool   `json:"success"`
// 	ID      string `json:"id"`
// 	Message string `json:"message"`
// }

// // CreateProfile handles profile creation
// func (h *Handler) CreateProfile(c *gin.Context) {
// 	var req CreateProfileRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		h.logger.Error("Invalid profile request body", "error", err)
// 		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid request format", err))
// 		return
// 	}

// 	// Call user service with data from request
// 	success, id, message, err := h.userClient.CreateProfile(
// 		c.Request.Context(),
// 		req.Name,
// 		req.Age,
// 		req.Gender,
// 		req.Religion,
// 		req.Caste,
// 		req.MotherTongue,
// 		req.MaritalStatus,
// 		req.HeightCm,
// 		req.Education,
// 		req.Occupation,
// 		req.AnnualIncome,
// 		req.Location.City,
// 		req.Location.State,
// 		req.Location.Country,
// 		req.About,
// 	)
// 	if err != nil {
// 		h.logger.Error("Profile creation failed", "error", err)
// 		pkghttp.Error(c, pkghttp.FromGRPCError(err))
// 		return
// 	}

// 	response := CreateProfileResponse{
// 		Success: success,
// 		ID:      id,
// 		Message: message,
// 	}

// 	// Return success response with 201 Created status
// 	pkghttp.Success(c, http.StatusCreated, message, response)
// }
