package database

import (
	"log"
	"strings"

	"pet-hotel/internal/config"
	"pet-hotel/internal/models"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDB(cfg *config.Config) *gorm.DB {
	var db *gorm.DB
	var err error

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	if cfg.DatabaseURL != "" && strings.HasPrefix(cfg.DatabaseURL, "postgres") {
		log.Println("Connecting to PostgreSQL (Cloud/Local):", cfg.DatabaseURL)
		db, err = gorm.Open(postgres.Open(cfg.DatabaseURL), gormConfig)
	} else {
		log.Println("DATABASE_URL not set or not Postgres. Falling back to SQLite: pethotel.db")
		db, err = gorm.Open(sqlite.Open("pethotel.db"), gormConfig)
	}

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto Migration
	log.Println("Running Database Auto-Migrations...")
	err = db.AutoMigrate(
		&models.User{},
		&models.Pet{},
		&models.Room{},
		&models.Booking{},
	)
	if err != nil {
		log.Fatalf("Failed to auto-migrate database: %v", err)
	}

	// Seed Initial Data
	seedInitialData(db)

	return db
}

func seedInitialData(db *gorm.DB) {
	// Seed Admin User
	var adminCount int64
	db.Model(&models.User{}).Where("email = ?", "admin@pethotel.com").Count(&adminCount)
	if adminCount == 0 {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123456"), bcrypt.DefaultCost)
		admin := models.User{
			Name:         "Hotel Admin",
			Email:        "admin@pethotel.com",
			PasswordHash: string(hashedPassword),
			Role:         models.RoleAdmin,
		}
		db.Create(&admin)
		log.Println("Seeded Admin: admin@pethotel.com / admin123456")
	}

	// Seed Demo User
	var userCount int64
	db.Model(&models.User{}).Where("email = ?", "user@pethotel.com").Count(&userCount)
	if userCount == 0 {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("user123456"), bcrypt.DefaultCost)
		user := models.User{
			Name:         "Somchai PetLover",
			Email:        "user@pethotel.com",
			PasswordHash: string(hashedPassword),
			Role:         models.RoleUser,
		}
		db.Create(&user)
		log.Println("Seeded User: user@pethotel.com / user123456")
	}

	// Seed Rooms
	var roomCount int64
	db.Model(&models.Room{}).Count(&roomCount)
	if roomCount == 0 {
		rooms := []models.Room{
			{
				RoomNumber:    "ST-101",
				RoomType:      "Standard Cozy Room",
				PricePerNight: 350.00,
				Status:        models.RoomStatusAvailable,
				Description:   "ห้องขนาดกะทัดรัด เหมาะสำหรับน้องหมาน้องแมวพันธุ์เล็ก พร้อมเบาะนอนนุ่มสบาย",
			},
			{
				RoomNumber:    "DL-201",
				RoomType:      "Deluxe Garden View",
				PricePerNight: 550.00,
				Status:        models.RoomStatusAvailable,
				Description:   "ห้องขนาดกลาง วิวสวน มีของเล่นและพื้นที่วิ่งเล่นส่วนตัว",
			},
			{
				RoomNumber:    "VP-301",
				RoomType:      "VIP Presidential Suite",
				PricePerNight: 950.00,
				Status:        models.RoomStatusAvailable,
				Description:   "ห้องสูทปรับอากาศตลอด 24 ชม. พร้อมกล้องวงจรปิดดูผ่านมือถือและห้องน้ำส่วนตัว",
			},
			{
				RoomNumber:    "MN-401",
				RoomType:      "Cat Condo Villa",
				PricePerNight: 450.00,
				Status:        models.RoomStatusMaintenance,
				Description:   "คอนโดแมวสูง 3 ชั้น พร้อมเสาลับเล็บ (กำลังปรับปรุงประจำสัปดาห์)",
			},
		}
		db.Create(&rooms)
		log.Println("Seeded 4 default pet hotel rooms")
	}
}
