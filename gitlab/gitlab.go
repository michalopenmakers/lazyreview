package gitlab

import (
	"encoding/json"
	"fmt"
	"github.com/michalopenmakers/lazyreview/config"
	"io"
	"net/http"
	"time"

	"github.com/michalopenmakers/lazyreview/logger"
)

type MergeRequest struct {
	IID       int
	ProjectID int
	Title     string
	WebURL    string
}

func GetMergeRequestChanges(cfg *config.Config, projectID string, mrID int) (string, error) {
	logger.Log(fmt.Sprintf("Getting changes for MR #%d in project %s", mrID, projectID))

	apiUrl := cfg.GitLabConfig.GetFullApiUrl()
	url := fmt.Sprintf("%s/projects/%s/merge_requests/%d/changes", apiUrl, projectID, mrID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Log(fmt.Sprintf("Error creating request for GitLab MR changes: %v", err))
		return "", err
	}

	req.Header.Set("PRIVATE-TOKEN", cfg.GitLabConfig.ApiToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log(fmt.Sprintf("Error connecting to GitLab API (%s): %v", apiUrl, err))
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("GitLab API responded with status code %d: %s", resp.StatusCode, string(body))
		logger.Log(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	var response struct {
		Changes []struct {
			OldPath     string `json:"old_path"`
			NewPath     string `json:"new_path"`
			Diff        string `json:"diff"`
			RenamedFile bool   `json:"renamed_file"`
		} `json:"changes"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		logger.Log(fmt.Sprintf("Error decoding GitLab response: %v", err))
		return "", err
	}

	var combinedDiff string
	for _, change := range response.Changes {
		fileHeader := fmt.Sprintf("--- %s\n+++ %s\n", change.OldPath, change.NewPath)
		combinedDiff += fileHeader + change.Diff + "\n\n"
	}

	logger.Log(fmt.Sprintf("Successfully fetched changes for MR #%d, total size: %d bytes", mrID, len(combinedDiff)))
	return combinedDiff, nil
}

func GetProjectCode(projectID string) (string, error) {
	logger.Log(fmt.Sprintf("Getting entire code for project %s", projectID))
	return fmt.Sprintf("Project code for project ID %s is too large to include directly. Review will focus on changes only.", projectID), nil
}

func GetMergeRequestsToReview(cfg *config.Config) ([]MergeRequest, error) {
	logger.Log("Fetching GitLab merge requests assigned for review")

	apiUrl := cfg.GitLabConfig.GetFullApiUrl()
	url := fmt.Sprintf("%s/merge_requests?reviewer_username=michal.zuchowski&state=opened", apiUrl)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Log(fmt.Sprintf("Error creating request for GitLab MRs: %v", err))
		return nil, err
	}

	req.Header.Set("PRIVATE-TOKEN", cfg.GitLabConfig.ApiToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log(fmt.Sprintf("Error connecting to GitLab API (%s): %v", apiUrl, err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("GitLab API responded with status code %d: %s", resp.StatusCode, string(body))
		logger.Log(errMsg)
		return nil, fmt.Errorf(errMsg)
	}

	var mergeRequests []MergeRequest
	if err := json.NewDecoder(resp.Body).Decode(&mergeRequests); err != nil {
		logger.Log(fmt.Sprintf("Error decoding GitLab response: %v", err))
		return nil, err
	}

	logger.Log(fmt.Sprintf("Successfully fetched %d merge requests for review", len(mergeRequests)))
	return mergeRequests, nil
}

func GetChangesBetweenCommits(cfg *config.Config, projectID, oldCommit, newCommit string) (string, error) {
	logger.Log(fmt.Sprintf("Getting changes between commits %s and %s for project %s", oldCommit, newCommit, projectID))

	if oldCommit == "" {
		return GetProjectCode(projectID)
	}

	apiUrl := cfg.GitLabConfig.GetFullApiUrl()
	url := fmt.Sprintf("%s/projects/%s/repository/compare?from=%s&to=%s", apiUrl, projectID, oldCommit, newCommit)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Log(fmt.Sprintf("Error creating request for GitLab diff: %v", err))
		return "", err
	}

	req.Header.Set("PRIVATE-TOKEN", cfg.GitLabConfig.ApiToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log(fmt.Sprintf("Error connecting to GitLab API (%s): %v", apiUrl, err))
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("GitLab API responded with status code %d: %s", resp.StatusCode, string(body))
		logger.Log(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	var compareResult struct {
		Diffs []struct {
			Diff        string `json:"diff"`
			OldPath     string `json:"old_path"`
			NewPath     string `json:"new_path"`
			RenamedFile bool   `json:"renamed_file"`
		} `json:"diffs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&compareResult); err != nil {
		logger.Log(fmt.Sprintf("Error decoding GitLab compare response: %v", err))
		return "", err
	}

	var combinedDiff string
	for _, diff := range compareResult.Diffs {
		fileHeader := fmt.Sprintf("--- %s\n+++ %s\n", diff.OldPath, diff.NewPath)
		combinedDiff += fileHeader + diff.Diff + "\n\n"
	}

	logger.Log(fmt.Sprintf("Successfully fetched changes between commits, total size: %d bytes", len(combinedDiff)))
	return combinedDiff, nil
}

func GetCurrentCommit(cfg *config.Config, projectID string, mrID int) (string, error) {
	logger.Log(fmt.Sprintf("Getting current commit for MR #%d in project %s", mrID, projectID))

	apiUrl := cfg.GitLabConfig.GetFullApiUrl()
	url := fmt.Sprintf("%s/projects/%s/merge_requests/%d/commits", apiUrl, projectID, mrID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Log(fmt.Sprintf("Error creating request for GitLab MR commits: %v", err))
		return "", err
	}

	req.Header.Set("PRIVATE-TOKEN", cfg.GitLabConfig.ApiToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log(fmt.Sprintf("Error connecting to GitLab API (%s): %v", apiUrl, err))
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("GitLab API responded with status code %d: %s", resp.StatusCode, string(body))
		logger.Log(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	var commits []struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		logger.Log(fmt.Sprintf("Error decoding GitLab commits response: %v", err))
		return "", err
	}

	if len(commits) > 0 {
		logger.Log(fmt.Sprintf("Current commit for MR #%d: %s", mrID, commits[0].ID))
		return commits[0].ID, nil
	}

	return "", fmt.Errorf("no commits found for merge request")
}
