package admin

import (
	"time"
)

// GetUsersRequest represents the REST API request for getting users list
type GetUsersRequest struct {
	Limit     int32  `form:"limit,default=20" binding:"min=1,max=100"`
	Offset    int32  `form:"offset,default=0" binding:"min=0"`
	SortBy    string `form:"sort_by,default=created_at" binding:"omitempty,oneof=created_at last_login_at email"`
	SortOrder string `form:"sort_order,default=desc" binding:"omitempty,oneof=asc desc"`

	// Boolean filters
	IsActive     *bool `form:"is_active"`
	VerifiedOnly *bool `form:"verified_only"`
	PremiumOnly  *bool `form:"premium_only"`

	// Date filters
	CreatedAfter    *time.Time `form:"created_after" time_format:"2006-01-02"`
	CreatedBefore   *time.Time `form:"created_before" time_format:"2006-01-02"`
	LastLoginAfter  *time.Time `form:"last_login_after" time_format:"2006-01-02"`
	LastLoginBefore *time.Time `form:"last_login_before" time_format:"2006-01-02"`
}

// GetUsersResponse represents the API response for getting users list
type GetUsersResponse struct {
	Success    bool                  `json:"success"`
	Message    string                `json:"message"`
	Users      []UserSummaryResponse `json:"users"`
	Pagination PaginationResponse    `json:"pagination"`
}

// UserSummaryResponse represents a user summary for listing
type UserSummaryResponse struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	Phone        string     `json:"phone"`
	Verified     bool       `json:"verified"`
	IsActive     bool       `json:"is_active"`
	IsPremium    bool       `json:"is_premium"`
	PremiumUntil *time.Time `json:"premium_until,omitempty"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// GetUserResponse represents the API response for getting single user details
type GetUserResponse struct {
	Success bool                `json:"success"`
	Message string              `json:"message"`
	User    UserDetailsResponse `json:"user"`
}

// UserDetailsResponse represents detailed user information
type UserDetailsResponse struct {
	Auth    AuthDataResponse             `json:"auth"`
	Profile *DetailedProfileDataResponse `json:"profile,omitempty"`
}

// AuthDataResponse represents auth service user data
type AuthDataResponse struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	Phone        string     `json:"phone"`
	Verified     bool       `json:"verified"`
	IsActive     bool       `json:"is_active"`
	IsPremium    bool       `json:"is_premium"`
	PremiumUntil *time.Time `json:"premium_until,omitempty"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// DetailedProfileDataResponse represents user profile data
type DetailedProfileDataResponse struct {
	ID                    uint64                      `json:"id"`
	IsBride               bool                        `json:"is_bride"`
	FullName              string                      `json:"full_name"`
	DateOfBirth           string                      `json:"date_of_birth"`
	HeightCm              int32                       `json:"height_cm"`
	PhysicallyChallenged  bool                        `json:"physically_challenged"`
	Community             string                      `json:"community"`
	MaritalStatus         string                      `json:"marital_status"`
	Profession            string                      `json:"profession"`
	ProfessionType        string                      `json:"profession_type"`
	HighestEducationLevel string                      `json:"highest_education_level"`
	HomeDistrict          string                      `json:"home_district"`
	ProfilePictureURL     string                      `json:"profile_picture_url"`
	LastLogin             *time.Time                  `json:"last_login,omitempty"`
	Age                   int32                       `json:"age"`
	PartnerPreferences    *PartnerPreferencesResponse `json:"partner_preferences,omitempty"`
	AdditionalPhotos      []UserPhotoResponse         `json:"additional_photos"`
	IntroVideo            *UserVideoResponse          `json:"intro_video,omitempty"`
}

// PaginationResponse represents pagination information
type PaginationResponse struct {
	Total       int32 `json:"total"`
	Limit       int32 `json:"limit"`
	Offset      int32 `json:"offset"`
	HasMore     bool  `json:"has_more"`
	TotalPages  int32 `json:"total_pages"`
	CurrentPage int32 `json:"current_page"`
}

// Additional response types for nested structures
type PartnerPreferencesResponse struct {
	MinAgeYears                int32    `json:"min_age_years"`
	MaxAgeYears                int32    `json:"max_age_years"`
	MinHeightCm                int32    `json:"min_height_cm"`
	MaxHeightCm                int32    `json:"max_height_cm"`
	AcceptPhysicallyChallenged bool     `json:"accept_physically_challenged"`
	PreferredCommunities       []string `json:"preferred_communities"`
	PreferredMaritalStatus     []string `json:"preferred_marital_status"`
	PreferredProfessions       []string `json:"preferred_professions"`
	PreferredProfessionTypes   []string `json:"preferred_profession_types"`
	PreferredEducationLevels   []string `json:"preferred_education_levels"`
	PreferredHomeDistricts     []string `json:"preferred_home_districts"`
}

type UserPhotoResponse struct {
	PhotoURL     string    `json:"photo_url"`
	DisplayOrder int32     `json:"display_order"`
	CreatedAt    time.Time `json:"created_at"`
}

type UserVideoResponse struct {
	VideoURL        string    `json:"video_url"`
	FileName        string    `json:"file_name"`
	FileSize        int64     `json:"file_size"`
	DurationSeconds int32     `json:"duration_seconds"`
	CreatedAt       time.Time `json:"created_at"`
}
