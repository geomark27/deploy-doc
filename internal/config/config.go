package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ProjectConfig holds the local paths and repo names for a project.
type ProjectConfig struct {
	BackendPath        string `yaml:"backend_path,omitempty"`
	BackendRepo        string `yaml:"backend_repo,omitempty"`
	FrontendPath       string `yaml:"frontend_path,omitempty"`
	FrontendRepo       string `yaml:"frontend_repo,omitempty"`
	VCSHost            string `yaml:"vcs_host,omitempty"`
	VCSOrg             string `yaml:"vcs_org,omitempty"`
	ConfluenceSpaceKey string `yaml:"confluence_space_key,omitempty"`
}

// Config holds all configuration needed by the CLI.
// Priority: env vars > ~/.config/gtt/config.yaml
type Config struct {
	AtlassianEmail     string                    `yaml:"atlassian_email"`
	AtlassianToken     string                    `yaml:"atlassian_token"`
	BaseURL            string                    `yaml:"base_url"`
	QAEmail            string                    `yaml:"qa_email,omitempty"`
	ConfluenceSpaceKey string                    `yaml:"confluence_space_key,omitempty"`
	DefaultProject     string                    `yaml:"default_project,omitempty"`
	Projects           map[string]*ProjectConfig `yaml:"projects,omitempty"`
}

// Load loads config following priority: env vars > config file.
func Load() (*Config, error) {
	cfg := &Config{
		AtlassianEmail:     os.Getenv("ATLASSIAN_EMAIL"),
		AtlassianToken:     os.Getenv("ATLASSIAN_TOKEN"),
		BaseURL:            os.Getenv("ATLASSIAN_BASE_URL"),
		ConfluenceSpaceKey: os.Getenv("CONFLUENCE_SPACE_KEY"),
	}

	// Always merge from file so all fields are loaded regardless of env vars.
	if fileCfg, err := loadFromFile(); err == nil {
		if cfg.AtlassianEmail == "" {
			cfg.AtlassianEmail = fileCfg.AtlassianEmail
		}
		if cfg.AtlassianToken == "" {
			cfg.AtlassianToken = fileCfg.AtlassianToken
		}
		if cfg.BaseURL == "" {
			cfg.BaseURL = fileCfg.BaseURL
		}
		if cfg.QAEmail == "" {
			cfg.QAEmail = fileCfg.QAEmail
		}
		if cfg.ConfluenceSpaceKey == "" {
			cfg.ConfluenceSpaceKey = fileCfg.ConfluenceSpaceKey
		}
		if cfg.DefaultProject == "" {
			cfg.DefaultProject = fileCfg.DefaultProject
		}
		if cfg.Projects == nil {
			cfg.Projects = fileCfg.Projects
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
	return filepath.Join(home, ".config", "gtt", "config.yaml"), nil
}

// MigrateIfNeeded copies the legacy ~/.config/deploy-doc/config.yaml to
// ~/.config/gtt/config.yaml the first time a user runs v1.2.0+.
// The old directory is removed only after a successful copy.
func MigrateIfNeeded() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	newPath := filepath.Join(home, ".config", "gtt", "config.yaml")
	oldPath := filepath.Join(home, ".config", "deploy-doc", "config.yaml")
	oldDir := filepath.Join(home, ".config", "deploy-doc")

	// Already migrated or fresh install — nothing to do.
	if _, err := os.Stat(newPath); err == nil {
		return
	}

	// No legacy config — fresh install, nothing to migrate.
	if _, err := os.Stat(oldPath); err != nil {
		return
	}

	if err := os.MkdirAll(filepath.Join(home, ".config", "gtt"), 0700); err != nil {
		return
	}

	data, err := os.ReadFile(oldPath)
	if err != nil {
		return
	}
	if err := os.WriteFile(newPath, data, 0600); err != nil {
		return
	}

	// Remove old directory only after successful copy.
	os.RemoveAll(oldDir)

	fmt.Println("✓ Configuración migrada a ~/.config/gtt/config.yaml")
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
