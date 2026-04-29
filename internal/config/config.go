package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ProjectConfig holds the local paths and repo names for a project.
type ProjectConfig struct {
	BackendPath  string `yaml:"backend_path,omitempty"`
	BackendRepo  string `yaml:"backend_repo,omitempty"`
	FrontendPath string `yaml:"frontend_path,omitempty"`
	FrontendRepo string `yaml:"frontend_repo,omitempty"`
}

// Config holds all configuration needed by the CLI.
// Priority: env vars > ~/.config/deploy-doc/config.yaml
type Config struct {
	AtlassianEmail string                    `yaml:"atlassian_email"`
	AtlassianToken string                    `yaml:"atlassian_token"`
	BaseURL        string                    `yaml:"base_url"`
	QAEmail        string                    `yaml:"qa_email,omitempty"`
	DefaultProject string                    `yaml:"default_project,omitempty"`
	Projects       map[string]*ProjectConfig `yaml:"projects,omitempty"`
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
			if cfg.DefaultProject == "" {
				cfg.DefaultProject = fileCfg.DefaultProject
			}
			if cfg.QAEmail == "" {
				cfg.QAEmail = fileCfg.QAEmail
			}
			if cfg.Projects == nil {
				cfg.Projects = fileCfg.Projects
			}
		}
	}

	if cfg.AtlassianEmail == "" || cfg.AtlassianToken == "" || cfg.BaseURL == "" {
		return nil, fmt.Errorf("configuración incompleta. Corre: gtt init")
	}

	return cfg, nil
}

// GetProject resolves which project to use.
// Priority: explicit name > default_project > nil (no project configured).
func (c *Config) GetProject(name string) (*ProjectConfig, string, error) {
	if name != "" {
		proj, ok := c.Projects[name]
		if !ok {
			return nil, "", fmt.Errorf("proyecto '%s' no encontrado. Usa: gtt project list", name)
		}
		return proj, name, nil
	}
	if c.DefaultProject != "" && c.Projects != nil {
		proj, ok := c.Projects[c.DefaultProject]
		if ok {
			return proj, c.DefaultProject, nil
		}
	}
	return nil, "", nil
}

// ConfigPath returns the path to the config file.
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "deploy-doc", "config.yaml"), nil
}

// loadFromFile reads and unmarshals the config file.
func loadFromFile() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save marshals and writes the config to disk with restricted permissions.
func Save(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	// 0600 = solo el usuario puede leer/escribir
	return os.WriteFile(path, data, 0600)
}
