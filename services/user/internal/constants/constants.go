package constants

// File size limits
const (
	MaxFileSize      = 5 * 1024 * 1024  // 5MB - general file size limit
	MaxImageFileSize = 5 * 1024 * 1024  // 5MB - specific for images
	MaxVideoFileSize = 50 * 1024 * 1024 // 50MB - specific for videos
)

// File type constants
var (
	// AllowedImageExtensions defines supported image file formats
	AllowedImageExtensions = []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}

	// AllowedVideoExtensions defines supported video file formats
	AllowedVideoExtensions = []string{".mp4", ".mov", ".avi", ".mkv"}
)

// Profile validation constraints
const (
	MinAge    = 18
	MaxAge    = 80
	MinHeight = 130 // cm
	MaxHeight = 220 // cm
)

// Photo constraints
const (
	MaxAdditionalPhotos  = 3
	MinDisplayOrder      = 1
	MaxDisplayOrder      = 3
	MaxPhotoDisplayOrder = 3 // Alias for MaxDisplayOrder for clarity
)

// Pagination constants
const (
	DefaultPaginationLimit = 10  // Default number of items per page
	MaxPaginationLimit     = 100 // Maximum number of items per page
	MinPaginationOffset    = 0   // Minimum offset value
)

// Match action constants
var (
	// ValidMatchActions defines the allowed match actions
	ValidMatchActions = []string{string(MatchStatusLiked), string(MatchStatusPassed)}
)

// Default values
const (
	DefaultNotMentioned = "not_mentioned"
)

// Profile enums
type Community string

const (
	CommunitySunni        Community = "sunni"
	CommunityMujahid      Community = "mujahid"
	CommunityTabligh      Community = "tabligh"
	CommunityJamateIslami Community = "jamate_islami"
	CommunityShia         Community = "shia"
	CommunityMuslim       Community = "muslim"
	CommunityNotMentioned Community = DefaultNotMentioned
)

type MaritalStatus string

const (
	MaritalNeverMarried  MaritalStatus = "never_married"
	MaritalDivorced      MaritalStatus = "divorced"
	MaritalNikkahDivorce MaritalStatus = "nikkah_divorce"
	MaritalWidowed       MaritalStatus = "widowed"
	MaritalNotMentioned  MaritalStatus = DefaultNotMentioned
)

type Profession string

const (
	ProfessionStudent      Profession = "student"
	ProfessionDoctor       Profession = "doctor"
	ProfessionEngineer     Profession = "engineer"
	ProfessionFarmer       Profession = "farmer"
	ProfessionTeacher      Profession = "teacher"
	ProfessionNotMentioned Profession = DefaultNotMentioned
)

type ProfessionType string

const (
	ProfessionTypeFullTime     ProfessionType = "full_time"
	ProfessionTypePartTime     ProfessionType = "part_time"
	ProfessionTypeFreelance    ProfessionType = "freelance"
	ProfessionTypeSelfEmployed ProfessionType = "self_employed"
	ProfessionTypeNotWorking   ProfessionType = "not_working"
	ProfessionTypeNotMentioned ProfessionType = DefaultNotMentioned
)

type EducationLevel string

const (
	EducationLessThanHighSchool EducationLevel = "less_than_high_school"
	EducationHighSchool         EducationLevel = "high_school"
	EducationHigherSecondary    EducationLevel = "higher_secondary"
	EducationUnderGraduation    EducationLevel = "under_graduation"
	EducationPostGraduation     EducationLevel = "post_graduation"
	EducationNotMentioned       EducationLevel = DefaultNotMentioned
)

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
	DistrictNotMentioned       HomeDistrict = DefaultNotMentioned
)

// Match status constants
type MatchStatus string

const (
	// MatchStatusLiked indicates the user liked the profile
	MatchStatusLiked MatchStatus = "liked"

	// MatchStatusPassed indicates the user passed on the profile
	MatchStatusPassed MatchStatus = "passed"
)

// Legacy alias for backward compatibility
const (
	MatchStatusPass = MatchStatusPassed // Alias for consistency
)
