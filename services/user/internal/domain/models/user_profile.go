package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const DefaultNotMentioned = "not_mentioned"

// Enum type definitions matching PostgreSQL enums
// Community defines the community_enum type
type Community string

const (
	CommunitySunni        Community = "sunni"
	CommunityMujahid      Community = "mujahid"
	CommunityTabligh      Community = "tabligh"
	CommunityJamateIslami Community = "jamate_islami"
	CommunityShia         Community = "shia"
	CommunityMuslim       Community = "muslim"
)

// MaritalStatus defines the marital_status_enum type
type MaritalStatus string

const (
	MaritalNeverMarried  MaritalStatus = "never_married"
	MaritalDivorced      MaritalStatus = "divorced"
	MaritalNikkahDivorce MaritalStatus = "nikkah_divorce"
	MaritalWidowed       MaritalStatus = "widowed"
)

// Profession defines the profession_enum type
type Profession string

const (
	ProfessionStudent  Profession = "student"
	ProfessionDoctor   Profession = "doctor"
	ProfessionEngineer Profession = "engineer"
	ProfessionFarmer   Profession = "farmer"
	ProfessionTeacher  Profession = "teacher"
)

// ProfessionType defines the profession_type_enum type
type ProfessionType string

const (
	ProfessionTypeFullTime     ProfessionType = "full_time"
	ProfessionTypePartTime     ProfessionType = "part_time"
	ProfessionTypeFreelance    ProfessionType = "freelance"
	ProfessionTypeSelfEmployed ProfessionType = "self_employed"
	ProfessionTypeNotWorking   ProfessionType = "not_working"
)

// EducationLevel defines the education_level_enum type
type EducationLevel string

const (
	EducationLessThanHighSchool EducationLevel = "less_than_high_school"
	EducationHighSchool         EducationLevel = "high_school"
	EducationHigherSecondary    EducationLevel = "higher_secondary"
	EducationUnderGraduation    EducationLevel = "under_graduation"
	EducationPostGraduation     EducationLevel = "post_graduation"
)

// HomeDistrict defines the home_district_enum type
type HomeDistrict string

const (
	DistrictThiruvananthapuram HomeDistrict = "thiruvananthapuram"
	DistrictKollam             HomeDistrict = "kollam"
	DistrictPathanamthitta     HomeDistrict = "pathanamthitta"
	DistrictAlappuzha          HomeDistrict = "alappuzha"
	DistrictKottayam           HomeDistrict = "kottayam"
	DistrictErnakulam          HomeDistrict = "ernakulam"
	DistrictThrissur           HomeDistrict = "thrissur"
	DistrictPalakkad           HomeDistrict = "palakkad"
	DistrictMalappuram         HomeDistrict = "malappuram"
	DistrictKozhikode          HomeDistrict = "kozhikode"
	DistrictWayanad            HomeDistrict = "wayanad"
	DistrictKannur             HomeDistrict = "kannur"
	DistrictKasaragod          HomeDistrict = "kasaragod"
	DistrictIdukki             HomeDistrict = "idukki"
)

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
	DeletedAt             gorm.DeletedAt `gorm:"index;column:deleted_at"`
}
