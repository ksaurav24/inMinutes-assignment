package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	Port        string
}

func LoadEnv() (Config, error) {
	if err := loadDotEnv(); err != nil {
		return Config{}, err
	}

	config := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port:        os.Getenv("PORT"),
	}
	if config.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if config.Port == "" {
		config.Port = "8080"
	}
	return config, nil
}

func loadDotEnv() error {
	for _, path := range []string{".env", "../../.env"} {
		if err := godotenv.Load(path); err == nil {
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("load %s: %w", path, err)
		}
	}
	return nil
}
