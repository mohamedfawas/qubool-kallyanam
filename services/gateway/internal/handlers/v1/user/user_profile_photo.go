package user

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/middleware"
)

// UploadProfilePhoto handles the profile photo upload endpoint
func (h *Handler) UploadProfilePhoto(c *gin.Context) {
	// Get user ID from context (previously set by auth middleware)
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	// Get the uploaded file
	file, err := c.FormFile("photo")
	if err != nil {
		h.logger.Error("Failed to get file from form", "error", err)
		pkghttp.Error(c, pkghttp.NewBadRequest("Failed to get file from form", err))
		return
	}

	// Check file size
	if file.Size > 5*1024*1024 { // 5MB
		h.logger.Debug("File too large", "size", file.Size)
		pkghttp.Error(c, pkghttp.NewBadRequest("File size exceeds the maximum allowed size of 5MB", nil))
		return
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	validExt := false
	for _, allowedExt := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp"} {
		if ext == allowedExt {
			validExt = true
			break
		}
	}

	if !validExt {
		h.logger.Debug("Invalid file type", "extension", ext)
		pkghttp.Error(c, pkghttp.NewBadRequest("Unsupported file type. Allowed types: jpg, jpeg, png, gif, webp", nil))
		return
	}

	// Open the file
	src, err := file.Open()
	if err != nil {
		h.logger.Error("Failed to open file", "error", err)
		pkghttp.Error(c, pkghttp.NewBadRequest("Failed to open file", err))
		return
	}
	defer src.Close()

	// Read the file content
	fileBytes, err := io.ReadAll(src)
	if err != nil {
		h.logger.Error("Failed to read file", "error", err)
		pkghttp.Error(c, pkghttp.NewBadRequest("Failed to read file", err))
		return
	}

	// Detect content type if not provided in the file
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(fileBytes)
	}

	// Call the user service with user ID from context
	success, message, photoURL, err := h.userClient.UploadProfilePhoto(
		c.Request.Context(),
		userID.(string),
		fileBytes,
		file.Filename,
		contentType,
	)

	if err != nil {
		h.logger.Error("Failed to upload profile photo", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	pkghttp.Success(c, http.StatusOK, message, gin.H{
		"success":   success,
		"photo_url": photoURL,
	})
}

func (h *Handler) DeleteProfilePhoto(c *gin.Context) {
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	// Call the user service with user ID from context
	success, message, err := h.userClient.DeleteProfilePhoto(
		c.Request.Context(),
		userID.(string),
	)

	if err != nil {
		h.logger.Error("Failed to delete profile photo", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	pkghttp.Success(c, http.StatusOK, message, gin.H{
		"success": success,
	})
}
