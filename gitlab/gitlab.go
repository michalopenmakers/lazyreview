package gitlab

import (
	"github.com/michalopenmakers/lazyreview/config"
)

type MergeRequest struct {
	IID       int
	ProjectID int
	Title     string
	WebURL    string
}

func GetOpenMergeRequests(cfg *config.Config) ([]MergeRequest, error) {
	return []MergeRequest{}, nil
}

func GetAssignedMergeRequests(cfg *config.Config) ([]MergeRequest, error) {
	return []MergeRequest{}, nil
}

func GetMergeRequestChanges(cfg *config.Config, projectID string, mrID int) (string, error) {
	return "Code changes for MR...", nil
}

func GetProjectCode(cfg *config.Config, projectID string) (string, error) {
	return "Entire project code from GitLab...", nil
}
