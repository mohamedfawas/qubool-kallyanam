package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/constants"
	"gorm.io/gorm"
)

// MatchStatus represents the action a user takes on a potential match
type MatchStatus = constants.MatchStatus

// ProfileMatch represents a user's action on another profile
type ProfileMatch struct {
	ID        uint           `gorm:"primaryKey;autoIncrement"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null;index:idx_profile_matches_user_id"`
	TargetID  uuid.UUID      `gorm:"type:uuid;not null;index:idx_profile_matches_target_id"`
	Status    MatchStatus    `gorm:"type:match_status_enum;not null;index:idx_profile_matches_status"`
	CreatedAt time.Time      `gorm:"not null;default:now()"`
	UpdatedAt time.Time      `gorm:"not null;default:now()"`
	IsDeleted bool           `gorm:"not null;default:false"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName returns the database table name for ProfileMatch
func (ProfileMatch) TableName() string {
	return "profile_matches"
}

// MutualMatch represents when two users have liked each other
type MutualMatch struct {
	ID        uint           `gorm:"primaryKey;autoIncrement"`
	UserID1   uuid.UUID      `gorm:"type:uuid;not null;column:user_id_1;index:idx_mutual_matches_user_id_1"`
	UserID2   uuid.UUID      `gorm:"type:uuid;not null;column:user_id_2;index:idx_mutual_matches_user_id_2"`
	MatchedAt time.Time      `gorm:"not null;default:now()"`
	IsActive  bool           `gorm:"not null;default:true"`
	CreatedAt time.Time      `gorm:"not null;default:now()"`
	UpdatedAt time.Time      `gorm:"not null;default:now()"`
	IsDeleted bool           `gorm:"not null;default:false"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName returns the database table name for MutualMatch
func (MutualMatch) TableName() string {
	return "mutual_matches"
}

// RecommendedProfile represents a profile recommended to a user with match reasons
type RecommendedProfile struct {
	ID                    uint
	UserID                uuid.UUID
	FullName              string
	Age                   int
	HeightCM              *int
	PhysicallyChallenged  bool
	Community             Community
	MaritalStatus         MaritalStatus
	Profession            Profession
	ProfessionType        ProfessionType
	HighestEducationLevel EducationLevel
	HomeDistrict          HomeDistrict
	ProfilePictureURL     *string
	LastLogin             time.Time
	MatchReasons          []string
}

// PaginationData contains information for paginated results
type PaginationData struct {
	Total   int
	Limit   int
	Offset  int
	HasMore bool
}

// MatchHistoryItem represents a user's past match action with profile details
type MatchHistoryItem struct {
	ProfileID             uint
	FullName              string
	Age                   int
	HeightCM              *int
	PhysicallyChallenged  bool
	Community             Community
	MaritalStatus         MaritalStatus
	Profession            Profession
	ProfessionType        ProfessionType
	HighestEducationLevel EducationLevel
	HomeDistrict          HomeDistrict
	ProfilePictureURL     *string
	Action                MatchStatus
	ActionDate            time.Time
}

type MutualMatchData struct {
	ProfileID             uint
	FullName              string
	Age                   int
	HeightCM              *int
	PhysicallyChallenged  bool
	Community             Community
	MaritalStatus         MaritalStatus
	Profession            Profession
	ProfessionType        ProfessionType
	HighestEducationLevel EducationLevel
	HomeDistrict          HomeDistrict
	ProfilePictureURL     *string
	LastLogin             time.Time
	MatchedAt             time.Time
}
