package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	LogDir             string  `json:"log_dir"`
	DBPath             string  `json:"db_path"`
	Port               int     `json:"port"`
	MonthlyBudgetUSD   float64 `json:"monthly_budget_usd"`
	OpenBrowserOnServe bool    `json:"open_browser_on_serve"`
}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		LogDir:             filepath.Join(home, ".claude", "projects"),
		DBPath:             filepath.Join(home, ".cctrack", "cctrack.db"),
		Port:               7432,
		MonthlyBudgetUSD:   0,
		OpenBrowserOnServe: true,
	}
}

func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cctrack")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.json")
}

func Load() (*Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Save() error {
	if err := os.MkdirAll(ConfigDir(), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), data, 0600)
}
