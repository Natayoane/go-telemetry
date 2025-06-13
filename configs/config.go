package configs

import (
	"os"
)

type Config struct {
	OpenWeatherMapAPIKey string
	ServiceBURL          string
}

func LoadConfig() Config {
	return Config{
		OpenWeatherMapAPIKey: getEnv("OPENWEATHERMAP_API_KEY", ""),
		ServiceBURL:          getEnv("SERVICE_B_URL", "http://localhost:8081"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
