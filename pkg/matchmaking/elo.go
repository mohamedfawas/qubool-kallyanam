package matchmaking

import (
	"math"
)

// CalculateExpectedOutcome computes the probability that "user" will prevail
// against an "opponent" based purely on their Elo ratings.
//
// Formula:
//
//	expected = 1 / (1 + 10^((opponentRating - userRating) / 400))
//
// Explanation:
//   - A difference of 400 points means the higher-rated player is expected to
//     score 10 times more often than the lower-rated one.
//   - Example 1: userRating=1500, opponentRating=1500
//     exponent = (1500-1500)/400 = 0.0 => 10^0 = 1 => expected = 1/(1+1) = 0.5
//     Means 50% win probability for each.
//   - Example 2: userRating=1600, opponentRating=1400
//     exponent = (1400-1600)/400 = -0.5 => 10^-0.5 ≈ 0.316 => expected ≈ 1/(1+0.316) ≈ 0.76
//     Means ~76% chance the 1600-rated user wins.
//   - Example 3: userRating=1200, opponentRating=1800
//     exponent = (1800-1200)/400 = 1.5 => 10^1.5 ≈ 31.62
//     expected ≈ 1/(1+31.62) ≈ 0.03 (3% chance)
//
// Returning a float between 0.0 and 1.0 makes it easy to compare with actual outcomes
// (win=1.0, draw=0.5, loss=0.0) when adjusting ratings.
func CalculateExpectedOutcome(userRating, opponentRating int) float64 {
	return 1.0 / (1.0 + math.Pow(10, float64(opponentRating-userRating)/400.0))
}

// UpdateRating recalculates a player's Elo rating after a match.
//
// Parameters:
//   - currentRating: the player's rating before the match (e.g., 1500).
//   - expectedOutcome: probability of winning from CalculateExpectedOutcome (e.g., 0.76).
//   - actualOutcome: 1.0 for win, 0.5 for draw, 0.0 for loss.
//   - config: RatingConfig holding KFactor and rating bounds; if KFactor ≤ 0,
//     DefaultRatingConfig is used.
//
// Steps:
// 1. Determine K-factor: how much one match can shift the rating.
// 2. Compute change = K * (actualOutcome - expectedOutcome).
//   - If you overperform (win when expected 0.76), change >0;
//   - If you underperform (lose with expected 0.76), change <0.
//
// 3. Round the change to nearest integer and add to currentRating.
// 4. Clamp the new rating to [MinRating, MaxRating].
//
// Examples:
//
//	a) Underdog win:
//	   currentRating=1400, expected=0.24, actual=1.0, K=32
//	   change = 32*(1.0-0.24)=32*0.76≈24.32→+24 ⇒ newRating=1424
//	b) Favorite loses:
//	   currentRating=1600, expected=0.76, actual=0.0, K=32
//	   change = 32*(0.0-0.76)=32*-0.76≈-24.32→-24 ⇒ newRating=1576
//	c) Draw when evenly matched:
//	   currentRating=1500, expected=0.5, actual=0.5, K=32
//	   change = 32*(0.5-0.5)=0 ⇒ newRating=1500 (no change)
func UpdateRating(currentRating int, expectedOutcome, actualOutcome float64, config RatingConfig) int {
	// If config has zero values, use defaults
	if config.KFactor <= 0 {
		config = DefaultRatingConfig
	}

	// Calculate rating change
	change := float64(config.KFactor) * (actualOutcome - expectedOutcome)

	// Round change to nearest integer
	newRating := currentRating + int(math.Round(change))

	// Ensure rating stays within allowed bounds
	if newRating < config.MinRating {
		newRating = config.MinRating
	} else if newRating > config.MaxRating {
		newRating = config.MaxRating
	}

	return newRating
}

// GetDynamicKFactor returns a K-factor that adapts to a user's experience.
//
// Rationale:
// - New or low-rated users have higher K (32) to let ratings adjust quickly.
// - As users accumulate matches or high ratings, K lowers to stabilize their rating.
//
// Logic:
// - rating > 2400 OR matchCount > 100 ⇒ K = 16 (very experienced/high-rated)
// - rating > 2000 OR matchCount > 50  ⇒ K = 24 (intermediate)
// - otherwise                          ⇒ K = 32 (new/low-rated)
//
// Examples:
// 1) rating=2500, matchCount=10 ⇒ returns 16
// 2) rating=2200, matchCount=60 ⇒ returns 24
// 3) rating=1800, matchCount=20 ⇒ returns 32
func GetDynamicKFactor(rating, matchCount int) int {
	switch {
	case rating > 2400 || matchCount > 100:
		return 16 // For high-rated or very active users
	case rating > 2000 || matchCount > 50:
		return 24 // For intermediate users
	default:
		return 32 // For new or low-rated users
	}
}

// ProcessMatchAction updates Elo ratings based on user actions
// Returns the new ratings for both users
func ProcessMatchAction(
	userRating, targetRating int,
	userMatches, targetMatches int,
	action MatchAction,
) (newUserRating, newTargetRating int) {

	// Define outcome values based on action
	var userOutcome, targetOutcome float64

	switch action {
	case MatchActionLike:
		userOutcome = 1.0   // User "wins" when they like
		targetOutcome = 0.0 // Target "loses" when they're liked
	case MatchActionDislike:
		userOutcome = 0.0   // User "loses" when they dislike
		targetOutcome = 1.0 // Target "wins" when they're disliked
	case MatchActionPass:
		userOutcome = 0.5 // Neutral for passes
		targetOutcome = 0.5
		return userRating, targetRating // No rating change for passes
	default:
		return userRating, targetRating // No change for unknown actions
	}

	// Calculate expected outcomes
	userExpected := CalculateExpectedOutcome(userRating, targetRating)
	targetExpected := CalculateExpectedOutcome(targetRating, userRating)

	// Get dynamic K-factors based on experience
	userK := GetDynamicKFactor(userRating, userMatches)
	targetK := GetDynamicKFactor(targetRating, targetMatches)

	// Update ratings
	config := DefaultRatingConfig

	config.KFactor = userK
	newUserRating = UpdateRating(userRating, userExpected, userOutcome, config)

	config.KFactor = targetK
	newTargetRating = UpdateRating(targetRating, targetExpected, targetOutcome, config)

	return newUserRating, newTargetRating
}
