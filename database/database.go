package database

import (
	"DomainManager/config"
	"DomainManager/models"
	"log"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init() {
	var err error
	DB, err = gorm.Open(sqlite.Open(config.AppConfig.DBPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	err = DB.AutoMigrate(
		&models.User{},
		&models.Domain{},
		&models.DomainPrice{},
		&models.Registrar{},
		&models.Certificate{},
		&models.NotificationChannel{},
		&models.NotificationLog{},
		&models.SystemSetting{},
	)
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	seedAdmin()
	assignLegacyRegistrarOwners()
	log.Println("database initialized successfully")
}

func seedAdmin() {
	var count int64
	DB.Model(&models.User{}).Where("username = ?", "admin").Count(&count)
	if count > 0 {
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("failed to hash admin password: %v", err)
		return
	}

	admin := models.User{
		Username: "admin",
		Password: string(hashedPassword),
		Nickname: "管理员",
		Email:    "admin@domain-manager.local",
		Role:     "admin",
	}

	if err := DB.Create(&admin).Error; err != nil {
		log.Printf("failed to create admin user: %v", err)
		return
	}

	log.Println("default admin account created (username: admin, password: 123456)")
}

// assignLegacyRegistrarOwners migrates registrar rows created before the
// per-user scoping was introduced (UserID == 0) to the first admin user.
func assignLegacyRegistrarOwners() {
	var adminUser models.User
	if err := DB.Where("role = ?", "admin").Order("id ASC").First(&adminUser).Error; err != nil {
		return
	}
	if err := DB.Model(&models.Registrar{}).Where("user_id = ?", 0).Update("user_id", adminUser.ID).Error; err != nil {
		log.Printf("failed to assign legacy registrars: %v", err)
	}
}
