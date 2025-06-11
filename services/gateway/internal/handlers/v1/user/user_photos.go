package user

import (
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/middleware"
)

// UploadUserPhoto handles additional photo upload endpoint
func (h *Handler) UploadUserPhoto(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	// Get display order from form
	displayOrderStr := c.PostForm("display_order")
	if displayOrderStr == "" {
		h.logger.Debug("Missing display order")
		pkghttp.Error(c, pkghttp.NewBadRequest("Display order is required", nil))
		return
	}

	displayOrder, err := strconv.Atoi(displayOrderStr)
	if err != nil || displayOrder < 1 || displayOrder > 3 {
		h.logger.Debug("Invalid display order", "displayOrder", displayOrderStr)
		pkghttp.Error(c, pkghttp.NewBadRequest("Display order must be 1, 2, or 3", nil))
		return
	}

	// Get the uploaded file
	file, err := c.FormFile("photo")
	if err != nil {
		h.logger.Error("Failed to get file from form", "error", err)
		pkghttp.Error(c, pkghttp.NewBadRequest("Failed to get file from form", err))
		return
	}

	// Check file size (5MB limit)
	if file.Size > 5*1024*1024 {
		h.logger.Debug("File too large", "size", file.Size)
		pkghttp.Error(c, pkghttp.NewBadRequest("File size exceeds the maximum allowed size of 5MB", nil))
		return
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	validExt := false
	for _, allowedExt := range []string{".jpg", ".jpeg", ".png"} {
		if ext == allowedExt {
			validExt = true
			break
		}
	}

	if !validExt {
		h.logger.Debug("Invalid file type", "extension", ext)
		pkghttp.Error(c, pkghttp.NewBadRequest("Unsupported file type. Allowed types: jpg, jpeg, png", nil))
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

	// Detect content type if not provided
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(fileBytes)
	}

	// Call the user service
	success, message, photoURL, err := h.userClient.UploadUserPhoto(
		c.Request.Context(),
		userID.(string),
		fileBytes,
		file.Filename,
		contentType,
		displayOrder,
	)

	if err != nil {
		h.logger.Error("Failed to upload user photo", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	pkghttp.Success(c, http.StatusOK, message, gin.H{
		"success":       success,
		"photo_url":     photoURL,
		"display_order": displayOrder,
	})
}

// GetUserPhotos handles retrieving all user photos
func (h *Handler) GetUserPhotos(c *gin.Context) {
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	// Call the user service
	success, message, photos, err := h.userClient.GetUserPhotos(
		c.Request.Context(),
		userID.(string),
	)

	if err != nil {
		h.logger.Error("Failed to get user photos", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	// Convert to response format
	photoList := make([]gin.H, len(photos))
	for i, photo := range photos {
		photoList[i] = gin.H{
			"photo_url":     photo.PhotoUrl,
			"display_order": photo.DisplayOrder,
			"created_at":    photo.CreatedAt.AsTime(),
		}
	}

	pkghttp.Success(c, http.StatusOK, message, gin.H{
		"success": success,
		"photos":  photoList,
	})
}

// DeleteUserPhoto handles deleting a specific user photo
func (h *Handler) DeleteUserPhoto(c *gin.Context) {
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	// Get display order from URL parameter
	displayOrderStr := c.Param("order")
	displayOrder, err := strconv.Atoi(displayOrderStr)
	if err != nil || displayOrder < 1 || displayOrder > 3 {
		h.logger.Debug("Invalid display order", "displayOrder", displayOrderStr)
		pkghttp.Error(c, pkghttp.NewBadRequest("Display order must be 1, 2, or 3", nil))
		return
	}

	// Call the user service
	success, message, err := h.userClient.DeleteUserPhoto(
		c.Request.Context(),
		userID.(string),
		displayOrder,
	)

	if err != nil {
		h.logger.Error("Failed to delete user photo", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	pkghttp.Success(c, http.StatusOK, message, gin.H{
		"success":       success,
		"display_order": displayOrder,
	})
}
