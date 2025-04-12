package gitlab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/michalopenmakers/lazyreview/config"
	"io"
	"net/http"
	"time"

	"github.com/michalopenmakers/lazyreview/logger"
)

type MergeRequest struct {
	IID       int    `json:"iid"`
	ProjectID int    `json:"project_id"`
	Title     string `json:"title"`
	WebURL    string `json:"web_url"`
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
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Log(fmt.Sprintf("Error closing response body: %v", err))
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("GitLab API responded with status code %d: %s", resp.StatusCode, string(body))
		logger.Log(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log(fmt.Sprintf("Error reading API response: %v", err))
		return "", err
	}
	logger.Log("API response: " + string(bodyBytes))

	var response struct {
		Changes []struct {
			OldPath     string `json:"old_path"`
			NewPath     string `json:"new_path"`
			Diff        string `json:"diff"`
			RenamedFile bool   `json:"renamed_file"`
		} `json:"changes"`
	}

	if err := json.Unmarshal(bodyBytes, &response); err != nil {
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
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Log(fmt.Sprintf("Error closing response body: %v", err))
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("GitLab API responded with status code %d: %s", resp.StatusCode, string(body))
		logger.Log(errMsg)
		return nil, fmt.Errorf(errMsg)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log(fmt.Sprintf("Error reading API response: %v", err))
		return nil, err
	}
	logger.Log("API response: " + string(bodyBytes))

	var mergeRequests []MergeRequest
	if err := json.Unmarshal(bodyBytes, &mergeRequests); err != nil {
		logger.Log(fmt.Sprintf("Error decoding GitLab response: %v", err))
		return nil, err
	}

	logger.Log(fmt.Sprintf("Successfully fetched %d merge requests for review", len(mergeRequests)))
	return mergeRequests, nil
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
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Log(fmt.Sprintf("Error closing response body: %v", err))
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("GitLab API responded with status code %d: %s", resp.StatusCode, string(body))
		logger.Log(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log(fmt.Sprintf("Error reading API response: %v", err))
		return "", err
	}
	logger.Log("API response: " + string(bodyBytes))

	var commits []struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(bodyBytes, &commits); err != nil {
		logger.Log(fmt.Sprintf("Error decoding GitLab commits response: %v", err))
		return "", err
	}

	if len(commits) > 0 {
		logger.Log(fmt.Sprintf("Current commit for MR #%d: %s", mrID, commits[0].ID))
		return commits[0].ID, nil
	}

	return "", fmt.Errorf("no commits found for merge request")
}

func AcceptMergeRequestReview(cfg *config.Config, projectID string, mrID int, reviewText string) error {
	logger.Log(fmt.Sprintf("Accepting review for MR #%d in project %s", mrID, projectID))
	apiUrl := cfg.GitLabConfig.GetFullApiUrl()
	discussionUrl := fmt.Sprintf("%s/projects/%s/merge_requests/%d/discussions", apiUrl, projectID, mrID)
	payload := map[string]string{
		"body": reviewText,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		logger.Log(fmt.Sprintf("Error marshaling review payload: %v", err))
		return err
	}
	req, err := http.NewRequest("POST", discussionUrl, bytes.NewBuffer(jsonPayload))
	if err != nil {
		logger.Log(fmt.Sprintf("Error creating request for review: %v", err))
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PRIVATE-TOKEN", cfg.GitLabConfig.ApiToken)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log(fmt.Sprintf("Error sending review request: %v", err))
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Log(fmt.Sprintf("Error closing response body: %v", err))
		}
	}(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("GitLab API responded with status code %d on review: %s", resp.StatusCode, string(body))
		logger.Log(errMsg)
		return fmt.Errorf(errMsg)
	}
	logger.Log(fmt.Sprintf("Successfully accepted review for MR #%d", mrID))
	return nil
}
