package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const configFileName = ".gatorconfig.json"

// Config represents the JSON config file structure
type Config struct {
	DbURL          string `json:"db_url"`
	CurrentUserName     string `json:"current_user_name"`
}

// Read reads the config file from $HOME/.gatorconfig.json and returns a Config
func Read() (Config, error) {
	path, err := getConfigFilePath()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("could not read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("could not parse config file: %w", err)
	}

	return cfg, nil
}

// SetUser sets the current_user_name and writes the config back to disk
func (c *Config) SetUser(username string) error {
	c.CurrentUserName  = username
	return write(*c)
}

// --- helpers ---

func getConfigFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get home dir: %w", err)
	}
	return filepath.Join(home, configFileName), nil
}

func write(cfg Config) error {
	path, err := getConfigFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode config: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}
