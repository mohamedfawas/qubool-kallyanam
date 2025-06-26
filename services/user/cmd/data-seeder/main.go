package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/database/postgres"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/constants"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/models"
	"gorm.io/gorm"
)

const (
	BATCH_SIZE      = 1000   // Insert 1000 records per batch
	DEFAULT_RECORDS = 100000 // Default: 1 lakh records
)

// ... existing main() and setupDatabase() functions remain unchanged ...

func main() {
	// Get number of records from command line argument
	targetRecords := DEFAULT_RECORDS
	if len(os.Args) > 1 {
		if records, err := strconv.Atoi(os.Args[1]); err == nil {
			targetRecords = records
		}
	}

	fmt.Printf("🚀 Starting bulk data insertion for %d user profiles...\n", targetRecords)

	// Initialize database connection
	db, err := setupDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Generate and insert data
	start := time.Now()
	err = generateBulkUserProfiles(db, targetRecords)
	if err != nil {
		log.Fatalf("Failed to generate bulk data: %v", err)
	}

	duration := time.Since(start)
	recordsPerSecond := float64(targetRecords) / duration.Seconds()

	fmt.Printf("✅ Successfully inserted %d user profiles in %v\n", targetRecords, duration)
	fmt.Printf("📊 Performance: %.0f records/second\n", recordsPerSecond)
}

func setupDatabase() (*gorm.DB, error) {
	// Load configuration
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Create PostgreSQL client using the correct function and structure
	pgClient, err := postgres.NewClient(&postgres.Config{
		Host:     cfg.Database.Postgres.Host,
		Port:     fmt.Sprintf("%d", cfg.Database.Postgres.Port),
		User:     cfg.Database.Postgres.User,
		Password: cfg.Database.Postgres.Password,
		DBName:   cfg.Database.Postgres.DBName,
		SSLMode:  cfg.Database.Postgres.SSLMode,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Get the GORM DB instance
	db := pgClient.DB

	// Optimize database connection for bulk operations
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Optimize connection pool for bulk operations
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

// ... generateBulkUserProfiles() and generateUserProfileBatch() remain unchanged ...

func generateBulkUserProfiles(db *gorm.DB, totalRecords int) error {
	batches := (totalRecords + BATCH_SIZE - 1) / BATCH_SIZE // Ceiling division

	for i := 0; i < batches; i++ {
		batchSize := BATCH_SIZE
		if i == batches-1 {
			// Last batch might be smaller
			batchSize = totalRecords - (i * BATCH_SIZE)
		}

		profiles := generateUserProfileBatch(batchSize, i)

		// Insert batch using GORM
		if err := db.CreateInBatches(profiles, BATCH_SIZE).Error; err != nil {
			return fmt.Errorf("failed to insert batch %d: %w", i+1, err)
		}

		// Progress reporting
		completed := (i + 1) * BATCH_SIZE
		if completed > totalRecords {
			completed = totalRecords
		}
		progress := float64(completed) / float64(totalRecords) * 100
		fmt.Printf("📈 Progress: %d/%d (%.1f%%) - Batch %d completed\n",
			completed, totalRecords, progress, i+1)
	}

	return nil
}

func generateUserProfileBatch(batchSize int, batchNumber int) []models.UserProfile {
	profiles := make([]models.UserProfile, batchSize)

	for i := 0; i < batchSize; i++ {
		profiles[i] = generateRandomUserProfile(batchNumber, i)
	}

	return profiles
}

func generateRandomUserProfile(batchNumber, recordIndex int) models.UserProfile {
	// Initialize random seed with current time and unique identifiers
	r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(batchNumber*1000+recordIndex)))

	// Generate random date of birth (age between 18-50)
	minAge, maxAge := 18, 50
	ageYears := r.Intn(maxAge-minAge+1) + minAge
	dob := time.Now().AddDate(-ageYears, -r.Intn(12), -r.Intn(365))

	// Generate random height (150-185 cm)
	height := r.Intn(36) + 150 // 150-185 cm

	// Generate UNIQUE phone number using batch and record index
	phone := fmt.Sprintf("91%03d%07d", batchNumber, recordIndex*10000+r.Intn(10000))

	// Generate UNIQUE email using UUID for absolute uniqueness
	userUUID := uuid.New()
	emailPrefix := strings.ToLower(generateRandomName(r))
	emailPrefix = strings.ReplaceAll(emailPrefix, " ", ".")
	email := fmt.Sprintf("%s.%s@testdata.com", emailPrefix, userUUID.String()[:8])

	// Random profile picture URL
	profilePicURL := fmt.Sprintf("https://randomuser.me/api/portraits/%s/%d.jpg",
		randomChoice(r, []string{"men", "women"}), r.Intn(99))

	return models.UserProfile{
		UserID:                userUUID,
		IsBride:               r.Float32() < 0.5, // 50% chance
		FullName:              generateRandomName(r),
		Email:                 email,
		Phone:                 phone,
		DateOfBirth:           &dob,
		HeightCM:              &height,
		PhysicallyChallenged:  r.Float32() < 0.05, // 5% chance
		Community:             getRandomCommunity(r),
		MaritalStatus:         getRandomMaritalStatus(r),
		Profession:            getRandomProfession(r),
		ProfessionType:        getRandomProfessionType(r),
		HighestEducationLevel: getRandomEducationLevel(r),
		HomeDistrict:          getRandomHomeDistrict(r),
		ProfilePictureURL:     &profilePicURL,
		LastLogin:             time.Now().Add(-time.Duration(r.Intn(30*24)) * time.Hour),
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
		IsDeleted:             false,
	}
}

// ... generateRandomName() function remains unchanged ...

// Updated random data generators to use constants package:

func getRandomCommunity(r *rand.Rand) constants.Community {
	communities := []constants.Community{
		constants.CommunitySunni,
		constants.CommunityMujahid,
		constants.CommunityTabligh,
		constants.CommunityJamateIslami,
		constants.CommunityShia,
		constants.CommunityMuslim,
	}
	return communities[r.Intn(len(communities))]
}

func getRandomMaritalStatus(r *rand.Rand) constants.MaritalStatus {
	statuses := []constants.MaritalStatus{
		constants.MaritalNeverMarried,
		constants.MaritalDivorced,
		constants.MaritalNikkahDivorce,
		constants.MaritalWidowed,
	}
	return statuses[r.Intn(len(statuses))]
}

func getRandomProfession(r *rand.Rand) constants.Profession {
	professions := []constants.Profession{
		constants.ProfessionStudent,
		constants.ProfessionDoctor,
		constants.ProfessionEngineer,
		constants.ProfessionFarmer,
		constants.ProfessionTeacher,
	}
	return professions[r.Intn(len(professions))]
}

func getRandomProfessionType(r *rand.Rand) constants.ProfessionType {
	types := []constants.ProfessionType{
		constants.ProfessionTypeFullTime,
		constants.ProfessionTypePartTime,
		constants.ProfessionTypeFreelance,
		constants.ProfessionTypeSelfEmployed,
		constants.ProfessionTypeNotWorking,
	}
	return types[r.Intn(len(types))]
}

func getRandomEducationLevel(r *rand.Rand) constants.EducationLevel {
	levels := []constants.EducationLevel{
		constants.EducationLessThanHighSchool,
		constants.EducationHighSchool,
		constants.EducationHigherSecondary,
		constants.EducationUnderGraduation,
		constants.EducationPostGraduation,
	}
	return levels[r.Intn(len(levels))]
}

func getRandomHomeDistrict(r *rand.Rand) constants.HomeDistrict {
	districts := []constants.HomeDistrict{
		constants.DistrictThiruvananthapuram,
		constants.DistrictKollam,
		constants.DistrictPathanamthitta,
		constants.DistrictAlappuzha,
		constants.DistrictKottayam,
		constants.DistrictErnakulam,
		constants.DistrictThrissur,
		constants.DistrictPalakkad,
		constants.DistrictMalappuram,
		constants.DistrictKozhikode,
		constants.DistrictWayanad,
		constants.DistrictKannur,
		constants.DistrictKasaragod,
		constants.DistrictIdukki,
	}
	return districts[r.Intn(len(districts))]
}

func randomChoice(r *rand.Rand, choices []string) string {
	return choices[r.Intn(len(choices))]
}

func generateRandomName(r *rand.Rand) string {
	firstNames := []string{
		"Mohammed", "Abdul", "Ahmed", "Ali", "Hassan", "Hussein", "Ibrahim", "Ismail",
		"Khalid", "Omar", "Rashid", "Said", "Tariq", "Yusuf", "Zayn", "Amjad",
		"Fatima", "Aisha", "Khadija", "Mariam", "Zainab", "Ruqayya", "Safiya",
		"Hafsa", "Umm", "Asma", "Farah", "Layla", "Nadia", "Rania", "Sara",
		"Abdullah", "Rahman", "Hamza", "Umar", "Bilal", "Salman", "Anas", "Zaid",
		"Aminah", "Hafiza", "Samira", "Yasmin", "Rahma", "Sana", "Hiba", "Dua",
	}

	lastNames := []string{
		"Khan", "Ahmed", "Ali", "Rahman", "Hassan", "Hussein", "Ibrahim", "Malik",
		"Sheikh", "Syed", "Qureshi", "Ansari", "Hashmi", "Abbasi", "Rizvi",
		"Nair", "Menon", "Pillai", "Kumar", "Varma", "Sharma", "Patel",
		"Chowdhury", "Rahman", "Hasan", "Hossain", "Islam", "Uddin", "Alam",
	}

	return fmt.Sprintf("%s %s",
		firstNames[r.Intn(len(firstNames))],
		lastNames[r.Intn(len(lastNames))])
}
