package database

import (
	"path/filepath"
	"testing"

	"DomainManager/config"
	"DomainManager/models"
)

func TestMigrateUserRoleGroups(t *testing.T) {
	config.AppConfig = &config.Config{DBPath: filepath.Join(t.TempDir(), "test.db"), JWTKey: "test"}
	Init()
	t.Cleanup(func() {
		if sqlDB, err := DB.DB(); err == nil {
			sqlDB.Close()
		}
	})

	// Simulate a legacy database: add the old role column, then create users.
	if err := DB.Exec("ALTER TABLE users ADD COLUMN role TEXT DEFAULT ''").Error; err != nil {
		t.Fatal(err)
	}

	admin := models.User{Username: "legacy_admin", Email: "legacy_admin@example.com", Password: "x", RoleGroup: "user"}
	if err := DB.Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	normal := models.User{Username: "legacy_user", Email: "legacy_user@example.com", Password: "x", RoleGroup: "user"}
	if err := DB.Create(&normal).Error; err != nil {
		t.Fatal(err)
	}
	if err := DB.Exec("UPDATE users SET role = 'admin' WHERE id = ?", admin.ID).Error; err != nil {
		t.Fatal(err)
	}

	migrateUserRoleGroups()

	var gotAdmin, gotNormal models.User
	if err := DB.First(&gotAdmin, admin.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotAdmin.RoleGroup != "admin" {
		t.Fatalf("legacy admin role_group = %q, want admin", gotAdmin.RoleGroup)
	}
	if err := DB.First(&gotNormal, normal.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotNormal.RoleGroup != "user" {
		t.Fatalf("legacy user role_group = %q, want user", gotNormal.RoleGroup)
	}

	// Legacy role column is cleared so it cannot resurrect privileges.
	var count int64
	if err := DB.Model(&models.User{}).Where("role != ''").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("legacy role column still has %d non-empty values", count)
	}
}

func TestMigrateUserRoleGroupsFreshDB(t *testing.T) {
	config.AppConfig = &config.Config{DBPath: filepath.Join(t.TempDir(), "test.db"), JWTKey: "test"}
	Init()
	t.Cleanup(func() {
		if sqlDB, err := DB.DB(); err == nil {
			sqlDB.Close()
		}
	})

	// Fresh databases have no legacy role column: migration must be a no-op.
	migrateUserRoleGroups()

	var admin models.User
	if err := DB.Where("username = ?", "admin").First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if admin.RoleGroup != "admin" {
		t.Fatalf("seeded admin role_group = %q, want admin", admin.RoleGroup)
	}
}
