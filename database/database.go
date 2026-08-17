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
		&models.UserOAuthBinding{},
	)
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	seedAdmin()
	assignLegacyRegistrarOwners()
	migrateUserRoleGroups()
	migrateOauthBindings()
	log.Println("database initialized successfully")
}

// LoadSettingsIntoConfig loads runtime settings from the system_settings table
// into config.AppConfig. Missing settings are seeded with the current
// environment-derived defaults so the admin UI always shows a value.
func LoadSettingsIntoConfig() {
	if DB == nil {
		return
	}

	var rows []models.SystemSetting
	if err := DB.Find(&rows).Error; err != nil {
		log.Printf("failed to load settings from database: %v", err)
		return
	}

	values := map[string]string{}
	for _, r := range rows {
		values[r.Key] = r.Value
	}

	// Seed defaults for settings that do not exist yet (first startup).
	for _, key := range config.DBSettingKeys {
		if _, ok := values[key]; ok {
			continue
		}
		def := config.DefaultDBSettingValue(key)
		if def == "" {
			continue
		}
		if err := DB.Create(&models.SystemSetting{Key: key, Value: def}).Error; err != nil {
			log.Printf("failed to seed setting %s: %v", key, err)
		} else {
			values[key] = def
		}
	}

	config.ApplyDBSettings(values)
}

// migrateUserRoleGroups backfills the new role_group column from the legacy
// role column (removed from the model). It copies role -> role_group once and
// then clears the legacy role column so old values can never resurrect
// privileges (e.g. re-promoting a demoted admin on the next startup).
// Fresh databases do not have the legacy role column and skip this step.
func migrateUserRoleGroups() {
	var cols []struct {
		Name string
	}
	if err := DB.Raw("PRAGMA table_info(users)").Scan(&cols).Error; err != nil {
		log.Printf("failed to inspect users table: %v", err)
		return
	}
	hasRoleColumn := false
	for _, c := range cols {
		if c.Name == "role" {
			hasRoleColumn = true
			break
		}
	}
	if !hasRoleColumn {
		return
	}

	if err := DB.Exec("UPDATE users SET role_group = role WHERE role != '' AND role_group != role").Error; err != nil {
		log.Printf("failed to migrate user role groups: %v", err)
		return
	}
	if err := DB.Exec("UPDATE users SET role = '' WHERE role != ''").Error; err != nil {
		log.Printf("failed to clear legacy user roles: %v", err)
	}
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
		Username:  "admin",
		Password:  string(hashedPassword),
		Nickname:  "管理员",
		Email:     "admin@domain-manager.local",
		RoleGroup: "admin",
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
	if err := DB.Where("role_group = ?", "admin").Order("id ASC").First(&adminUser).Error; err != nil {
		return
	}
	if err := DB.Model(&models.Registrar{}).Where("user_id = ?", 0).Update("user_id", adminUser.ID).Error; err != nil {
		log.Printf("failed to assign legacy registrars: %v", err)
	}
}

// migrateOauthBindings backfills user_oauth_bindings rows from the legacy
// oauth_provider / oauth_openid columns on users so third-party logins keep
// resolving to the same account after the binding table was introduced.
func migrateOauthBindings() {
	var users []models.User
	if err := DB.Where("oauth_provider != '' AND oauth_openid != ''").Find(&users).Error; err != nil {
		log.Printf("failed to load users for oauth binding migration: %v", err)
		return
	}
	for _, u := range users {
		var count int64
		DB.Model(&models.UserOAuthBinding{}).Where("user_id = ? AND provider = ? AND openid = ?", u.ID, u.OauthProvider, u.OauthOpenID).Count(&count)
		if count > 0 {
			continue
		}
		binding := models.UserOAuthBinding{
			UserID:   u.ID,
			Provider: u.OauthProvider,
			OpenID:   u.OauthOpenID,
			Nickname: u.Nickname,
			Avatar:   u.OauthAvatar,
		}
		if err := DB.Create(&binding).Error; err != nil {
			log.Printf("failed to migrate oauth binding for user %d: %v", u.ID, err)
		}
	}
}
