package services

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"DomainManager/config"
	"DomainManager/database"
	"DomainManager/models"
)

func TestIsScheduledTaskDue(t *testing.T) {
	now := time.Now()

	// Never ran -> due.
	if !IsScheduledTaskDue(models.ScheduledTask{}, now) {
		t.Fatal("task without last_run_at should be due")
	}

	// Ran just now with a daily interval -> not due.
	last := now.Add(-time.Minute)
	if IsScheduledTaskDue(models.ScheduledTask{LastRunAt: &last, IntervalMinutes: 1440}, now) {
		t.Fatal("task run a minute ago should not be due yet")
	}

	// Interval elapsed -> due.
	last = now.Add(-(24*time.Hour + time.Minute))
	if !IsScheduledTaskDue(models.ScheduledTask{LastRunAt: &last, IntervalMinutes: 1440}, now) {
		t.Fatal("task with elapsed interval should be due")
	}

	// Zero interval falls back to the default.
	if !IsScheduledTaskDue(models.ScheduledTask{LastRunAt: &last, IntervalMinutes: 0}, now) {
		t.Fatal("zero interval should fall back to the default")
	}
}

func TestBuildSystemInfoReport(t *testing.T) {
	config.AppConfig = &config.Config{
		DBPath: filepath.Join(t.TempDir(), "test.db"),
		JWTKey: "test-jwt",
	}
	database.Init()
	t.Cleanup(func() {
		if sqlDB, err := database.DB.DB(); err == nil {
			sqlDB.Close()
		}
	})

	database.DB.Create(&models.Domain{UserID: 1, Name: "example.com", Status: "active"})
	database.DB.Create(&models.Certificate{UserID: 1, Domain: "example.com", Status: "active"})

	report := BuildSystemInfoReport()
	if !strings.Contains(report, "域名总数：1") {
		t.Fatalf("report missing domain count: %s", report)
	}
	if !strings.Contains(report, "证书总数：1") {
		t.Fatalf("report missing certificate count: %s", report)
	}
	if !strings.Contains(report, config.Version) {
		t.Fatalf("report missing version: %s", report)
	}
}
