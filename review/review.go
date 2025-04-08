package review

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/michalopenmakers/lazyreview/config"
	"github.com/michalopenmakers/lazyreview/github"
	"github.com/michalopenmakers/lazyreview/gitlab"
	"github.com/michalopenmakers/lazyreview/notifications"
	"github.com/michalopenmakers/lazyreview/openai"
	"github.com/michalopenmakers/lazyreview/state"
	"github.com/michalopenmakers/lazyreview/store"
)

type CodeReview struct {
	ID            string
	Source        string
	ProjectID     string
	MRID          int
	Repository    string
	PRID          int
	Title         string
	URL           string
	Status        string
	Review        string
	CreatedAt     time.Time
	IsFirstReview bool
}

var (
	monitoringActive bool
	monitoringMutex  sync.Mutex
	stopMonitoring   chan struct{}
)

func GetCodeReviews() []CodeReview {
	reviewsJSON := state.GetState()
	if reviewsJSON == "" {
		return []CodeReview{}
	}
	var reviews []CodeReview
	return reviews
}

func StopMonitoring() {
	monitoringMutex.Lock()
	defer monitoringMutex.Unlock()
	if monitoringActive {
		close(stopMonitoring)
		monitoringActive = false
	}
}

func StartMonitoring(cfg *config.Config) {
	monitoringMutex.Lock()
	if monitoringActive {
		close(stopMonitoring)
	}
	stopMonitoring = make(chan struct{})
	monitoringActive = true
	monitoringMutex.Unlock()
	go monitorMergeRequests(cfg, stopMonitoring)
	go monitorReviewRequests(cfg, stopMonitoring)
}

func monitorMergeRequests(cfg *config.Config, stop chan struct{}) {
	ticker := time.NewTicker(time.Duration(cfg.MergeRequestsPollingInterval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			if cfg.GitLabConfig.Enabled {
				mergeRequests, err := gitlab.GetOpenMergeRequests(cfg)
				if err != nil {
					notifications.SendNotification(fmt.Sprintf("Error fetching MR from GitLab: %v", err))
				} else {
					for _, mr := range mergeRequests {
						exists := false
						reviews := GetCodeReviews()
						for _, review := range reviews {
							if review.Source == "gitlab" && review.ProjectID == fmt.Sprintf("%d", mr.ProjectID) && review.MRID == mr.IID {
								exists = true
								break
							}
						}
						if !exists {
							go reviewGitLabMR(cfg, mr)
						}
					}
				}
			}
			if cfg.GitHubConfig.Enabled {
				pullRequests, err := github.GetOpenPullRequests(cfg)
				if err != nil {
					notifications.SendNotification(fmt.Sprintf("Error fetching PR from GitHub: %v", err))
				} else {
					for _, pr := range pullRequests {
						exists := false
						reviews := GetCodeReviews()
						for _, review := range reviews {
							if review.Source == "github" && review.Repository == pr.Repository && review.PRID == pr.Number {
								exists = true
								break
							}
						}
						if !exists {
							go reviewGitHubPR(cfg, pr)
						}
					}
				}
			}
		}
	}
}

func monitorReviewRequests(cfg *config.Config, stop chan struct{}) {
	ticker := time.NewTicker(time.Duration(cfg.ReviewRequestsPollingInterval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			if cfg.GitLabConfig.Enabled {
				reviewRequests, err := gitlab.GetAssignedMergeRequests(cfg)
				if err != nil {
					notifications.SendNotification(fmt.Sprintf("Error fetching assigned MR from GitLab: %v", err))
				} else {
					for _, mr := range reviewRequests {
						exists := false
						reviews := GetCodeReviews()
						for _, review := range reviews {
							if review.Source == "gitlab" && review.ProjectID == fmt.Sprintf("%d", mr.ProjectID) && review.MRID == mr.IID {
								exists = true
								break
							}
						}
						if !exists {
							go reviewGitLabMR(cfg, mr)
						}
					}
				}
			}
			if cfg.GitHubConfig.Enabled {
				reviewRequests, err := github.GetAssignedPullRequests(cfg)
				if err != nil {
					notifications.SendNotification(fmt.Sprintf("Error fetching assigned PR from GitHub: %v", err))
				} else {
					for _, pr := range reviewRequests {
						exists := false
						reviews := GetCodeReviews()
						for _, review := range reviews {
							if review.Source == "github" && review.Repository == pr.Repository && review.PRID == pr.Number {
								exists = true
								break
							}
						}
						if !exists {
							go reviewGitHubPR(cfg, pr)
						}
					}
				}
			}
		}
	}
}

func reviewGitLabMR(cfg *config.Config, mr gitlab.MergeRequest) {
	reviews := GetCodeReviews()
	isFirstReview := true
	for _, r := range reviews {
		if r.Source == "gitlab" && r.ProjectID == fmt.Sprintf("%d", mr.ProjectID) && r.MRID == mr.IID {
			isFirstReview = false
			break
		}
	}
	review := CodeReview{
		ID:            fmt.Sprintf("gitlab-%d-%d", mr.ProjectID, mr.IID),
		Source:        "gitlab",
		ProjectID:     fmt.Sprintf("%d", mr.ProjectID),
		MRID:          mr.IID,
		Title:         mr.Title,
		URL:           mr.WebURL,
		Status:        "pending",
		CreatedAt:     time.Now(),
		IsFirstReview: isFirstReview,
	}
	saveReview(review)
	var codeChanges string
	var err error
	if isFirstReview {
		notifications.SendNotification(fmt.Sprintf("First review for GitLab MR #%d - retrieving entire project code", review.MRID))
		codeChanges, err = gitlab.GetProjectCode(cfg, review.ProjectID)
	} else {
		codeChanges, err = gitlab.GetMergeRequestChanges(cfg, review.ProjectID, review.MRID)
	}
	if err != nil {
		review.Status = "error"
		review.Review = fmt.Sprintf("Error fetching code: %v", err)
		saveReview(review)
		notifications.SendNotification(fmt.Sprintf("Error during review of GitLab MR #%d: %v", review.MRID, err))
		return
	}
	h := sha1.New()
	h.Write([]byte(codeChanges))
	currentHash := hex.EncodeToString(h.Sum(nil))
	lastHash, err := store.GetLastHash(review.ID)
	if err == nil && lastHash == currentHash {
		review.Status = "completed"
		review.Review = "No new changes since last review."
		saveReview(review)
		notifications.SendNotification(fmt.Sprintf("No new changes for GitLab MR #%d", review.MRID))
		return
	}
	aiReview, err := openai.CodeReview(cfg, codeChanges)
	if err != nil {
		review.Status = "error"
		review.Review = fmt.Sprintf("AI error: %v", err)
		saveReview(review)
		notifications.SendNotification(fmt.Sprintf("AI error during review of GitLab MR #%d: %v", review.MRID, err))
		return
	}
	review.Status = "completed"
	review.Review = aiReview
	saveReview(review)
	_ = store.UpdateLastHash(review.ID, currentHash)
	if isFirstReview {
		notifications.SendNotification(fmt.Sprintf("Completed first full review of GitLab MR #%d: %s", review.MRID, review.Title))
	} else {
		notifications.SendNotification(fmt.Sprintf("Completed review of changes for GitLab MR #%d: %s", review.MRID, review.Title))
	}
}

func reviewGitHubPR(cfg *config.Config, pr github.PullRequest) {
	reviews := GetCodeReviews()
	isFirstReview := true
	for _, r := range reviews {
		if r.Source == "github" && r.Repository == pr.Repository && r.PRID == pr.Number {
			isFirstReview = false
			break
		}
	}
	review := CodeReview{
		ID:            fmt.Sprintf("github-%s-%d", pr.Repository, pr.Number),
		Source:        "github",
		Repository:    pr.Repository,
		PRID:          pr.Number,
		Title:         pr.Title,
		URL:           pr.HTMLURL,
		Status:        "pending",
		CreatedAt:     time.Now(),
		IsFirstReview: isFirstReview,
	}
	saveReview(review)
	var codeChanges string
	var err error
	if isFirstReview {
		notifications.SendNotification(fmt.Sprintf("First review for GitHub PR #%d - retrieving entire repository code", review.PRID))
		codeChanges, err = github.GetRepositoryCode(cfg, review.Repository)
	} else {
		codeChanges, err = github.GetPullRequestChanges(cfg, review.Repository, review.PRID)
	}
	if err != nil {
		review.Status = "error"
		review.Review = fmt.Sprintf("Error fetching code: %v", err)
		saveReview(review)
		notifications.SendNotification(fmt.Sprintf("Error during review of GitHub PR #%d: %v", review.PRID, err))
		return
	}
	h := sha1.New()
	h.Write([]byte(codeChanges))
	currentHash := hex.EncodeToString(h.Sum(nil))
	lastHash, err := store.GetLastHash(review.ID)
	if err == nil && lastHash == currentHash {
		review.Status = "completed"
		review.Review = "No new changes since last review."
		saveReview(review)
		notifications.SendNotification(fmt.Sprintf("No new changes for GitHub PR #%d", review.PRID))
		return
	}
	aiReview, err := openai.CodeReview(cfg, codeChanges)
	if err != nil {
		review.Status = "error"
		review.Review = fmt.Sprintf("AI error: %v", err)
		saveReview(review)
		notifications.SendNotification(fmt.Sprintf("AI error during review of GitHub PR #%d: %v", review.PRID, err))
		return
	}
	review.Status = "completed"
	review.Review = aiReview
	saveReview(review)
	_ = store.UpdateLastHash(review.ID, currentHash)
	if isFirstReview {
		notifications.SendNotification(fmt.Sprintf("Completed first full review of GitHub PR #%d: %s", review.PRID, review.Title))
	} else {
		notifications.SendNotification(fmt.Sprintf("Completed review of changes for GitHub PR #%d: %s", review.PRID, review.Title))
	}
}

func saveReview(review CodeReview) {
	state.SetState(fmt.Sprintf("Review saved: %s", review.ID))
}
