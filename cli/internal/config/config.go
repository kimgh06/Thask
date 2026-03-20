package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	URL     string `json:"url"`
	Token   string `json:"token"`
	Team    string `json:"team"`
	Project string `json:"project"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".thask", "config.json")
}

func Load() *Config {
	c := &Config{URL: "http://localhost:7244"}

	// File
	data, err := os.ReadFile(configPath())
	if err == nil {
		_ = json.Unmarshal(data, c)
	}

	// Env overrides
	if v := os.Getenv("THASK_URL"); v != "" {
		c.URL = v
	}
	if v := os.Getenv("THASK_TOKEN"); v != "" {
		c.Token = v
	}
	if v := os.Getenv("THASK_TEAM"); v != "" {
		c.Team = v
	}
	if v := os.Getenv("THASK_PROJECT"); v != "" {
		c.Project = v
	}

	return c
}

func (c *Config) Save() error {
	p := configPath()
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0600)
}

func (c *Config) Set(key, value string) error {
	switch key {
	case "url":
		c.URL = value
	case "token":
		c.Token = value
	case "team":
		c.Team = value
	case "project":
		c.Project = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return c.Save()
}

func (c *Config) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("url is not configured")
	}
	if c.Token == "" {
		return fmt.Errorf("token is not configured (set via THASK_TOKEN or 'thask config set token <key>')")
	}
	return nil
}

func (c *Config) MaskedToken() string {
	if len(c.Token) <= 12 {
		return c.Token
	}
	return c.Token[:12] + "..."
}
