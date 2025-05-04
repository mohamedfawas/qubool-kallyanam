package indianstandardtime

import (
	"time"
)

// ISTLocation represents Indian Standard Time (UTC+5:30)
var ISTLocation *time.Location

func init() {
	// Initialize IST location once during package initialization
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		// Fallback to fixed offset if timezone data not available
		ISTLocation = time.FixedZone("IST", 5*60*60+30*60) // UTC+5:30
		return
	}
	ISTLocation = loc
}

// Now returns current time in Indian Standard Time
func Now() time.Time {
	return time.Now().In(ISTLocation)
}

// ParseInIST parses a time string in IST location
func ParseInIST(layout, value string) (time.Time, error) {
	return time.ParseInLocation(layout, value, ISTLocation)
}

// FormatTime formats a time in IST using RFC3339 format
func FormatTime(t time.Time) string {
	return t.In(ISTLocation).Format(time.RFC3339)
}
