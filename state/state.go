package state

import (
	"encoding/json"
	"fmt"
	"github.com/michalopenmakers/lazyreview/logger"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ProjectState struct {
	LastReviewedCommit string
	LastReviewTime     int64
	ReviewCount        int
	Commented          bool
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

	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Log(fmt.Sprintf("Error getting user home directory: %v", err))
		homeDir = "."
	}
	stateFilePath = filepath.Join(homeDir, ".lazyreview_state.json")
	logger.Log(fmt.Sprintf("Using state file: %s", stateFilePath))

	if _, err := os.Stat(stateFilePath); err == nil {
		// Plik istnieje, wczytaj stan
		if err := LoadState(); err != nil {
			logger.Log(fmt.Sprintf("Error loading state, creating new: %v", err))
			createNewState()
		}
	} else if os.IsNotExist(err) {
		// Plik nie istnieje, utwórz nowy stan
		logger.Log("State file doesn't exist, creating new")
		createNewState()
	} else {
		// Inny błąd
		logger.Log(fmt.Sprintf("Error checking state file: %v", err))
		createNewState()
	}

	initialized = true
}

func createNewState() {
	appState = &AppState{
		GitLabProjects: make(map[string]*ProjectState),
		GitHubRepos:    make(map[string]*ProjectState),
	}
	if err := SaveState(); err != nil {
		logger.Log(fmt.Sprintf("Error saving initial state: %v", err))
	}
}

func SaveState() error {
	if appState == nil {
		return fmt.Errorf("cannot save nil state")
	}

	tmpFile := stateFilePath + ".tmp"
	data, err := json.MarshalIndent(appState, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling state: %w", err)
	}

	err = os.WriteFile(tmpFile, data, 0600)
	if err != nil {
		return fmt.Errorf("error writing to temporary state file: %w", err)
	}

	err = os.Rename(tmpFile, stateFilePath)
	if err != nil {
		return fmt.Errorf("error renaming temporary state file: %w", err)
	}

	logger.Log("State saved successfully")
	return nil
}

func LoadState() error {
	data, err := os.ReadFile(stateFilePath)
	if err != nil {
		return fmt.Errorf("error reading state file: %w", err)
	}

	var state AppState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("error unmarshaling state: %w", err)
	}

	appState = &state
	logger.Log("State loaded successfully")
	return nil
}

func Init() {
	initialize()
	logger.Log("State module initialized")
}

func UpdateGitLabProjectState(projectID, commitID string, timestamp int64) {
	initialize()
	stateMutex.Lock()
	defer stateMutex.Unlock()

	if project, exists := appState.GitLabProjects[projectID]; exists {
		project.LastReviewedCommit = commitID
		project.LastReviewTime = timestamp
		project.ReviewCount++
		project.Commented = false // resetujemy flagę przy aktualizacji commit
	} else {
		appState.GitLabProjects[projectID] = &ProjectState{
			LastReviewedCommit: commitID,
			LastReviewTime:     timestamp,
			ReviewCount:        1,
			Commented:          false,
		}
	}
	err := SaveState()
	if err != nil {
		logger.Log("Error saving state: " + err.Error())
		return
	}
	logger.Log(fmt.Sprintf("GitLab project state updated for %s, commit %s", projectID, commitID))
}

func UpdateGitHubRepoState(repo, commitID string, timestamp int64) {
	initialize()
	stateMutex.Lock()
	defer stateMutex.Unlock()

	if project, exists := appState.GitHubRepos[repo]; exists {
		project.LastReviewedCommit = commitID
		project.LastReviewTime = timestamp
		project.ReviewCount++
		project.Commented = false
	} else {
		appState.GitHubRepos[repo] = &ProjectState{
			LastReviewedCommit: commitID,
			LastReviewTime:     timestamp,
			ReviewCount:        1,
			Commented:          false,
		}
	}
	err := SaveState()
	if err != nil {
		logger.Log("Error saving state: " + err.Error())
		return
	}
	logger.Log(fmt.Sprintf("GitHub repo state updated for %s, commit %s", repo, commitID))
}

func MarkGitLabProjectCommented(projectID string) {
	initialize()
	stateMutex.Lock()
	defer stateMutex.Unlock()

	if appState.GitLabProjects == nil {
		logger.Log("Warning: GitLabProjects map is nil, initializing")
		appState.GitLabProjects = make(map[string]*ProjectState)
	}

	if project, exists := appState.GitLabProjects[projectID]; exists {
		project.Commented = true
		logger.Log(fmt.Sprintf("Setting commented flag for project %s", projectID))
	} else {
		logger.Log(fmt.Sprintf("Creating new state entry for project %s", projectID))
		appState.GitLabProjects[projectID] = &ProjectState{
			LastReviewedCommit: "unknown",
			LastReviewTime:     time.Now().Unix(),
			ReviewCount:        1,
			Commented:          true,
		}
	}

	logger.Log("Calling SaveState after marking project as commented")
	err := SaveState()
	if err != nil {
		logger.Log(fmt.Sprintf("ERROR marking GitLab project as commented: %v", err))
	} else {
		logger.Log(fmt.Sprintf("CONFIRMATION: GitLab project %s marked as commented", projectID))
	}
}
