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

// UploadUserVideo handles introduction video upload endpoint
func (h *Handler) UploadUserVideo(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	// Get the uploaded file
	file, err := c.FormFile("video")
	if err != nil {
		h.logger.Error("Failed to get video file from form", "error", err)
		pkghttp.Error(c, pkghttp.NewBadRequest("Failed to get video file from form", err))
		return
	}

	// Check file size (50MB limit for 1-minute video)
	if file.Size > 50*1024*1024 {
		h.logger.Debug("Video file too large", "size", file.Size)
		pkghttp.Error(c, pkghttp.NewBadRequest("Video file size exceeds the maximum allowed size of 50MB", nil))
		return
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	validExt := false
	for _, allowedExt := range []string{".mp4", ".mov", ".avi", ".mkv"} {
		if ext == allowedExt {
			validExt = true
			break
		}
	}

	if !validExt {
		h.logger.Debug("Invalid video file type", "extension", ext)
		pkghttp.Error(c, pkghttp.NewBadRequest("Unsupported video file type. Allowed types: mp4, mov, avi, mkv", nil))
		return
	}

	// Open the file
	src, err := file.Open()
	if err != nil {
		h.logger.Error("Failed to open video file", "error", err)
		pkghttp.Error(c, pkghttp.NewBadRequest("Failed to open video file", err))
		return
	}
	defer src.Close()

	// Read the file content
	fileBytes, err := io.ReadAll(src)
	if err != nil {
		h.logger.Error("Failed to read video file", "error", err)
		pkghttp.Error(c, pkghttp.NewBadRequest("Failed to read video file", err))
		return
	}

	// Detect content type
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(fileBytes)
	}

	// Call the user service
	success, message, videoURL, err := h.userClient.UploadUserVideo(
		c.Request.Context(),
		userID.(string),
		fileBytes,
		file.Filename,
		contentType,
	)

	if err != nil {
		h.logger.Error("Failed to upload user video", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	pkghttp.Success(c, http.StatusOK, message, gin.H{
		"success":   success,
		"video_url": videoURL,
	})
}

// GetUserVideo handles retrieving user video
func (h *Handler) GetUserVideo(c *gin.Context) {
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	// Call the user service
	success, message, video, err := h.userClient.GetUserVideo(
		c.Request.Context(),
		userID.(string),
	)

	if err != nil {
		h.logger.Error("Failed to get user video", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	// Convert to response format
	var videoData gin.H
	if video != nil {
		videoData = gin.H{
			"video_url":        video.VideoUrl,
			"file_name":        video.FileName,
			"file_size":        video.FileSize,
			"duration_seconds": video.DurationSeconds,
			"created_at":       video.CreatedAt.AsTime(),
		}
	}

	pkghttp.Success(c, http.StatusOK, message, gin.H{
		"success": success,
		"video":   videoData,
	})
}

// DeleteUserVideo handles deleting user video
func (h *Handler) DeleteUserVideo(c *gin.Context) {
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	// Call the user service
	success, message, err := h.userClient.DeleteUserVideo(
		c.Request.Context(),
		userID.(string),
	)

	if err != nil {
		h.logger.Error("Failed to delete user video", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	pkghttp.Success(c, http.StatusOK, message, gin.H{
		"success": success,
	})
}
