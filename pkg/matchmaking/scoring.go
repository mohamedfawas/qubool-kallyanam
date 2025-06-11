package matchmaking

import (
	"time"
)

// CalculateAge computes age in years from a date of birth
func CalculateAge(dob time.Time) int {
	now := time.Now()
	age := now.Year() - dob.Year()

	// Adjust if birthday hasn't occurred yet this year
	if now.Month() < dob.Month() || (now.Month() == dob.Month() && now.Day() < dob.Day()) {
		age--
	}

	return age
}

// CalculateMatchScore determines compatibility between two profiles
// Returns a score from 0-100 and reasons for the match
func CalculateMatchScore(profile ProfileData, preferences PreferenceData) MatchScore {
	score := 0.0
	var reasons []string

	// Check gender compatibility (implicitly handled by only showing opposite gender)

	// Age compatibility (up to 15 points)
	if preferences.MinAgeYears != nil && preferences.MaxAgeYears != nil {
		if profile.Age >= *preferences.MinAgeYears && profile.Age <= *preferences.MaxAgeYears {
			score += 15.0
			reasons = append(reasons, "Age matches preferences")
		} else {
			// Partial points for close matches
			ageDiff := 0
			if profile.Age < *preferences.MinAgeYears {
				ageDiff = *preferences.MinAgeYears - profile.Age
			} else {
				ageDiff = profile.Age - *preferences.MaxAgeYears
			}

			if ageDiff <= 2 {
				score += 5.0
				reasons = append(reasons, "Age close to preferences")
			}
		}
	}

	// Height compatibility (up to 10 points)
	if preferences.MinHeightCM != nil && preferences.MaxHeightCM != nil && profile.HeightCM > 0 {
		if profile.HeightCM >= *preferences.MinHeightCM && profile.HeightCM <= *preferences.MaxHeightCM {
			score += 10.0
			reasons = append(reasons, "Height matches preferences")
		}
	}

	// Community match (up to 20 points)
	if len(preferences.PreferredCommunities) > 0 {
		for _, c := range preferences.PreferredCommunities {
			if profile.Community == c {
				score += 20.0
				reasons = append(reasons, "Community matches preference")
				break
			}
		}
	}

	// Marital status match (up to 15 points)
	if len(preferences.PreferredMaritalStatus) > 0 {
		for _, ms := range preferences.PreferredMaritalStatus {
			if profile.MaritalStatus == ms {
				score += 15.0
				reasons = append(reasons, "Marital status matches preference")
				break
			}
		}
	}

	// Education match (up to 10 points)
	if len(preferences.PreferredEducationLevels) > 0 {
		for _, el := range preferences.PreferredEducationLevels {
			if profile.HighestEducationLevel == el {
				score += 10.0
				reasons = append(reasons, "Education level matches preference")
				break
			}
		}
	}

	// Profession match (up to 10 points)
	if len(preferences.PreferredProfessions) > 0 {
		for _, p := range preferences.PreferredProfessions {
			if profile.Profession == p {
				score += 10.0
				reasons = append(reasons, "Profession matches preference")
				break
			}
		}
	}

	// Profession type match (up to 5 points)
	if len(preferences.PreferredProfessionTypes) > 0 {
		for _, pt := range preferences.PreferredProfessionTypes {
			if profile.ProfessionType == pt {
				score += 5.0
				reasons = append(reasons, "Profession type matches preference")
				break
			}
		}
	}

	// Home district match (up to 10 points)
	if len(preferences.PreferredHomeDistricts) > 0 {
		for _, hd := range preferences.PreferredHomeDistricts {
			if profile.HomeDistrict == hd {
				score += 10.0
				reasons = append(reasons, "Home district matches preference")
				break
			}
		}
	}

	// Physical challenge compatibility (potential deal-breaker)
	if profile.PhysicallyChallenged && !preferences.AcceptPhysicallyChallenged {
		// Significant penalty if user doesn't accept physically challenged
		score -= 40.0
	}

	// Cap the score at 0-100 range
	if score > 100.0 {
		score = 100.0
	}
	if score < 0.0 {
		score = 0.0
	}

	return MatchScore{
		Score:   score,
		Reasons: reasons,
	}
}

// BuildProfileData converts raw profile data into the format needed for matching
func BuildProfileData(
	isBride bool,
	dateOfBirth *time.Time,
	heightCM *int,
	physicallyChallenged bool,
	community string,
	maritalStatus string,
	profession string,
	professionType string,
	educationLevel string,
	homeDistrict string,
) ProfileData {

	profile := ProfileData{
		IsBride:               isBride,
		PhysicallyChallenged:  physicallyChallenged,
		Community:             community,
		MaritalStatus:         maritalStatus,
		Profession:            profession,
		ProfessionType:        professionType,
		HighestEducationLevel: educationLevel,
		HomeDistrict:          homeDistrict,
	}

	// Calculate age if date of birth provided
	if dateOfBirth != nil {
		profile.Age = CalculateAge(*dateOfBirth)
	}

	// Set height if provided
	if heightCM != nil {
		profile.HeightCM = *heightCM
	}

	return profile
}

// BuildPreferenceData converts raw preference data into the format needed for matching
func BuildPreferenceData(
	minAgeYears *int,
	maxAgeYears *int,
	minHeightCM *int,
	maxHeightCM *int,
	acceptPhysicallyChallenged bool,
	preferredCommunities []string,
	preferredMaritalStatus []string,
	preferredProfessions []string,
	preferredProfessionTypes []string,
	preferredEducationLevels []string,
	preferredHomeDistricts []string,
) PreferenceData {

	return PreferenceData{
		MinAgeYears:                minAgeYears,
		MaxAgeYears:                maxAgeYears,
		MinHeightCM:                minHeightCM,
		MaxHeightCM:                maxHeightCM,
		AcceptPhysicallyChallenged: acceptPhysicallyChallenged,
		PreferredCommunities:       preferredCommunities,
		PreferredMaritalStatus:     preferredMaritalStatus,
		PreferredProfessions:       preferredProfessions,
		PreferredProfessionTypes:   preferredProfessionTypes,
		PreferredEducationLevels:   preferredEducationLevels,
		PreferredHomeDistricts:     preferredHomeDistricts,
	}
}
