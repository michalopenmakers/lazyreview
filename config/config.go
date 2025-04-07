package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	AppName                       string
	GitLabConfig                  GitLabConfig
	GitHubConfig                  GitHubConfig
	AIModelConfig                 AIModelConfig
	MergeRequestsPollingInterval  int
	ReviewRequestsPollingInterval int
}

type GitLabConfig struct {
	Enabled    bool
	ApiToken   string
	ApiUrl     string
	ProjectIDs []string
}

type GitHubConfig struct {
	Enabled      bool
	ApiToken     string
	ApiUrl       string
	Repositories []string
}

type AIModelConfig struct {
	Model     string
	ApiKey    string
	MaxTokens int
}

func GetConfigFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return filepath.Join(homeDir, ".lazyreview_config.json")
}

func LoadConfig() *Config {
	configPath := GetConfigFilePath()
	if _, err := os.Stat(configPath); err == nil {
		file, err := os.ReadFile(configPath)
		if err == nil {
			var cfg Config
			if err = json.Unmarshal(file, &cfg); err == nil {
				if cfg.MergeRequestsPollingInterval == 0 && cfg.ReviewRequestsPollingInterval == 0 {
					legacyConfig := struct {
						PollingInterval int
					}{}
					if json.Unmarshal(file, &legacyConfig) == nil && legacyConfig.PollingInterval > 0 {
						cfg.MergeRequestsPollingInterval = legacyConfig.PollingInterval
						cfg.ReviewRequestsPollingInterval = legacyConfig.PollingInterval
					} else {
						cfg.MergeRequestsPollingInterval = 300
						cfg.ReviewRequestsPollingInterval = 120
					}
				}
				return &cfg
			}
		}
	}
	return &Config{
		AppName: "LazyReview",
		GitLabConfig: GitLabConfig{
			Enabled:    true,
			ApiToken:   "",
			ApiUrl:     "https://gitlab.com/api/v4",
			ProjectIDs: []string{},
		},
		GitHubConfig: GitHubConfig{
			Enabled:      false,
			ApiToken:     "",
			ApiUrl:       "https://api.github.com",
			Repositories: []string{},
		},
		AIModelConfig: AIModelConfig{
			Model:     "o3-mini-high",
			ApiKey:    "",
			MaxTokens: 4000,
		},
		MergeRequestsPollingInterval:  300,
		ReviewRequestsPollingInterval: 120,
	}
}

func SaveConfig(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	configPath := GetConfigFilePath()
	return os.WriteFile(configPath, data, 0644)
}
