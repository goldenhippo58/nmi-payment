package config

import (
    "log"
    "os"
    "strconv"

    "github.com/joho/godotenv"
)

// Config holds all configuration values
type Config struct {
    APIKey     string
    APIBaseURL string
    DebugMode  bool
    Port       string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
    // Load .env file if it exists
    err := godotenv.Load()
    if err != nil {
        log.Printf("Warning: .env file not found: %v", err)
    }

    // Set default values
    config := &Config{
        Port: "8080",
    }

    // Load from environment variables
    if apiKey := os.Getenv("NMI_API_KEY"); apiKey != "" {
        config.APIKey = apiKey
    } else {
        log.Fatal("NMI_API_KEY environment variable is required")
    }

    if apiURL := os.Getenv("API_URL"); apiURL != "" {
        config.APIBaseURL = apiURL
    } else {
        config.APIBaseURL = "https://secure.nmi.com/api/transact.php"
    }

    config.DebugMode, _ = strconv.ParseBool(os.Getenv("DEBUG_MODE"))

    // Validate required configurations
    if err := config.validate(); err != nil {
        log.Fatalf("Configuration error: %v", err)
    }

    return config
}

// validate checks if all required configuration values are present
func (c *Config) validate() error {
    if c.APIKey == "" {
        return fmt.Errorf("NMI_API_KEY is required")
    }
    if c.APIBaseURL == "" {
        return fmt.Errorf("API_URL is required")
    }
    return nil
}