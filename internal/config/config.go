package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds all configuration needed by the CLI.
// Priority: env vars > ~/.config/deploy-doc/config.yaml
type Config struct {
	AtlassianEmail string
	AtlassianToken string
	BaseURL        string
}

// Load loads config following priority: env vars > config file.
func Load() (*Config, error) {
	cfg := &Config{
		AtlassianEmail: os.Getenv("ATLASSIAN_EMAIL"),
		AtlassianToken: os.Getenv("ATLASSIAN_TOKEN"),
		BaseURL:        os.Getenv("ATLASSIAN_BASE_URL"),
	}

	if cfg.AtlassianEmail == "" || cfg.AtlassianToken == "" || cfg.BaseURL == "" {
		fileCfg, err := loadFromFile()
		if err == nil {
			if cfg.AtlassianEmail == "" {
				cfg.AtlassianEmail = fileCfg.AtlassianEmail
			}
			if cfg.AtlassianToken == "" {
				cfg.AtlassianToken = fileCfg.AtlassianToken
			}
			if cfg.BaseURL == "" {
				cfg.BaseURL = fileCfg.BaseURL
			}
		}
	}

	if cfg.AtlassianEmail == "" || cfg.AtlassianToken == "" || cfg.BaseURL == "" {
		return nil, fmt.Errorf("configuración incompleta. Corre: deploy-doc init")
	}

	return cfg, nil
}

// ConfigPath returns the path to the config file.
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "deploy-doc", "config.yaml"), nil
}

// loadFromFile reads the config file (simple key: value format).
func loadFromFile() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := &Config{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "atlassian_email":
			cfg.AtlassianEmail = val
		case "atlassian_token":
			cfg.AtlassianToken = val
		case "base_url":
			cfg.BaseURL = val
		}
	}
	return cfg, scanner.Err()
}

// Save writes the config to the config file.
func Save(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	content := fmt.Sprintf(
		"atlassian_email: %s\natlassian_token: %s\nbase_url: %s\n",
		cfg.AtlassianEmail, cfg.AtlassianToken, cfg.BaseURL,
	)
	// 0600 = solo el usuario puede leer/escribir
	return os.WriteFile(path, []byte(content), 0600)
}
