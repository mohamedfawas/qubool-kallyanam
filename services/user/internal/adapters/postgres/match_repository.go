package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/indianstandardtime"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/repositories"
	"gorm.io/gorm"
)

type MatchRepository struct {
	db          *gorm.DB
	hardFilters *config.HardFiltersConfig
}

func NewMatchRepository(db *gorm.DB, cfg *config.Config) repositories.MatchRepository {
	return &MatchRepository{
		db:          db,
		hardFilters: &cfg.Matchmaking.HardFilters,
	}
}

func (r *MatchRepository) GetMatchedProfileIDs(ctx context.Context,
	userID uuid.UUID) ([]uuid.UUID, error) {
	var matchedIDs []uuid.UUID
	err := r.db.WithContext(ctx).Model(&models.ProfileMatch{}).
		Where("user_id = ?", userID).
		Pluck("target_id", &matchedIDs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get matched profile IDs: %w", err)
	}
	return matchedIDs, nil
}

func (r *MatchRepository) GetPotentialProfiles(
	ctx context.Context,
	userID uuid.UUID,
	excludeIDs []uuid.UUID,
	preferences *models.PartnerPreferences) ([]*models.UserProfile, error) {

	query := r.db.WithContext(ctx).Model(&models.UserProfile{}).
		Where("user_id != ?", userID)

	// Get the gender preference (bride/groom) information of the respective user
	var userProfile models.UserProfile
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&userProfile).Error; err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	// Filter profiles by opposite gender
	query = query.Where("is_bride != ?", userProfile.IsBride)

	// Exclude already matched profiles
	if len(excludeIDs) > 0 {
		query = query.Where("user_id NOT IN ?", excludeIDs)
	}

	// Apply hard filters based on partner preferences
	if preferences != nil {
		query = r.applyHardFilters(query, preferences)
	}

	var profiles []*models.UserProfile
	if err := query.Find(&profiles).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch potential profiles: %w", err)
	}

	return profiles, nil
}

func (r *MatchRepository) applyHardFilters(query *gorm.DB, preferences *models.PartnerPreferences) *gorm.DB {
	if !r.hardFilters.Enabled {
		return query
	}

	// Hard filter 1: Age range
	if r.hardFilters.ApplyAgeFilter && (preferences.MinAgeYears != nil || preferences.MaxAgeYears != nil) {
		if preferences.MinAgeYears != nil {
			// calculates the maximum birthdate someone can have in order to be at least MinAgeYears old
			maxBirthDate := time.Now().AddDate(-*preferences.MinAgeYears, 0, 0)
			query = query.Where("date_of_birth <= ? OR date_of_birth IS NULL", maxBirthDate)
		}
		if preferences.MaxAgeYears != nil {
			// calculates the minimum birthdate someone can have to be at most MaxAgeYears old.
			// We subtract MaxAgeYears + 1 from the current year to ensure we're handling partial-year birthdays correctly
			minBirthDate := time.Now().AddDate(-*preferences.MaxAgeYears-1, 0, 0)
			query = query.Where("date_of_birth > ? OR date_of_birth IS NULL", minBirthDate)
		}
	}

	// Hard filter 2: Height range
	if r.hardFilters.ApplyHeightFilter {
		if preferences.MinHeightCM != nil {
			query = query.Where("height_cm >= ? OR height_cm IS NULL", *preferences.MinHeightCM)
		}
		if preferences.MaxHeightCM != nil {
			query = query.Where("height_cm <= ? OR height_cm IS NULL", *preferences.MaxHeightCM)
		}
	}

	// Hard filter 3: Physically challenged
	if r.hardFilters.ApplyPhysicallyChallengedFilter && !preferences.AcceptPhysicallyChallenged {
		query = query.Where("physically_challenged = false")
	}

	// Hard filter 4: Marital status
	if r.hardFilters.ApplyMaritalStatusFilter && len(preferences.PreferredMaritalStatus) > 0 {
		maritalStatusStrings := make([]string, len(preferences.PreferredMaritalStatus))
		for i, status := range preferences.PreferredMaritalStatus {
			maritalStatusStrings[i] = string(status)
		}
		query = query.Where("marital_status IN ? OR marital_status = ?", maritalStatusStrings, models.MaritalNotMentioned)
	}

	// Hard filter 5: Education level
	if r.hardFilters.ApplyEducationFilter && len(preferences.PreferredEducationLevels) > 0 {
		educationLevelStrings := make([]string, len(preferences.PreferredEducationLevels))
		for i, level := range preferences.PreferredEducationLevels {
			educationLevelStrings[i] = string(level)
		}
		query = query.Where("highest_education_level IN ? OR highest_education_level = ?", educationLevelStrings, models.EducationNotMentioned)
	}

	return query
}

func (r *MatchRepository) RecordMatchAction(ctx context.Context, userID, targetID uuid.UUID, status models.MatchStatus) error {
	var existingMatch models.ProfileMatch
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND target_id = ?", userID, targetID).
		First(&existingMatch).Error

	now := indianstandardtime.Now()
	if err == nil {
		return r.db.WithContext(ctx).Model(&existingMatch).
			Updates(map[string]interface{}{
				"status":     status,
				"updated_at": now,
			}).Error
	}

	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("error checking for existing match: %w", err)
	}

	match := models.ProfileMatch{
		UserID:    userID,
		TargetID:  targetID,
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return r.db.WithContext(ctx).Create(&match).Error
}

func (r *MatchRepository) CheckForMutualMatch(ctx context.Context, userID1, userID2 uuid.UUID) (bool, error) {
	var count int64

	// First query: has Alice liked Bob?
	//
	//    SELECT COUNT(*)
	//    FROM profile_matches
	//    WHERE user_id = userID1
	//      AND target_id = userID2
	//      AND status = MatchStatusLiked;
	err := r.db.WithContext(ctx).Model(&models.ProfileMatch{}).
		Where("user_id = ? AND target_id = ? AND status = ?", userID1, userID2, models.MatchStatusLiked).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("error checking first direction of match: %w", err)
	}

	if count == 0 {
		return false, nil
	}

	// Second query: has Bob liked Alice?
	err = r.db.WithContext(ctx).Model(&models.ProfileMatch{}).
		Where("user_id = ? AND target_id = ? AND status = ?", userID2, userID1, models.MatchStatusLiked).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("error checking second direction of match: %w", err)
	}

	return count > 0, nil
}

func (r *MatchRepository) CreateMutualMatch(ctx context.Context, userID1, userID2 uuid.UUID) error {
	// Ensure consistent ordering of IDs
	if userID1.String() > userID2.String() {
		userID1, userID2 = userID2, userID1
	}

	var count int64
	err := r.db.WithContext(ctx).Model(&models.MutualMatch{}).
		Where("user_id_1 = ? AND user_id_2 = ?", userID1, userID2).
		Count(&count).Error
	if err != nil {
		return fmt.Errorf("error checking for existing mutual match: %w", err)
	}

	now := indianstandardtime.Now()
	if count > 0 {
		return r.db.WithContext(ctx).Model(&models.MutualMatch{}).
			Where("user_id_1 = ? AND user_id_2 = ?", userID1, userID2).
			Updates(map[string]interface{}{
				"is_active":  true,
				"updated_at": now,
			}).Error
	}

	match := models.MutualMatch{
		UserID1:   userID1,
		UserID2:   userID2,
		MatchedAt: now,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return r.db.WithContext(ctx).Create(&match).Error
}

func (r *MatchRepository) GetMatchHistory(ctx context.Context, userID uuid.UUID, status *models.MatchStatus, limit, offset int) ([]*models.MatchHistoryItem, int, error) {
	// Build the base query with JOIN to get profile details
	query := r.db.WithContext(ctx).
		Table("profile_matches pm").
		Select(`
			up.id as profile_id,
			up.full_name,
			up.date_of_birth,
			up.height_cm,
			up.physically_challenged,
			up.community,
			up.marital_status,
			up.profession,
			up.profession_type,
			up.highest_education_level,
			up.home_district,
			up.profile_picture_url,
			pm.status as action,
			pm.created_at as action_date
		`).
		Joins("JOIN user_profiles up ON pm.target_id = up.user_id").
		Where("pm.user_id = ? AND pm.is_deleted = false", userID)

	// Add status filter if provided
	if status != nil {
		query = query.Where("pm.status = ?", *status)
	}

	// Get total count for pagination
	var total int64
	countQuery := r.db.WithContext(ctx).
		Table("profile_matches pm").
		Where("pm.user_id = ? AND pm.is_deleted = false", userID)

	if status != nil {
		countQuery = countQuery.Where("pm.status = ?", *status)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count match history: %w", err)
	}

	// Apply pagination and ordering
	query = query.
		Order("pm.created_at DESC").
		Limit(limit).
		Offset(offset)

	// Execute query
	var rows []struct {
		ProfileID             uint                  `gorm:"column:profile_id"`
		FullName              string                `gorm:"column:full_name"`
		DateOfBirth           *time.Time            `gorm:"column:date_of_birth"`
		HeightCM              *int                  `gorm:"column:height_cm"`
		PhysicallyChallenged  bool                  `gorm:"column:physically_challenged"`
		Community             models.Community      `gorm:"column:community"`
		MaritalStatus         models.MaritalStatus  `gorm:"column:marital_status"`
		Profession            models.Profession     `gorm:"column:profession"`
		ProfessionType        models.ProfessionType `gorm:"column:profession_type"`
		HighestEducationLevel models.EducationLevel `gorm:"column:highest_education_level"`
		HomeDistrict          models.HomeDistrict   `gorm:"column:home_district"`
		ProfilePictureURL     *string               `gorm:"column:profile_picture_url"`
		Action                models.MatchStatus    `gorm:"column:action"`
		ActionDate            time.Time             `gorm:"column:action_date"`
	}

	if err := query.Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get match history: %w", err)
	}

	// Convert to domain models
	items := make([]*models.MatchHistoryItem, len(rows))
	for i, row := range rows {
		var age int
		if row.DateOfBirth != nil {
			age = calculateAge(*row.DateOfBirth)
		}

		items[i] = &models.MatchHistoryItem{
			ProfileID:             row.ProfileID,
			FullName:              row.FullName,
			Age:                   age,
			HeightCM:              row.HeightCM,
			PhysicallyChallenged:  row.PhysicallyChallenged,
			Community:             row.Community,
			MaritalStatus:         row.MaritalStatus,
			Profession:            row.Profession,
			ProfessionType:        row.ProfessionType,
			HighestEducationLevel: row.HighestEducationLevel,
			HomeDistrict:          row.HomeDistrict,
			ProfilePictureURL:     row.ProfilePictureURL,
			Action:                row.Action,
			ActionDate:            row.ActionDate,
		}
	}

	return items, int(total), nil
}

func calculateAge(dateOfBirth time.Time) int {
	now := time.Now()
	age := now.Year() - dateOfBirth.Year()
	if now.Month() < dateOfBirth.Month() || (now.Month() == dateOfBirth.Month() && now.Day() < dateOfBirth.Day()) {
		age--
	}
	return age
}

func (r *MatchRepository) DeactivateMutualMatch(ctx context.Context, userID1, userID2 uuid.UUID) error {
	// Ensure consistent ordering of IDs
	if userID1.String() > userID2.String() {
		userID1, userID2 = userID2, userID1
	}

	now := indianstandardtime.Now()
	return r.db.WithContext(ctx).Model(&models.MutualMatch{}).
		Where("user_id_1 = ? AND user_id_2 = ? AND is_active = true", userID1, userID2).
		Updates(map[string]interface{}{
			"is_active":  false,
			"updated_at": now,
		}).Error
}

func (r *MatchRepository) GetMutualMatches(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.MutualMatchData, int, error) {
	// Build the base query with JOIN to get profile details
	// We need to find mutual matches where the user is either user_id_1 or user_id_2
	query := r.db.WithContext(ctx).
		Table("mutual_matches mm").
		Select(`
			up.id as profile_id,
			up.full_name,
			up.date_of_birth,
			up.height_cm,
			up.physically_challenged,
			up.community,
			up.marital_status,
			up.profession,
			up.profession_type,
			up.highest_education_level,
			up.home_district,
			up.profile_picture_url,
			up.last_login,
			mm.matched_at
		`).
		Joins(`JOIN user_profiles up ON 
			(mm.user_id_1 = ? AND mm.user_id_2 = up.user_id) OR 
			(mm.user_id_2 = ? AND mm.user_id_1 = up.user_id)`, userID, userID).
		Where("mm.is_active = true AND mm.is_deleted = false")

	// Get total count for pagination
	var total int64
	countQuery := r.db.WithContext(ctx).
		Table("mutual_matches mm").
		Where(`(mm.user_id_1 = ? OR mm.user_id_2 = ?) AND mm.is_active = true AND mm.is_deleted = false`, userID, userID)

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count mutual matches: %w", err)
	}

	// Apply pagination and ordering (most recent matches first)
	query = query.
		Order("mm.matched_at DESC").
		Limit(limit).
		Offset(offset)

	// Execute query
	var rows []struct {
		ProfileID             uint                  `gorm:"column:profile_id"`
		FullName              string                `gorm:"column:full_name"`
		DateOfBirth           *time.Time            `gorm:"column:date_of_birth"`
		HeightCM              *int                  `gorm:"column:height_cm"`
		PhysicallyChallenged  bool                  `gorm:"column:physically_challenged"`
		Community             models.Community      `gorm:"column:community"`
		MaritalStatus         models.MaritalStatus  `gorm:"column:marital_status"`
		Profession            models.Profession     `gorm:"column:profession"`
		ProfessionType        models.ProfessionType `gorm:"column:profession_type"`
		HighestEducationLevel models.EducationLevel `gorm:"column:highest_education_level"`
		HomeDistrict          models.HomeDistrict   `gorm:"column:home_district"`
		ProfilePictureURL     *string               `gorm:"column:profile_picture_url"`
		LastLogin             time.Time             `gorm:"column:last_login"`
		MatchedAt             time.Time             `gorm:"column:matched_at"`
	}

	if err := query.Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get mutual matches: %w", err)
	}

	// Convert to domain models
	items := make([]*models.MutualMatchData, len(rows))
	for i, row := range rows {
		var age int
		if row.DateOfBirth != nil {
			age = calculateAge(*row.DateOfBirth)
		}

		items[i] = &models.MutualMatchData{
			ProfileID:             row.ProfileID,
			FullName:              row.FullName,
			Age:                   age,
			HeightCM:              row.HeightCM,
			PhysicallyChallenged:  row.PhysicallyChallenged,
			Community:             row.Community,
			MaritalStatus:         row.MaritalStatus,
			Profession:            row.Profession,
			ProfessionType:        row.ProfessionType,
			HighestEducationLevel: row.HighestEducationLevel,
			HomeDistrict:          row.HomeDistrict,
			ProfilePictureURL:     row.ProfilePictureURL,
			LastLogin:             row.LastLogin,
			MatchedAt:             row.MatchedAt,
		}
	}

	return items, int(total), nil
}
