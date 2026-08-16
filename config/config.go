package config

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	JWTKey             string
	DBPath             string
	WhoisAPIURL        string
	ICPAPIURL          string
	DigitalPlatRDAPURL string
}

var AppConfig *Config

func Load() {
	godotenv.Load()

	AppConfig = &Config{
		Port:               getEnv("PORT", "8080"),
		JWTKey:             os.Getenv("JWT_KEY"),
		DBPath:             getEnv("DB_PATH", "domain_manager.db"),
		WhoisAPIURL:        getEnv("WHOIS_API_URL", "https://who.zmh.me"),
		ICPAPIURL:          getEnv("ICP_API_URL", "http://127.0.0.1:16181"),
		DigitalPlatRDAPURL: getEnv("DIGITALPLAT_RDAP_URL", "https://rdap.digitalplat.org"),
	}

	if AppConfig.JWTKey == "" {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			log.Fatalf("failed to generate JWT key: %v", err)
		}
		AppConfig.JWTKey = hex.EncodeToString(key)
		log.Println("WARNING: JWT_KEY is not set; generated a random signing key. Tokens will be invalidated on every restart. Set JWT_KEY in your environment for stable sessions.")
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
