package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/michalopenmakers/lazyreview/config"
	"github.com/michalopenmakers/lazyreview/logger"
)

type PullRequest struct {
	Number     int `json:"number"`
	Repository string
	Title      string `json:"title"`
	HTMLURL    string `json:"html_url"`
}

func getFullApiUrl(cfg *config.Config) string {
	return cfg.GitHubConfig.GetGitHubApiUrl()
}

func GetPullRequestsToReview(cfg *config.Config) ([]PullRequest, error) {
	logger.Log("Fetching GitHub pull requests assigned for review")

	apiUrl := getFullApiUrl(cfg)
	url := fmt.Sprintf("%s/issues?filter=assigned&state=open", apiUrl)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Log(fmt.Sprintf("Error creating request for GitHub PRs: %v", err))
		return nil, err
	}

	req.Header.Set("Authorization", "token "+cfg.GitHubConfig.ApiToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log(fmt.Sprintf("Error connecting to GitHub API (%s): %v", apiUrl, err))
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
		errMsg := fmt.Sprintf("GitHub API responded with status code %d: %s", resp.StatusCode, string(body))
		logger.Log(errMsg)
		return nil, fmt.Errorf(errMsg)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log(fmt.Sprintf("Error reading API response: %v", err))
		return nil, err
	}
	logger.Log("API response: " + string(bodyBytes))

	var issues []struct {
		Number      int    `json:"number"`
		Title       string `json:"title"`
		HTMLURL     string `json:"html_url"`
		PullRequest struct {
			URL string `json:"url"`
		} `json:"pull_request"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}

	if err := json.Unmarshal(bodyBytes, &issues); err != nil {
		logger.Log(fmt.Sprintf("Error decoding GitHub response: %v", err))
		return nil, err
	}

	var pullRequests []PullRequest
	for _, issue := range issues {
		if issue.PullRequest.URL != "" {
			pr := PullRequest{
				Number:     issue.Number,
				Title:      issue.Title,
				HTMLURL:    issue.HTMLURL,
				Repository: issue.Repository.FullName,
			}
			pullRequests = append(pullRequests, pr)
		}
	}

	logger.Log(fmt.Sprintf("Successfully fetched %d pull requests for review", len(pullRequests)))
	return pullRequests, nil
}

func GetPullRequestChanges(cfg *config.Config, repository string, prID int) (string, error) {
	logger.Log(fmt.Sprintf("Getting changes for PR #%d in repo %s", prID, repository))

	apiUrl := getFullApiUrl(cfg)
	url := fmt.Sprintf("%s/repos/%s/pulls/%d/files", apiUrl, repository, prID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Log(fmt.Sprintf("Error creating request for GitHub PR changes: %v", err))
		return "", err
	}

	req.Header.Set("Authorization", "token "+cfg.GitHubConfig.ApiToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log(fmt.Sprintf("Error connecting to GitHub API (%s): %v", apiUrl, err))
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("GitHub API responded with status code %d: %s", resp.StatusCode, string(body))
		logger.Log(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log(fmt.Sprintf("Error reading API response: %v", err))
		return "", err
	}
	logger.Log("API response: " + string(bodyBytes))

	var files []struct {
		Filename string `json:"filename"`
		Patch    string `json:"patch"`
	}

	if err := json.Unmarshal(bodyBytes, &files); err != nil {
		logger.Log(fmt.Sprintf("Error decoding GitHub PR files response: %v", err))
		return "", err
	}

	var combinedChanges string
	for _, file := range files {
		fileHeader := fmt.Sprintf("--- a/%s\n+++ b/%s\n", file.Filename, file.Filename)
		if file.Patch != "" {
			combinedChanges += fileHeader + file.Patch + "\n\n"
		}
	}

	logger.Log(fmt.Sprintf("Successfully fetched changes for PR #%d, total size: %d bytes", prID, len(combinedChanges)))
	return combinedChanges, nil
}

func GetRepositoryCode(cfg *config.Config, repository string) (string, error) {
	logger.Log(fmt.Sprintf("Getting entire code for repository %s", repository))

	apiUrl := getFullApiUrl(cfg)
	url := fmt.Sprintf("%s/repos/%s/zipball", apiUrl, repository)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Log(fmt.Sprintf("Error creating request for GitHub repo code: %v", err))
		return "", err
	}

	req.Header.Set("Authorization", "token "+cfg.GitHubConfig.ApiToken)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log(fmt.Sprintf("Error connecting to GitHub API (%s): %v", apiUrl, err))
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("GitHub API responded with status code %d: %s", resp.StatusCode, string(body))
		logger.Log(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	logger.Log(fmt.Sprintf("Successfully fetched repository code for %s", repository))
	return fmt.Sprintf("Repository code for %s is too large to include directly. This is a placeholder for the actual downloaded code.", repository), nil
}

func GetChangesBetweenCommits(cfg *config.Config, repo, oldCommit, newCommit string) (string, error) {
	logger.Log(fmt.Sprintf("Getting changes between commits %s and %s for repo %s", oldCommit, newCommit, repo))
	if oldCommit == "" {
		return GetRepositoryCode(cfg, repo)
	}

	apiUrl := getFullApiUrl(cfg)
	url := fmt.Sprintf("%s/repos/%s/compare/%s...%s", apiUrl, repo, oldCommit, newCommit)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Log(fmt.Sprintf("Error creating request for GitHub compare: %v", err))
		return "", err
	}
	req.Header.Set("Authorization", "token "+cfg.GitHubConfig.ApiToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log(fmt.Sprintf("Error connecting to GitHub API (%s): %v", apiUrl, err))
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("GitHub API responded with status code %d: %s", resp.StatusCode, string(body))
		logger.Log(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log(fmt.Sprintf("Error reading API response: %v", err))
		return "", err
	}
	logger.Log("API response: " + string(bodyBytes))

	var compareResult struct {
		Files []struct {
			Filename string `json:"filename"`
			Patch    string `json:"patch"`
		} `json:"files"`
	}

	if err := json.Unmarshal(bodyBytes, &compareResult); err != nil {
		logger.Log(fmt.Sprintf("Error decoding GitHub compare response: %v", err))
		return "", err
	}

	var combinedDiff string
	for _, file := range compareResult.Files {
		fileHeader := fmt.Sprintf("--- a/%s\n+++ b/%s\n", file.Filename, file.Filename)
		combinedDiff += fileHeader + file.Patch + "\n\n"
	}

	logger.Log(fmt.Sprintf("Successfully fetched changes between commits, total size: %d bytes", len(combinedDiff)))
	return combinedDiff, nil
}

func GetCurrentCommit(cfg *config.Config, repo string, prID int) (string, error) {
	logger.Log(fmt.Sprintf("Getting current commit for PR #%d in repo %s", prID, repo))

	apiUrl := getFullApiUrl(cfg)
	url := fmt.Sprintf("%s/repos/%s/pulls/%d/commits", apiUrl, repo, prID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Log(fmt.Sprintf("Error creating request for GitHub PR commits: %v", err))
		return "", err
	}

	req.Header.Set("Authorization", "token "+cfg.GitHubConfig.ApiToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log(fmt.Sprintf("Error connecting to GitHub API (%s): %v", apiUrl, err))
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("GitHub API responded with status code %d: %s", resp.StatusCode, string(body))
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
		SHA string `json:"sha"`
	}

	if err := json.Unmarshal(bodyBytes, &commits); err != nil {
		logger.Log(fmt.Sprintf("Error decoding GitHub commits response: %v", err))
		return "", err
	}

	if len(commits) > 0 {
		logger.Log(fmt.Sprintf("Current commit for PR #%d: %s", prID, commits[len(commits)-1].SHA))
		return commits[len(commits)-1].SHA, nil
	}

	return "", fmt.Errorf("no commits found for pull request")
}

func AcceptPullRequest(cfg *config.Config, repository string, prNumber int, reviewMessage string) error {
	apiUrl := cfg.GitHubConfig.GetGitHubApiUrl()
	url := fmt.Sprintf("%s/repos/%s/pulls/%d/reviews", apiUrl, repository, prNumber)

	payload := map[string]string{
		"body":  reviewMessage,
		"event": "APPROVE",
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		logger.Log(fmt.Sprintf("Error marshaling accept review payload for GitHub: %v", err))
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		logger.Log(fmt.Sprintf("Error creating request for accepting GitHub review: %v", err))
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token "+cfg.GitHubConfig.ApiToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log(fmt.Sprintf("Error sending accept review request to GitHub: %v", err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("GitHub API responded with status code %d on accept review: %s", resp.StatusCode, string(body))
		logger.Log(errMsg)
		return fmt.Errorf(errMsg)
	}
	logger.Log(fmt.Sprintf("Successfully accepted review for PR #%d in repository %s", prNumber, repository))
	return nil
}
