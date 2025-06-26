package postgres

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/constants"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/repositories"
)

// MatchRepo implements the match repository interface
type MatchRepo struct {
	db *gorm.DB
}

// NewMatchRepository creates a new match repository
func NewMatchRepository(db *gorm.DB) repositories.MatchRepository {
	return &MatchRepo{
		db: db,
	}
}

// GetMatchedProfileIDs retrieves profile IDs that the user has already acted on
func (r *MatchRepo) GetMatchedProfileIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	var targetIDs []uuid.UUID
	err := r.db.WithContext(ctx).
		Model(&models.ProfileMatch{}).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Pluck("target_id", &targetIDs).Error

	return targetIDs, err
}

// GetPotentialProfiles retrieves potential profiles for matching based on preferences
func (r *MatchRepo) GetPotentialProfiles(
	ctx context.Context,
	userID uuid.UUID,
	excludeIDs []uuid.UUID,
	preferences *models.PartnerPreferences) ([]*models.UserProfile, error) {

	query := r.db.WithContext(ctx).
		Where("user_id != ? AND is_deleted = ?", userID, false)

	// Exclude already matched profiles
	if len(excludeIDs) > 0 {
		query = query.Where("user_id NOT IN ?", excludeIDs)
	}

	// Apply preferences filters if provided
	if preferences != nil {
		if preferences.MinAgeYears != nil && preferences.MaxAgeYears != nil {
			query = query.Where("EXTRACT(YEAR FROM AGE(date_of_birth)) BETWEEN ? AND ?",
				*preferences.MinAgeYears, *preferences.MaxAgeYears)
		}

		if preferences.MinHeightCM != nil && preferences.MaxHeightCM != nil {
			query = query.Where("height_cm BETWEEN ? AND ?",
				*preferences.MinHeightCM, *preferences.MaxHeightCM)
		}

		if !preferences.AcceptPhysicallyChallenged {
			query = query.Where("physically_challenged = ?", false)
		}

		if len(preferences.PreferredCommunities) > 0 {
			communities := make([]string, len(preferences.PreferredCommunities))
			for i, c := range preferences.PreferredCommunities {
				communities[i] = string(c)
			}
			query = query.Where("community IN ?", communities)
		}

		if len(preferences.PreferredMaritalStatus) > 0 {
			statuses := make([]string, len(preferences.PreferredMaritalStatus))
			for i, s := range preferences.PreferredMaritalStatus {
				statuses[i] = string(s)
			}
			query = query.Where("marital_status IN ?", statuses)
		}

		if len(preferences.PreferredProfessions) > 0 {
			professions := make([]string, len(preferences.PreferredProfessions))
			for i, p := range preferences.PreferredProfessions {
				professions[i] = string(p)
			}
			query = query.Where("profession IN ?", professions)
		}

		if len(preferences.PreferredEducationLevels) > 0 {
			levels := make([]string, len(preferences.PreferredEducationLevels))
			for i, l := range preferences.PreferredEducationLevels {
				levels[i] = string(l)
			}
			query = query.Where("highest_education_level IN ?", levels)
		}

		if len(preferences.PreferredHomeDistricts) > 0 {
			districts := make([]string, len(preferences.PreferredHomeDistricts))
			for i, d := range preferences.PreferredHomeDistricts {
				districts[i] = string(d)
			}
			query = query.Where("home_district IN ?", districts)
		}
	}

	var profiles []*models.UserProfile
	err := query.Order("last_login DESC").Limit(50).Find(&profiles).Error

	return profiles, err
}

// RecordMatchAction records a user's action on another profile
// RecordMatchAction records or updates a user's action on another profile
func (r *MatchRepo) RecordMatchAction(ctx context.Context, userID, targetID uuid.UUID, status models.MatchStatus) error {
	var match models.ProfileMatch

	// Try to find existing record first
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND target_id = ? AND is_deleted = ?", userID, targetID, false).
		First(&match)

	if result.Error == gorm.ErrRecordNotFound {
		// Record doesn't exist, create new one
		match = models.ProfileMatch{
			UserID:   userID,
			TargetID: targetID,
			Status:   status,
		}
		return r.db.WithContext(ctx).Create(&match).Error
	} else if result.Error != nil {
		// Some other error occurred
		return result.Error
	} else {
		// Record exists, update it
		match.Status = status
		return r.db.WithContext(ctx).Save(&match).Error
	}
}

// CheckForMutualMatch checks if two users have liked each other
func (r *MatchRepo) CheckForMutualMatch(ctx context.Context, userID1, userID2 uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.ProfileMatch{}).
		Where(
			"((user_id = ? AND target_id = ?) OR (user_id = ? AND target_id = ?)) AND status = ? AND is_deleted = ?",
			userID1, userID2, userID2, userID1, constants.MatchStatusLiked, false,
		).
		Count(&count).Error

	return count == 2, err // Both users must have liked each other
}

// CreateMutualMatch creates a mutual match record
func (r *MatchRepo) CreateMutualMatch(ctx context.Context, userID1, userID2 uuid.UUID) error {
	// Ensure user_id_1 < user_id_2 (database constraint requirement)
	if userID1.String() > userID2.String() {
		userID1, userID2 = userID2, userID1
	}

	mutualMatch := &models.MutualMatch{
		UserID1: userID1,
		UserID2: userID2,
	}

	return r.db.WithContext(ctx).Create(mutualMatch).Error
}

// DeactivateMutualMatch deactivates a mutual match
func (r *MatchRepo) DeactivateMutualMatch(ctx context.Context, userID1, userID2 uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&models.MutualMatch{}).
		Where(
			"((user_id_1 = ? AND user_id_2 = ?) OR (user_id_1 = ? AND user_id_2 = ?)) AND is_deleted = ?",
			userID1, userID2, userID2, userID1, false,
		).
		Update("is_active", false).Error
}

// GetMatchHistory retrieves a user's match history
func (r *MatchRepo) GetMatchHistory(ctx context.Context, userID uuid.UUID, status *models.MatchStatus, limit, offset int) ([]*models.MatchHistoryItem, int, error) {
	query := r.db.WithContext(ctx).
		Table("profile_matches pm").
		Select(`
			up.id as profile_id,
			up.full_name,
			EXTRACT(YEAR FROM AGE(up.date_of_birth)) as age,
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
		Where("pm.user_id = ? AND pm.is_deleted = ? AND up.is_deleted = ?", userID, false, false)

	if status != nil {
		query = query.Where("pm.status = ?", *status)
	}

	// Get total count
	var total int64
	countQuery := r.db.WithContext(ctx).
		Table("profile_matches pm").
		Joins("JOIN user_profiles up ON pm.target_id = up.user_id").
		Where("pm.user_id = ? AND pm.is_deleted = ? AND up.is_deleted = ?", userID, false, false)

	if status != nil {
		countQuery = countQuery.Where("pm.status = ?", *status)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	var history []*models.MatchHistoryItem
	err := query.
		Order("pm.created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&history).Error

	return history, int(total), err
}

// GetMutualMatches retrieves mutual matches for a user
func (r *MatchRepo) GetMutualMatches(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.MutualMatchData, int, error) {
	// Get total count
	var total int64
	countQuery := r.db.WithContext(ctx).
		Table("mutual_matches mm").
		Joins("JOIN user_profiles up ON (CASE WHEN mm.user_id_1 = ? THEN mm.user_id_2 ELSE mm.user_id_1 END) = up.user_id", userID).
		Where("(mm.user_id_1 = ? OR mm.user_id_2 = ?) AND mm.is_active = ? AND mm.is_deleted = ? AND up.is_deleted = ?",
			userID, userID, true, false, false)

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	query := r.db.WithContext(ctx).
		Table("mutual_matches mm").
		Select(`
			up.id as profile_id,
			up.full_name,
			EXTRACT(YEAR FROM AGE(up.date_of_birth)) as age,
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
		Joins("JOIN user_profiles up ON (CASE WHEN mm.user_id_1 = ? THEN mm.user_id_2 ELSE mm.user_id_1 END) = up.user_id", userID).
		Where("(mm.user_id_1 = ? OR mm.user_id_2 = ?) AND mm.is_active = ? AND mm.is_deleted = ? AND up.is_deleted = ?",
			userID, userID, true, false, false)

	var matches []*models.MutualMatchData
	err := query.
		Order("mm.matched_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&matches).Error

	return matches, int(total), err
}
