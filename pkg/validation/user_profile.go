// pkg/validation/profile.go
package validation

import (
	"errors"
	"time"
)

var (
	ErrInvalidHeight         = errors.New("height must be between 130 and 220 cm")
	ErrInvalidCommunity      = errors.New("invalid community value")
	ErrInvalidMaritalStatus  = errors.New("invalid marital status value")
	ErrInvalidProfession     = errors.New("invalid profession value")
	ErrInvalidProfessionType = errors.New("invalid profession type value")
	ErrInvalidEducationLevel = errors.New("invalid education level value")
	ErrInvalidHomeDistrict   = errors.New("invalid home district value")
	ErrInvalidDateOfBirth    = errors.New("invalid date of birth, user must be at least 18 years old")
)

// ValidateHeight checks if height is within acceptable range
func ValidateHeight(height *int) error {
	if height == nil {
		return nil
	}

	if *height < 130 || *height > 220 {
		return ErrInvalidHeight
	}

	return nil
}

// ValidateCommunity checks if community value is valid
func ValidateCommunity(community string) error {
	validCommunities := map[string]bool{
		"sunni": true, "mujahid": true, "tabligh": true,
		"jamate_islami": true, "shia": true, "muslim": true,
		"not_mentioned": true,
	}

	if _, ok := validCommunities[community]; !ok {
		return ErrInvalidCommunity
	}

	return nil
}

// ValidateMaritalStatus checks if marital status value is valid
func ValidateMaritalStatus(status string) error {
	validStatuses := map[string]bool{
		"never_married": true, "divorced": true, "nikkah_divorce": true,
		"widowed": true, "not_mentioned": true,
	}

	if _, ok := validStatuses[status]; !ok {
		return ErrInvalidMaritalStatus
	}

	return nil
}

// ValidateProfession checks if profession value is valid
func ValidateProfession(profession string) error {
	validProfessions := map[string]bool{
		"student": true, "doctor": true, "engineer": true,
		"farmer": true, "teacher": true, "not_mentioned": true,
	}

	if _, ok := validProfessions[profession]; !ok {
		return ErrInvalidProfession
	}

	return nil
}

// ValidateProfessionType checks if profession type value is valid
func ValidateProfessionType(professionType string) error {
	validTypes := map[string]bool{
		"full_time": true, "part_time": true, "freelance": true,
		"self_employed": true, "not_working": true, "not_mentioned": true,
	}

	if _, ok := validTypes[professionType]; !ok {
		return ErrInvalidProfessionType
	}

	return nil
}

// ValidateEducationLevel checks if education level value is valid
func ValidateEducationLevel(level string) error {
	validLevels := map[string]bool{
		"less_than_high_school": true, "high_school": true, "higher_secondary": true,
		"under_graduation": true, "post_graduation": true, "not_mentioned": true,
	}

	if _, ok := validLevels[level]; !ok {
		return ErrInvalidEducationLevel
	}

	return nil
}

// ValidateHomeDistrict checks if home district value is valid
func ValidateHomeDistrict(district string) error {
	validDistricts := map[string]bool{
		"thiruvananthapuram": true, "kollam": true, "pathanamthitta": true,
		"alappuzha": true, "kottayam": true, "ernakulam": true, "thrissur": true,
		"palakkad": true, "malappuram": true, "kozhikode": true, "wayanad": true,
		"kannur": true, "kasaragod": true, "idukki": true, "not_mentioned": true,
	}

	if _, ok := validDistricts[district]; !ok {
		return ErrInvalidHomeDistrict
	}

	return nil
}

// ValidateDateOfBirth ensures user is at least 18 years old
func ValidateDateOfBirth(dob *time.Time) error {
	if dob == nil {
		return nil
	}

	minAge := 18
	now := time.Now()
	yearDiff := now.Year() - dob.Year()

	// Check if birthday has occurred this year
	if now.Month() < dob.Month() || (now.Month() == dob.Month() && now.Day() < dob.Day()) {
		yearDiff--
	}

	if yearDiff < minAge {
		return ErrInvalidDateOfBirth
	}

	return nil
}
