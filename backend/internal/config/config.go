package config

import (
	"errors"
	"fmt"
	"os"
)

// minJWTSecretLength is the minimum accepted length (in bytes) for JWT_SECRET.
const minJWTSecretLength = 32

type Config struct {
	DBUrl string
	Port  string
}

func Load() Config {
	return Config{
		DBUrl: getEnv(
			"DB_URL",
			"host=localhost user=postgres password=postgres1234 dbname=job_tracker port=5432 sslmode=disable",
		),
		Port: getEnv("PORT", "8080"),
	}
}

// JWTSecret returns the signing key for JWTs from the JWT_SECRET environment
// variable. It is the single source of truth shared by token generation and
// validation. It reads the env at call time so it works after godotenv.Load().
func JWTSecret() []byte {
	return []byte(os.Getenv("JWT_SECRET"))
}

// ValidateJWTSecret fails safely when JWT_SECRET is missing, too short, or set
// to a known insecure default. Call it once at startup and abort on error.
func ValidateJWTSecret() error {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return errors.New("JWT_SECRET is not set")
	}
	if len(secret) < minJWTSecretLength {
		return fmt.Errorf("JWT_SECRET must be at least %d characters long", minJWTSecretLength)
	}
	switch secret {
	case "supersecretkey", "changeme", "secret", "password":
		return errors.New("JWT_SECRET is set to a known insecure default value")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
