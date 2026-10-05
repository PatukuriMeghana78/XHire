package config

import "os"

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

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
