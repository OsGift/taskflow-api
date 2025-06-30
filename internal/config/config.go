package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	MongoURI            string
	DBName              string
	JWTSecret           string
	Port                string
	PasswordResetSecret string

	// Email SMTP Configuration
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string

	// Cloudinary Configuration
	CloudinaryCloudName   string
	CloudinaryAPIKey      string
	CloudinaryAPISecret   string
}

// LoadConfig loads configuration from .env file or environment variables
func LoadConfig(path string) (*Config, error) {
	if err := godotenv.Load(path); err != nil {
		log.Printf("No .env file found at %s, attempting to read from environment variables. Error: %v", path, err)
	}

	return &Config{
		MongoURI:            getEnv("MONGO_URI", "mongodb://localhost:27017"),
		DBName:              getEnv("DB_NAME", "taskflow_db"),
		JWTSecret:           getEnv("JWT_SECRET", "your_very_secret_jwt_key_here_change_this_in_production"),
		Port:                getEnv("PORT", "8080"),
		PasswordResetSecret: getEnv("PASSWORD_RESET_SECRET", "another_super_secret_key_for_password_resets"),

		SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUsername: getEnv("SMTP_USERNAME", "your_email@gmail.com"),
		SMTPPassword: getEnv("SMTP_PASSWORD", "your_app_password"), // Use app password for Gmail

		CloudinaryCloudName:   getEnv("CLOUDINARY_CLOUD_NAME", ""),
		CloudinaryAPIKey:      getEnv("CLOUDINARY_API_KEY", ""),
		CloudinaryAPISecret:   getEnv("CLOUDINARY_API_SECRET", ""),
	}, nil
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
