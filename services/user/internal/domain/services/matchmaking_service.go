package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/repositories"
)

var (
	ErrMatchActionFailed       = errors.New("failed to record match action")
	ErrNoMatches               = errors.New("no matches found")
	ErrInvalidActionType       = errors.New("invalid action type")
	ErrInvalidMatchingCriteria = errors.New("invalid matching criteria")
)

type MatchmakingService struct {
	matchRepo           repositories.MatchRepository
	profileRepo         repositories.ProfileRepository
	notificationService *NotificationService
	logger              logging.Logger
	matchWeights        config.MatchWeights
}

func NewMatchmakingService(
	matchRepo repositories.MatchRepository,
	profileRepo repositories.ProfileRepository,
	notificationService *NotificationService,
	logger logging.Logger,
	config *config.Config,
) *MatchmakingService {
	return &MatchmakingService{
		matchRepo:           matchRepo,
		profileRepo:         profileRepo,
		notificationService: notificationService,
		logger:              logger,
		matchWeights:        config.Matchmaking.Weights,
	}
}

func (s *MatchmakingService) GetRecommendedMatches(ctx context.Context, userID string, limit, offset int) ([]*models.RecommendedProfile, *models.PaginationData, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: invalid user ID format: %v", ErrInvalidInput, err)
	}

	userProfile, err := s.profileRepo.GetProfileByUserID(ctx, userUUID)
	if err != nil {
		return nil, nil, fmt.Errorf("error retrieving user profile: %w", err)
	}
	if userProfile == nil {
		return nil, nil, ErrProfileNotFound
	}

	preferences, err := s.profileRepo.GetPartnerPreferences(ctx, userProfile.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("error retrieving partner preferences: %w", err)
	}

	excludedIDs, err := s.matchRepo.GetMatchedProfileIDs(ctx, userUUID)
	if err != nil {
		s.logger.Warn("Failed to get matched profile IDs", "error", err, "userID", userID)
		excludedIDs = []uuid.UUID{}
	}

	// Get potential profiles with hard filters applied
	potentialProfiles, err := s.matchRepo.GetPotentialProfiles(ctx, userUUID, excludedIDs, preferences)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get potential profiles: %w", err)
	}

	// Score and sort profiles using scoring logic
	scoredProfiles := s.scoreAndSortProfiles(potentialProfiles, preferences)

	totalCount := len(scoredProfiles)
	end := offset + limit // end index of the profiles to be returned
	if end > totalCount {
		end = totalCount
	}

	var result []*models.UserProfile
	if offset < totalCount {
		result = scoredProfiles[offset:end]
	} else {
		result = []*models.UserProfile{}
	}

	if len(result) == 0 {
		s.logger.Info("No matches found for user", "userID", userID)
		return []*models.RecommendedProfile{}, &models.PaginationData{
			Total:   0,
			Limit:   limit,
			Offset:  offset,
			HasMore: false,
		}, nil
	}

	recommendedProfiles := make([]*models.RecommendedProfile, len(result))
	for i, profile := range result {
		var age int
		if profile.DateOfBirth != nil {
			age = s.calculateAge(*profile.DateOfBirth)
		}

		matchReasons := s.determineMatchReasons(profile, preferences)
		recommendedProfiles[i] = &models.RecommendedProfile{
			ID:                    profile.ID,
			UserID:                profile.UserID,
			FullName:              profile.FullName,
			Age:                   age,
			HeightCM:              profile.HeightCM,
			PhysicallyChallenged:  profile.PhysicallyChallenged,
			Community:             profile.Community,
			MaritalStatus:         profile.MaritalStatus,
			Profession:            profile.Profession,
			ProfessionType:        profile.ProfessionType,
			HighestEducationLevel: profile.HighestEducationLevel,
			HomeDistrict:          profile.HomeDistrict,
			ProfilePictureURL:     profile.ProfilePictureURL,
			LastLogin:             profile.LastLogin,
			MatchReasons:          matchReasons,
		}
	}

	pagination := &models.PaginationData{
		Total:   totalCount,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+limit < totalCount,
	}

	return recommendedProfiles, pagination, nil
}

func (s *MatchmakingService) RecordMatchAction(ctx context.Context, userID string, profileID uint, action string) (bool, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return false, fmt.Errorf("%w: invalid user ID format: %v", ErrInvalidInput, err)
	}

	targetProfile, err := s.profileRepo.GetProfileByID(ctx, profileID)
	if err != nil {
		return false, fmt.Errorf("error retrieving target profile: %w", err)
	}

	if targetProfile == nil {
		return false, fmt.Errorf("%w: target profile not found", ErrInvalidInput)
	}

	targetUUID := targetProfile.UserID

	var status models.MatchStatus
	switch action {
	case "liked":
		status = models.MatchStatusLiked
	case "disliked":
		status = models.MatchStatusDisliked
	case "passed":
		status = models.MatchStatusPassed
	default:
		return false, fmt.Errorf("%w: action must be one of: liked, disliked, passed", ErrInvalidActionType)
	}

	if err := s.matchRepo.RecordMatchAction(ctx, userUUID, targetUUID, status); err != nil {
		return false, fmt.Errorf("%w: %v", ErrMatchActionFailed, err)
	}

	if status == models.MatchStatusLiked && s.notificationService != nil {
		// Get the liker's profile information
		likerProfile, err := s.profileRepo.GetProfileByUserID(ctx, userUUID)
		if err != nil {
			s.logger.Error("Failed to get liker profile for notification", "error", err, "userID", userID)
		} else if likerProfile != nil {
			// Send email notification to the target user asynchronously
			go func() {
				if err := s.notificationService.SendLikeNotificationEmail(
					context.Background(),
					targetUUID.String(),
					fmt.Sprintf("%d", likerProfile.ID), // Use profile ID, not UUID
					likerProfile.FullName,
				); err != nil {
					s.logger.Error("Failed to send like notification", "error", err)
				}
			}()
		}
	}

	var isMutualMatch bool // default is false
	if status == models.MatchStatusLiked {
		mutualMatch, err := s.matchRepo.CheckForMutualMatch(ctx, userUUID, targetUUID)
		if err != nil {
			s.logger.Error("Failed to check for mutual match", "error", err, "userID", userID, "profileID", profileID)
		} else if mutualMatch {
			if err := s.matchRepo.CreateMutualMatch(ctx, userUUID, targetUUID); err != nil {
				s.logger.Error("Failed to create mutual match record", "error", err, "userID", userID, "profileID", profileID)
			} else {
				isMutualMatch = true
				s.logger.Info("Mutual match created", "userID", userID, "profileID", profileID)
			}
		}
	}

	return isMutualMatch, nil
}

func (s *MatchmakingService) scoreAndSortProfiles(
	profiles []*models.UserProfile,
	preferences *models.PartnerPreferences) []*models.UserProfile {

	// to hold profile and its score
	type scoredProfile struct {
		profile *models.UserProfile
		score   float64
	}

	scoredProfiles := make([]scoredProfile, 0, len(profiles))

	for _, profile := range profiles {
		score := s.calculateMatchScore(profile, preferences)
		scoredProfiles = append(scoredProfiles, scoredProfile{
			profile: profile,
			score:   score,
		})
	}

	sort.Slice(scoredProfiles, func(i, j int) bool {
		// If two profiles have the same score, prefer the one who logged in more recently
		if scoredProfiles[i].score == scoredProfiles[j].score {
			return scoredProfiles[i].profile.LastLogin.After(scoredProfiles[j].profile.LastLogin)
		}
		// Sort the scored profiles in descending order of score
		return scoredProfiles[i].score > scoredProfiles[j].score
	})

	// Extract just the sorted profiles (without their scores) for the final result
	result := make([]*models.UserProfile, len(scoredProfiles))
	for i, sp := range scoredProfiles {
		result[i] = sp.profile
	}

	return result
}

func (s *MatchmakingService) calculateMatchScore(profile *models.UserProfile, prefs *models.PartnerPreferences) float64 {
	// if no preferences, use recency score (last login)
	if prefs == nil {
		return s.calculateRecencyScore(profile.LastLogin)
	}

	// holds the total matching score
	var score float64 = 0

	// Using weights from config
	weightCommunity := s.matchWeights.Community
	weightProfession := s.matchWeights.Profession
	weightLocation := s.matchWeights.Location
	weightRecency := s.matchWeights.Recency

	// A matching community gets a full score of 1.0 * weightCommunity
	// If no communities are specified, use a default score of 0.5 * weightCommunity
	if len(prefs.PreferredCommunities) > 0 {
		communityScore := 0.5
		for _, c := range prefs.PreferredCommunities {
			if profile.Community == c {
				communityScore = 1.0
				break
			}
		}
		score += communityScore * weightCommunity
	} else {
		score += 0.5 * weightCommunity
	}

	if len(prefs.PreferredProfessions) > 0 {
		professionScore := 0.5
		for _, p := range prefs.PreferredProfessions {
			if profile.Profession == p {
				professionScore = 1.0
				break
			}
		}
		score += professionScore * weightProfession
	} else {
		score += 0.5 * weightProfession
	}

	if len(prefs.PreferredHomeDistricts) > 0 {
		locationScore := 0.5
		for _, d := range prefs.PreferredHomeDistricts {
			if profile.HomeDistrict == d {
				locationScore = 1.0
				break
			}
		}
		score += locationScore * weightLocation
	} else {
		score += 0.5 * weightLocation
	}

	score += s.calculateRecencyScore(profile.LastLogin) * weightRecency

	return score
}

func (s *MatchmakingService) calculateRecencyScore(lastLogin time.Time) float64 {
	hoursAgo := time.Since(lastLogin).Hours()
	if hoursAgo < 24 {
		return 1.0
	} else if hoursAgo < 72 {
		return 0.8
	} else if hoursAgo < 168 {
		return 0.6
	} else if hoursAgo < 336 {
		return 0.4
	} else {
		return 0.2
	}
}

func (s *MatchmakingService) calculateAge(dateOfBirth time.Time) int {
	now := time.Now()
	age := now.Year() - dateOfBirth.Year()
	if now.Month() < dateOfBirth.Month() || (now.Month() == dateOfBirth.Month() && now.Day() < dateOfBirth.Day()) {
		age--
	}
	return age
}

func (s *MatchmakingService) determineMatchReasons(matchProfile *models.UserProfile, prefs *models.PartnerPreferences) []string {
	var reasons []string

	if prefs == nil {
		reasons = append(reasons, "Compatible profile")
		return reasons
	}

	if len(prefs.PreferredCommunities) > 0 {
		for _, c := range prefs.PreferredCommunities {
			if matchProfile.Community == c {
				reasons = append(reasons, "Community match")
				break
			}
		}
	}

	if len(prefs.PreferredProfessions) > 0 {
		for _, p := range prefs.PreferredProfessions {
			if matchProfile.Profession == p {
				reasons = append(reasons, "Profession match")
				break
			}
		}
	}

	if len(prefs.PreferredHomeDistricts) > 0 {
		for _, d := range prefs.PreferredHomeDistricts {
			if matchProfile.HomeDistrict == d {
				reasons = append(reasons, "Location match")
				break
			}
		}
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "Compatible profile")
	}

	if time.Since(matchProfile.LastLogin).Hours() < 24*7 {
		reasons = append(reasons, "Recently active")
	}

	return reasons
}

func (s *MatchmakingService) GetMatchHistory(ctx context.Context, userID string, statusFilter string, limit, offset int) ([]*models.MatchHistoryItem, *models.PaginationData, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: invalid user ID format: %v", ErrInvalidInput, err)
	}

	var status *models.MatchStatus
	if statusFilter != "" && statusFilter != "all" {
		switch statusFilter {
		case "liked":
			s := models.MatchStatusLiked
			status = &s
		case "disliked":
			s := models.MatchStatusDisliked
			status = &s
		case "passed":
			s := models.MatchStatusPassed
			status = &s
		// Now `status` is either:
		// - nil (no filter)
		// - a pointer to one of the valid MatchStatus values (liked, disliked, passed)
		default:
			return nil, nil, fmt.Errorf("%w: status must be one of: liked, disliked, passed, all", ErrInvalidInput)
		}
	}

	// Get match history from repository
	items, total, err := s.matchRepo.GetMatchHistory(ctx, userUUID, status, limit, offset)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get match history: %w", err)
	}

	// Create pagination data
	pagination := &models.PaginationData{
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+limit < total,
	}

	return items, pagination, nil
}

func (s *MatchmakingService) UpdateMatchAction(ctx context.Context, userID string, profileID uint, action string) (bool, bool, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return false, false, fmt.Errorf("%w: invalid user ID format: %v", ErrInvalidInput, err)
	}

	// Get target profile using profile ID
	targetProfile, err := s.profileRepo.GetProfileByID(ctx, profileID)
	if err != nil {
		return false, false, fmt.Errorf("error retrieving target profile: %w", err)
	}

	if targetProfile == nil {
		return false, false, fmt.Errorf("%w: target profile not found", ErrInvalidInput)
	}

	targetUUID := targetProfile.UserID

	// Validate action
	var newStatus models.MatchStatus
	switch action {
	case "liked":
		newStatus = models.MatchStatusLiked
	case "disliked":
		newStatus = models.MatchStatusDisliked
	case "passed":
		newStatus = models.MatchStatusPassed
	default:
		return false, false, fmt.Errorf("%w: action must be one of: liked, disliked, passed", ErrInvalidActionType)
	}

	// Check if there was a previous mutual match
	wasMutualMatch, err := s.matchRepo.CheckForMutualMatch(ctx, userUUID, targetUUID)
	if err != nil {
		s.logger.Warn("Failed to check previous mutual match status", "error", err, "userID", userID, "profileID", profileID)
		wasMutualMatch = false
	}

	// Update the match action
	if err := s.matchRepo.RecordMatchAction(ctx, userUUID, targetUUID, newStatus); err != nil {
		return false, false, fmt.Errorf("%w: %v", ErrMatchActionFailed, err)
	}

	// Check for new mutual match status
	var isMutualMatch bool
	var wasMutualMatchBroken bool

	if newStatus == models.MatchStatusLiked {
		// Check if this creates a new mutual match
		mutualMatch, err := s.matchRepo.CheckForMutualMatch(ctx, userUUID, targetUUID)
		if err != nil {
			s.logger.Error("Failed to check for mutual match", "error", err, "userID", userID, "profileID", profileID)
		} else if mutualMatch {
			if err := s.matchRepo.CreateMutualMatch(ctx, userUUID, targetUUID); err != nil {
				s.logger.Error("Failed to create mutual match record", "error", err, "userID", userID, "profileID", profileID)
			} else {
				isMutualMatch = true
				s.logger.Info("Mutual match created via update", "userID", userID, "profileID", profileID)
			}
		}
	} else {
		// If the new action is not "liked", check if we broke a mutual match
		if wasMutualMatch {
			wasMutualMatchBroken = true
			// Deactivate the mutual match
			if err := s.deactivateMutualMatch(ctx, userUUID, targetUUID); err != nil {
				s.logger.Error("Failed to deactivate mutual match", "error", err, "userID", userID, "profileID", profileID)
			} else {
				s.logger.Info("Mutual match deactivated due to action update", "userID", userID, "profileID", profileID, "newAction", action)
			}
		}
	}

	return isMutualMatch, wasMutualMatchBroken, nil
}

func (s *MatchmakingService) deactivateMutualMatch(ctx context.Context, userID1, userID2 uuid.UUID) error {
	// Ensure consistent ordering of IDs
	if userID1.String() > userID2.String() {
		userID1, userID2 = userID2, userID1
	}

	// This would need to be added to the repository interface
	return s.matchRepo.DeactivateMutualMatch(ctx, userID1, userID2)
}

func (s *MatchmakingService) GetMutualMatches(ctx context.Context, userID string, limit, offset int) ([]*models.MutualMatchData, *models.PaginationData, error) {
	// Validate and set defaults for pagination
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	// Parse user ID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: invalid user ID format: %v", ErrInvalidInput, err)
	}

	// Get mutual matches from repository
	matches, total, err := s.matchRepo.GetMutualMatches(ctx, userUUID, limit, offset)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get mutual matches: %w", err)
	}

	// Create pagination data
	pagination := &models.PaginationData{
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+limit < total,
	}

	return matches, pagination, nil
}
