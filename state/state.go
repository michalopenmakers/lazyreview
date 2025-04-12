package state

import (
	"encoding/json"
	"fmt"
	"github.com/michalopenmakers/lazyreview/logger"
	"os"
	"path/filepath"
	"sync"
)

type ProjectState struct {
	LastReviewedCommit string
	LastReviewTime     int64
	ReviewCount        int
}

type AppState struct {
	GitLabProjects map[string]*ProjectState
	GitHubRepos    map[string]*ProjectState
}

var (
	appState      *AppState
	stateMutex    sync.RWMutex
	initialized   bool
	stateFilePath string
)

func initialize() {
	if initialized {
		return
	}
	stateMutex.Lock()
	defer stateMutex.Unlock()

	appState = &AppState{
		GitLabProjects: make(map[string]*ProjectState),
		GitHubRepos:    make(map[string]*ProjectState),
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	stateFilePath = filepath.Join(homeDir, ".lazyreview_state.json")

	if _, err := os.Stat(stateFilePath); err == nil {
		file, err := os.ReadFile(stateFilePath)
		if err == nil {
			err = json.Unmarshal(file, appState)
			if err != nil {
				fmt.Printf("Error unmarshaling state: %v, creating new state\n", err)
			}
		}
	}

	initialized = true
}

func SaveState() error {
	initialize()
	stateMutex.RLock()
	defer stateMutex.RUnlock()

	data, err := json.MarshalIndent(appState, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(stateFilePath, data, 0644)
}

func IsFirstGitLabReview(projectID string) bool {
	initialize()
	stateMutex.RLock()
	defer stateMutex.RUnlock()

	_, exists := appState.GitLabProjects[projectID]
	return !exists
}

func IsFirstGitHubReview(repoName string) bool {
	initialize()
	stateMutex.RLock()
	defer stateMutex.RUnlock()

	_, exists := appState.GitHubRepos[repoName]
	return !exists
}

func UpdateGitLabProjectState(projectID, commitID string, timestamp int64) {
	initialize()
	stateMutex.Lock()
	defer stateMutex.Unlock()

	if project, exists := appState.GitLabProjects[projectID]; exists {
		project.LastReviewedCommit = commitID
		project.LastReviewTime = timestamp
		project.ReviewCount++
	} else {
		appState.GitLabProjects[projectID] = &ProjectState{
			LastReviewedCommit: commitID,
			LastReviewTime:     timestamp,
			ReviewCount:        1,
		}
	}
	err := SaveState()
	if err != nil {
		logger.Log("Error saving state: " + err.Error())
		return
	}
}

func UpdateGitHubRepoState(repoName, commitID string, timestamp int64) {
	initialize()
	stateMutex.Lock()
	defer stateMutex.Unlock()

	if repo, exists := appState.GitHubRepos[repoName]; exists {
		repo.LastReviewedCommit = commitID
		repo.LastReviewTime = timestamp
		repo.ReviewCount++
	} else {
		appState.GitHubRepos[repoName] = &ProjectState{
			LastReviewedCommit: commitID,
			LastReviewTime:     timestamp,
			ReviewCount:        1,
		}
	}
	err := SaveState()
	if err != nil {
		return
	}
}
