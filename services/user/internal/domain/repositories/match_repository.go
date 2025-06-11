package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/models"
)

type MatchRepository interface {
	GetMatchedProfileIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetPotentialProfiles(
		ctx context.Context,
		userID uuid.UUID,
		excludeIDs []uuid.UUID,
		preferences *models.PartnerPreferences) ([]*models.UserProfile, error)
	RecordMatchAction(ctx context.Context, userID, targetID uuid.UUID, status models.MatchStatus) error
	CheckForMutualMatch(ctx context.Context, userID1, userID2 uuid.UUID) (bool, error)
	CreateMutualMatch(ctx context.Context, userID1, userID2 uuid.UUID) error
	DeactivateMutualMatch(ctx context.Context, userID1, userID2 uuid.UUID) error
	GetMatchHistory(ctx context.Context, userID uuid.UUID, status *models.MatchStatus, limit, offset int) ([]*models.MatchHistoryItem, int, error)
	GetMutualMatches(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.MutualMatchData, int, error)
}
