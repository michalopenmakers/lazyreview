package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func (g *GitLabConfig) GetFullApiUrl() string {
	baseUrl := strings.TrimSpace(g.ApiUrl)
	if !strings.HasPrefix(baseUrl, "http://") && !strings.HasPrefix(baseUrl, "https://") {
		baseUrl = "https://" + baseUrl
	}
	baseUrl = strings.TrimSuffix(baseUrl, "/")
	if strings.HasSuffix(baseUrl, "/api/v4") {
		return baseUrl
	}
	return baseUrl + "/api/v4"
}

type GitHubConfig struct {
	Enabled      bool
	ApiToken     string
	ApiUrl       string
	Repositories []string
}

func (g *GitHubConfig) GetGitHubApiUrl() string {
	return "https://api.github.com"
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
				cfg.GitHubConfig.ApiUrl = "https://api.github.com"
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
			ApiUrl:     "https://gitlab.com",
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
	cfg.GitHubConfig.ApiUrl = "https://api.github.com"
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	configPath := GetConfigFilePath()
	return os.WriteFile(configPath, data, 0644)
}
