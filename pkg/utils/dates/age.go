package dates

import (
	"fmt"
	"math"
	"time"
)

// IsLeapYear checks if the provided year is a leap year
func IsLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// CalculateAge calculates age from the given birth date to current date
func CalculateAge(birthDate time.Time) int {
	now := time.Now()
	years := now.Year() - birthDate.Year()

	// Adjust age if birthday hasn't occurred yet this year
	if now.Month() < birthDate.Month() || (now.Month() == birthDate.Month() && now.Day() < birthDate.Day()) {
		years--
	}

	return years
}

// CalculateAgeAt calculates age at a specific reference date
func CalculateAgeAt(birthDate, referenceDate time.Time) int {
	years := referenceDate.Year() - birthDate.Year()

	// Adjust age if birthday hasn't occurred yet in the reference year
	if referenceDate.Month() < birthDate.Month() || (referenceDate.Month() == birthDate.Month() && referenceDate.Day() < birthDate.Day()) {
		years--
	}

	return years
}

// AgeRange represents a min-max age range
type AgeRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// IsInRange checks if the given age is within the specified range
func (r AgeRange) IsInRange(age int) bool {
	return age >= r.Min && age <= r.Max
}

// GetBirthYearRange returns the birth year range based on the current date
// and the specified age range
func (r AgeRange) GetBirthYearRange() (minYear, maxYear int) {
	currentYear := time.Now().Year()
	// Min age corresponds to max birth year and vice versa
	return currentYear - r.Max, currentYear - r.Min
}

// FormatBirthDateRange returns a formatted string representing the birth date range
func (r AgeRange) FormatBirthDateRange() string {
	minYear, maxYear := r.GetBirthYearRange()
	return fmt.Sprintf("Born between %d and %d", minYear, maxYear)
}

// CalculateHijriAge calculates age according to the Islamic Hijri calendar
// This is a simplified version that approximates Hijri years
func CalculateHijriAge(birthDate time.Time) int {
	gregorianAge := CalculateAge(birthDate)

	// Hijri years are approximately 3% shorter than Gregorian years
	// 100 Gregorian years ≈ 103 Hijri years
	hijriAge := float64(gregorianAge) * 1.03

	return int(math.Floor(hijriAge))
}
