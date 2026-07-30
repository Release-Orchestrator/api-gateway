// Package config provides application configuration loaded from environment variables.
package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config holds application configuration.
type Config struct {
	Port              string
	UserServiceURL    string
	OrderServiceURL   string
	PaymentServiceURL string
}

// Load reads configuration from environment variables.
func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		Port:              getEnv("PORT", "8080"),
		UserServiceURL:    getEnv("USER_SERVICE_URL", "http://localhost:8081"),
		OrderServiceURL:   getEnv("ORDER_SERVICE_URL", "http://localhost:8082"),
		PaymentServiceURL: getEnv("PAYMENT_SERVICE_URL", "http://localhost:8083"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
