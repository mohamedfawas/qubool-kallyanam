package auth

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
)

type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required"`
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid delete account request body", "error", err)
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid request format", err))
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		h.logger.Error("User ID not found in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	// Create a new context with user-id value
	ctx := context.WithValue(c.Request.Context(), "user-id", userID.(string))

	success, message, err := h.authClient.Delete(ctx, req.Password)
	if err != nil {
		h.logger.Error("Delete account failed", "error", err, "userID", userID)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	response := map[string]interface{}{
		"success": success,
		"message": message,
	}

	pkghttp.Success(c, http.StatusOK, message, response)
}
