package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	JWTKey      string
	DBPath      string
	WhoisAPIURL string
	ICPAPIURL   string
}

var AppConfig *Config

func Load() {
	godotenv.Load()

	AppConfig = &Config{
		Port:        getEnv("PORT", "8080"),
		JWTKey:      getEnv("JWT_KEY", "domain-manager-secret-key-2026"),
		DBPath:      getEnv("DB_PATH", "domain_manager.db"),
		WhoisAPIURL: getEnv("WHOIS_API_URL", "https://who.zmh.me"),
		ICPAPIURL:   getEnv("ICP_API_URL", "http://127.0.0.1:16181"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
