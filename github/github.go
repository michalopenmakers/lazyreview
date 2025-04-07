package github

import (
	"github.com/michalopenmakers/lazyreview/config"
)

type PullRequest struct {
	Number     int
	Repository string
	Title      string
	HTMLURL    string
}

func GetOpenPullRequests(cfg *config.Config) ([]PullRequest, error) {
	return []PullRequest{}, nil
}

func GetAssignedPullRequests(cfg *config.Config) ([]PullRequest, error) {
	return []PullRequest{}, nil
}

func GetPullRequestChanges(cfg *config.Config, repository string, prID int) (string, error) {
	return "Code changes for PR...", nil
}

func GetRepositoryCode(cfg *config.Config, repository string) (string, error) {
	return "Entire repository code from GitHub...", nil
}
