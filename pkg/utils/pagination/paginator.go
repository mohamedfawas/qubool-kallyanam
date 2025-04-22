package pagination

import (
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// PageRequest represents pagination parameters from client
type PageRequest struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"page_size" form:"page_size"`
	// For cursor-based pagination
	Cursor string `json:"cursor" form:"cursor"`
}

// PageResponse represents a paginated response
type PageResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalItems int64       `json:"total_items"`
	TotalPages int         `json:"total_pages"`
	// For cursor-based pagination
	NextCursor string `json:"next_cursor,omitempty"`
	PrevCursor string `json:"prev_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// DefaultPageSize is the default number of items per page
const DefaultPageSize = 20

// MaxPageSize is the maximum allowed page size
const MaxPageSize = 100

// NormalizePageRequest ensures pagination parameters are within bounds
func NormalizePageRequest(req *PageRequest) *PageRequest {
	if req == nil {
		return &PageRequest{
			Page:     1,
			PageSize: DefaultPageSize,
		}
	}

	// Ensure page number is at least 1
	if req.Page < 1 {
		req.Page = 1
	}

	// Enforce page size limits
	if req.PageSize < 1 {
		req.PageSize = DefaultPageSize
	} else if req.PageSize > MaxPageSize {
		req.PageSize = MaxPageSize
	}

	return req
}

// CalculateOffset calculates the offset for SQL queries
func CalculateOffset(page, pageSize int) int {
	return (page - 1) * pageSize
}

// CalculateTotalPages calculates the total number of pages
func CalculateTotalPages(totalItems int64, pageSize int) int {
	return int(math.Ceil(float64(totalItems) / float64(pageSize)))
}

// NewPageResponse creates a new paginated response
func NewPageResponse(data interface{}, req *PageRequest, totalItems int64) *PageResponse {
	req = NormalizePageRequest(req)
	totalPages := CalculateTotalPages(totalItems, req.PageSize)

	return &PageResponse{
		Data:       data,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasMore:    req.Page < totalPages,
	}
}

// Cursor-based pagination

// CursorOptions defines options for cursor-based pagination
type CursorOptions struct {
	// ID of the last item in the current page
	LastID string
	// For forward/backward navigation
	Direction string
	// Number of items to fetch
	Limit int
}

// EncodeCursor creates a cursor string from CursorOptions
func EncodeCursor(opts CursorOptions) string {
	if opts.LastID == "" {
		return ""
	}

	// Format: direction:lastID:limit
	data := fmt.Sprintf("%s:%s:%d", opts.Direction, opts.LastID, opts.Limit)
	return base64.StdEncoding.EncodeToString([]byte(data))
}

// DecodeCursor decodes a cursor string into CursorOptions
func DecodeCursor(cursor string) (CursorOptions, error) {
	if cursor == "" {
		return CursorOptions{Direction: "next", Limit: DefaultPageSize}, nil
	}

	data, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return CursorOptions{}, fmt.Errorf("invalid cursor encoding: %w", err)
	}

	parts := strings.Split(string(data), ":")
	if len(parts) != 3 {
		return CursorOptions{}, fmt.Errorf("invalid cursor format")
	}

	limit, err := strconv.Atoi(parts[2])
	if err != nil {
		return CursorOptions{}, fmt.Errorf("invalid limit in cursor: %w", err)
	}

	// Enforce limit bounds
	if limit < 1 {
		limit = DefaultPageSize
	} else if limit > MaxPageSize {
		limit = MaxPageSize
	}

	return CursorOptions{
		Direction: parts[0],
		LastID:    parts[1],
		Limit:     limit,
	}, nil
}

// NewCursorPageResponse creates a response for cursor-based pagination
func NewCursorPageResponse(data interface{}, items []interface{}, cursor string, hasMore bool) *PageResponse {
	opts, _ := DecodeCursor(cursor)

	resp := &PageResponse{
		Data:     data,
		PageSize: opts.Limit,
		HasMore:  hasMore,
	}

	if hasMore && len(items) > 0 {
		// Get the ID of the last item for the next cursor
		lastItem := items[len(items)-1]

		// This assumes lastItem has an ID field - you may need to adjust
		// based on your actual data structure
		var lastID string
		switch v := lastItem.(type) {
		case map[string]interface{}:
			if id, ok := v["id"].(string); ok {
				lastID = id
			}
		case struct{ ID string }:
			lastID = v.ID
		}

		if lastID != "" {
			nextOpts := CursorOptions{
				Direction: "next",
				LastID:    lastID,
				Limit:     opts.Limit,
			}
			resp.NextCursor = EncodeCursor(nextOpts)
		}
	}

	return resp
}
