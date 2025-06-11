package matchmaking

// MatchReason represents a specific reason why profiles match
type MatchReason string

// MatchScore holds the final compatibility score (0-100) and
// a list of reasons explaining why the score was given.
// Example: {Score: 75.0, Reasons: ["Age close to preferences", "Education matches preference"]}
type MatchScore struct {
	Score   float64 // Compatibility percentage (0.0 – 100.0)
	Reasons []string
}

// ProfileData contains minimal user profile data needed for matchmaking
type ProfileData struct {
	// Demographic information
	IsBride              bool
	Age                  int
	HeightCM             int
	PhysicallyChallenged bool

	// Background attributes
	Community             string
	MaritalStatus         string
	Profession            string
	ProfessionType        string
	HighestEducationLevel string
	HomeDistrict          string
}

// PreferenceData contains user's partner preferences
type PreferenceData struct {
	// Age preferences
	MinAgeYears *int
	MaxAgeYears *int

	// Height preferences
	MinHeightCM *int
	MaxHeightCM *int

	// Other preferences
	AcceptPhysicallyChallenged bool
	PreferredCommunities       []string
	PreferredMaritalStatus     []string
	PreferredProfessions       []string
	PreferredProfessionTypes   []string
	PreferredEducationLevels   []string
	PreferredHomeDistricts     []string
}

// MatchAction defines what action the user took on a potential match.
type MatchAction string

const (
	MatchActionLike    MatchAction = "liked"
	MatchActionDislike MatchAction = "disliked"
	MatchActionPass    MatchAction = "passed"
)

// RatingConfig holds parameters for Elo rating calculations
type RatingConfig struct {
	InitialRating int // Starting rating for new profiles (e.g., 1500)
	KFactor       int // How strongly each match affects rating
	MinRating     int // Lowest possible rating (e.g., 1000)
	MaxRating     int // Highest possible rating (e.g., 3000)
}

// DefaultRatingConfig provides standard values for Elo ratings
var DefaultRatingConfig = RatingConfig{
	InitialRating: 1500,
	KFactor:       32,
	MinRating:     1000,
	MaxRating:     3000,
}
