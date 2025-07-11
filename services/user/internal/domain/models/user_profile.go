package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/constants"
	"gorm.io/gorm"
)

// Type aliases to use constants from constants package
type Community = constants.Community
type MaritalStatus = constants.MaritalStatus
type Profession = constants.Profession
type ProfessionType = constants.ProfessionType
type EducationLevel = constants.EducationLevel
type HomeDistrict = constants.HomeDistrict

// UserProfile represents the user_profiles table in PostgreSQL
// Note: ID is BIGSERIAL (auto-increment), so we use uint
// DeletedAt is managed by GORM for soft deletes
// gorm.Model includes ID, CreatedAt, UpdatedAt, DeletedAt but ID is uint by default

// We define our own struct to customize types

// UserProfile is the GORM model for user_profiles
// nolint:structcheck
type UserProfile struct {
	ID                    uint           `gorm:"primaryKey;autoIncrement"`
	UserID                uuid.UUID      `gorm:"type:uuid;not null;index:idx_user_profiles_user_id"`
	IsBride               bool           `gorm:"not null;default:false;column:is_bride"`
	FullName              string         `gorm:"size:200;column:full_name"`
	Phone                 string         `gorm:"size:20;column:phone"`
	Email                 string         `gorm:"size:255;column:email"`
	DateOfBirth           *time.Time     `gorm:"type:date;column:date_of_birth"`
	HeightCM              *int           `gorm:"column:height_cm;check:height_cm>=130 AND height_cm<=220"`
	PhysicallyChallenged  bool           `gorm:"not null;default:false;column:physically_challenged"`
	Community             Community      `gorm:"type:community_enum"`
	MaritalStatus         MaritalStatus  `gorm:"type:marital_status_enum;column:marital_status"`
	Profession            Profession     `gorm:"type:profession_enum"`
	ProfessionType        ProfessionType `gorm:"type:profession_type_enum;column:profession_type"`
	HighestEducationLevel EducationLevel `gorm:"type:education_level_enum;column:highest_education_level"`
	HomeDistrict          HomeDistrict   `gorm:"type:home_district_enum;column:home_district"`
	ProfilePictureURL     *string        `gorm:"size:255;column:profile_picture_url"`
	LastLogin             time.Time      `gorm:"not null;default:now();column:last_login"`
	CreatedAt             time.Time      `gorm:"not null;default:now();column:created_at"`
	UpdatedAt             time.Time      `gorm:"not null;default:now();column:updated_at"`
	IsDeleted             bool           `gorm:"not null;default:false;column:is_deleted"`
	DeletedAt             gorm.DeletedAt `gorm:"index;column:deleted_at"`
}

type PartnerPreferences struct {
	ID            uint `gorm:"primaryKey;autoIncrement;column:id"`
	UserProfileID uint `gorm:"not null;index:idx_partner_preferences_user_profile_id;column:user_profile_id"`

	MinAgeYears *int `gorm:"column:min_age_years;check:min_age_years>=18 AND min_age_years<=80"`
	MaxAgeYears *int `gorm:"column:max_age_years;check:max_age_years>=18 AND max_age_years<=80"`

	MinHeightCM *int `gorm:"column:min_height_cm;check:min_height_cm>=130 AND min_height_cm<=220"`
	MaxHeightCM *int `gorm:"column:max_height_cm;check:max_height_cm>=130 AND max_height_cm<=220"`

	AcceptPhysicallyChallenged bool `gorm:"not null;default:true;column:accept_physically_challenged"`

	PreferredCommunities     []Community      `gorm:"type:community_enum[];column:preferred_communities"`
	PreferredMaritalStatus   []MaritalStatus  `gorm:"type:marital_status_enum[];column:preferred_marital_status"`
	PreferredProfessions     []Profession     `gorm:"type:profession_enum[];column:preferred_professions"`
	PreferredProfessionTypes []ProfessionType `gorm:"type:profession_type_enum[];column:preferred_profession_types"`
	PreferredEducationLevels []EducationLevel `gorm:"type:education_level_enum[];column:preferred_education_levels"`
	PreferredHomeDistricts   []HomeDistrict   `gorm:"type:home_district_enum[];column:preferred_home_districts"`

	CreatedAt time.Time      `gorm:"not null;default:now();column:created_at"`
	UpdatedAt time.Time      `gorm:"not null;default:now();column:updated_at"`
	IsDeleted bool           `gorm:"not null;default:false;column:is_deleted"`
	DeletedAt gorm.DeletedAt `gorm:"index;column:deleted_at"`
}

// PartnerPreferencesWithArrays is used for properly handling PostgreSQL arrays in GORM
type PartnerPreferencesWithArrays struct {
	ID                            uint             `gorm:"primaryKey;autoIncrement;column:id"`
	UserProfileID                 uint             `gorm:"not null;index:idx_partner_preferences_user_profile_id;column:user_profile_id"`
	MinAgeYears                   *int             `gorm:"column:min_age_years;check:min_age_years>=18 AND min_age_years<=80"`
	MaxAgeYears                   *int             `gorm:"column:max_age_years;check:max_age_years>=18 AND max_age_years<=80"`
	MinHeightCM                   *int             `gorm:"column:min_height_cm;check:min_height_cm>=130 AND min_height_cm<=220"`
	MaxHeightCM                   *int             `gorm:"column:max_height_cm;check:max_height_cm>=130 AND max_height_cm<=220"`
	AcceptPhysicallyChallenged    bool             `gorm:"not null;default:true;column:accept_physically_challenged"`
	PreferredCommunities          []Community      `gorm:"-"`
	PreferredMaritalStatus        []MaritalStatus  `gorm:"-"`
	PreferredProfessions          []Profession     `gorm:"-"`
	PreferredProfessionTypes      []ProfessionType `gorm:"-"`
	PreferredEducationLevels      []EducationLevel `gorm:"-"`
	PreferredHomeDistricts        []HomeDistrict   `gorm:"-"`
	PreferredCommunitiesArray     pq.StringArray   `gorm:"type:community_enum[];column:preferred_communities"`
	PreferredMaritalStatusArray   pq.StringArray   `gorm:"type:marital_status_enum[];column:preferred_marital_status"`
	PreferredProfessionsArray     pq.StringArray   `gorm:"type:profession_enum[];column:preferred_professions"`
	PreferredProfessionTypesArray pq.StringArray   `gorm:"type:profession_type_enum[];column:preferred_profession_types"`
	PreferredEducationLevelsArray pq.StringArray   `gorm:"type:education_level_enum[];column:preferred_education_levels"`
	PreferredHomeDistrictsArray   pq.StringArray   `gorm:"type:home_district_enum[];column:preferred_home_districts"`
	CreatedAt                     time.Time        `gorm:"not null;default:now();column:created_at"`
	UpdatedAt                     time.Time        `gorm:"not null;default:now();column:updated_at"`
	IsDeleted                     bool             `gorm:"not null;default:false;column:is_deleted"`
	DeletedAt                     gorm.DeletedAt   `gorm:"index;column:deleted_at"`
}

func (PartnerPreferencesWithArrays) TableName() string {
	return "partner_preferences"
}

// func (p *PartnerPreferencesWithArrays) BeforeCreate(*gorm.DB) error {
// 	// Convert the typed arrays to string arrays for pq
// 	p.PreferredCommunitiesArray = make(pq.StringArray, len(p.PreferredCommunities))
// 	for i, v := range p.PreferredCommunities {
// 		p.PreferredCommunitiesArray[i] = string(v)
// 	}

// 	p.PreferredMaritalStatusArray = make(pq.StringArray, len(p.PreferredMaritalStatus))
// 	for i, v := range p.PreferredMaritalStatus {
// 		p.PreferredMaritalStatusArray[i] = string(v)
// 	}

// 	p.PreferredProfessionsArray = make(pq.StringArray, len(p.PreferredProfessions))
// 	for i, v := range p.PreferredProfessions {
// 		p.PreferredProfessionsArray[i] = string(v)
// 	}

// 	p.PreferredProfessionTypesArray = make(pq.StringArray, len(p.PreferredProfessionTypes))
// 	for i, v := range p.PreferredProfessionTypes {
// 		p.PreferredProfessionTypesArray[i] = string(v)
// 	}

// 	p.PreferredEducationLevelsArray = make(pq.StringArray, len(p.PreferredEducationLevels))
// 	for i, v := range p.PreferredEducationLevels {
// 		p.PreferredEducationLevelsArray[i] = string(v)
// 	}

// 	p.PreferredHomeDistrictsArray = make(pq.StringArray, len(p.PreferredHomeDistricts))
// 	for i, v := range p.PreferredHomeDistricts {
// 		p.PreferredHomeDistrictsArray[i] = string(v)
// 	}

// 	return nil
// }
