package review

import (
	"fmt"
	"github.com/michalopenmakers/lazyreview/config"
	"github.com/michalopenmakers/lazyreview/github"
	"github.com/michalopenmakers/lazyreview/gitlab"
	"github.com/michalopenmakers/lazyreview/logger"
	"github.com/michalopenmakers/lazyreview/openai"
	"github.com/michalopenmakers/lazyreview/state"
	"sync"
	"time"
	// ...existing imports...
)

var stopChan = make(chan struct{})
var reviewsMutex = &sync.Mutex{}
var reviews []CodeReview

type CodeReview struct {
	ID           string
	Title        string
	URL          string
	LastCommit   string
	ReviewedAt   time.Time
	Source       string
	ProjectID    string
	MergeReqID   int
	Repository   string
	PullReqID    int
	ReviewText   string
	IsInProgress bool
}

func monitorMergeRequests(cfg *config.Config) {
	if !cfg.GitLabConfig.Enabled {
		logger.Log("GitLab integration is disabled, not monitoring MRs")
		return
	}

	ticker := time.NewTicker(time.Duration(cfg.MergeRequestsPollingInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			logger.Log("Stopping merge request monitoring")
			return
		case <-ticker.C:
			logger.Log("Pulling new merge requests")
			if cfg.GitLabConfig.ApiToken == "" {
				logger.Log("GitLab API token not configured")
				continue
			}
			mergeRequests, err := gitlab.GetMergeRequestsToReview(cfg)
			if err != nil {
				logger.Log(fmt.Sprintf("Error fetching merge requests: %v", err))
				continue
			}
			for _, mr := range mergeRequests {
				projectID := fmt.Sprintf("%d", mr.ProjectID)
				exists := false
				reviewsMutex.Lock()
				for i, review := range reviews {
					if review.Source == "gitlab" && review.ProjectID == projectID && review.MergeReqID == mr.IID {
						exists = true
						currentCommit, err := gitlab.GetCurrentCommit(cfg, projectID, mr.IID)
						if err != nil {
							logger.Log(fmt.Sprintf("Error getting current commit: %v", err))
							break
						}
						if currentCommit != review.LastCommit && !review.IsInProgress {
							logger.Log(fmt.Sprintf("New commit detected for MR #%d, generating review", mr.IID))
							reviews[i].IsInProgress = true
							reviewsMutex.Unlock()
							changes, err := gitlab.GetChangesBetweenCommits(cfg, projectID, review.LastCommit, currentCommit)
							if err != nil {
								logger.Log(fmt.Sprintf("Error getting changes: %v", err))
								markReviewNotInProgress(review.ID)
								break
							}
							reviewText, err := openai.CodeReview(cfg, changes, false)
							if err != nil {
								logger.Log(fmt.Sprintf("Error generating review: %v", err))
								markReviewNotInProgress(review.ID)
								break
							}
							reviewsMutex.Lock()
							reviews[i].LastCommit = currentCommit
							reviews[i].ReviewText = reviewText
							reviews[i].ReviewedAt = time.Now()
							reviews[i].IsInProgress = false
							reviewsMutex.Unlock()
							state.UpdateGitLabProjectState(projectID, currentCommit, time.Now().Unix())
							logger.Log(fmt.Sprintf("Updated review for MR #%d", mr.IID))
						} else {
							reviewsMutex.Unlock()
						}
						break
					}
				}
				if !exists {
					currentCommit, err := gitlab.GetCurrentCommit(cfg, projectID, mr.IID)
					if err != nil {
						logger.Log(fmt.Sprintf("Error getting current commit: %v", err))
						reviewsMutex.Unlock()
						continue
					}
					var changes string
					if state.IsFirstGitLabReview(projectID) {
						changes, err = gitlab.GetProjectCode(projectID)
					} else {
						changes, err = gitlab.GetMergeRequestChanges(cfg, projectID, mr.IID)
					}
					if err != nil {
						logger.Log(fmt.Sprintf("Error getting initial changes: %v", err))
						reviewsMutex.Unlock()
						continue
					}
					newReview := CodeReview{
						ID:           fmt.Sprintf("gitlab-%s-%d", projectID, mr.IID),
						Title:        mr.Title,
						URL:          mr.WebURL,
						LastCommit:   currentCommit,
						ReviewedAt:   time.Now(),
						Source:       "gitlab",
						ProjectID:    projectID,
						MergeReqID:   mr.IID,
						IsInProgress: true,
					}
					reviews = append(reviews, newReview)
					reviewsMutex.Unlock()
					reviewText, err := openai.CodeReview(cfg, changes, state.IsFirstGitLabReview(projectID))
					if err != nil {
						logger.Log(fmt.Sprintf("Error generating review: %v", err))
						markReviewNotInProgress(newReview.ID)
						continue
					}
					reviewsMutex.Lock()
					for i := range reviews {
						if reviews[i].ID == newReview.ID {
							reviews[i].ReviewText = reviewText
							reviews[i].IsInProgress = false
						}
					}
					reviewsMutex.Unlock()
					state.UpdateGitLabProjectState(projectID, currentCommit, time.Now().Unix())
					logger.Log(fmt.Sprintf("Added new review for MR #%d", mr.IID))
				} else {
					if !exists {
						reviewsMutex.Unlock()
					}
				}
			}
		}
	}
}

func monitorReviewRequests(cfg *config.Config) {
	if !cfg.GitHubConfig.Enabled {
		logger.Log("GitHub integration is disabled, not monitoring PRs")
		return
	}

	ticker := time.NewTicker(time.Duration(cfg.ReviewRequestsPollingInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			logger.Log("Stopping pull request monitoring")
			return
		case <-ticker.C:
			logger.Log("Pulling new pull requests")
			if cfg.GitHubConfig.ApiToken == "" {
				logger.Log("GitHub API token not configured")
				continue
			}
			pullRequests, err := github.GetPullRequestsToReview(cfg)
			if err != nil {
				logger.Log(fmt.Sprintf("Error fetching pull requests: %v", err))
				continue
			}
			for _, pr := range pullRequests {
				exists := false
				reviewsMutex.Lock()
				for i, review := range reviews {
					if review.Source == "github" && review.Repository == pr.Repository && review.PullReqID == pr.Number {
						exists = true
						currentCommit, err := github.GetCurrentCommit(cfg, pr.Repository, pr.Number)
						if err != nil {
							logger.Log(fmt.Sprintf("Error getting current commit: %v", err))
							break
						}
						if currentCommit != review.LastCommit && !review.IsInProgress {
							logger.Log(fmt.Sprintf("New commit detected for PR #%d in %s, generating review", pr.Number, pr.Repository))
							reviews[i].IsInProgress = true
							reviewsMutex.Unlock()
							changes, err := github.GetChangesBetweenCommits(cfg, pr.Repository, review.LastCommit, currentCommit)
							if err != nil {
								logger.Log(fmt.Sprintf("Error getting changes: %v", err))
								markReviewNotInProgress(review.ID)
								break
							}
							reviewText, err := openai.CodeReview(cfg, changes, false)
							if err != nil {
								logger.Log(fmt.Sprintf("Error generating review: %v", err))
								markReviewNotInProgress(review.ID)
								break
							}
							reviewsMutex.Lock()
							reviews[i].LastCommit = currentCommit
							reviews[i].ReviewText = reviewText
							reviews[i].ReviewedAt = time.Now()
							reviews[i].IsInProgress = false
							reviewsMutex.Unlock()
							state.UpdateGitHubRepoState(pr.Repository, currentCommit, time.Now().Unix())
							logger.Log(fmt.Sprintf("Updated review for PR #%d in %s", pr.Number, pr.Repository))
						} else {
							reviewsMutex.Unlock()
						}
						break
					}
				}
				if !exists {
					currentCommit, err := github.GetCurrentCommit(cfg, pr.Repository, pr.Number)
					if err != nil {
						logger.Log(fmt.Sprintf("Error getting current commit: %v", err))
						reviewsMutex.Unlock()
						continue
					}
					var changes string
					if state.IsFirstGitHubReview(pr.Repository) {
						changes, err = github.GetRepositoryCode(cfg, pr.Repository)
					} else {
						changes, err = github.GetPullRequestChanges(cfg, pr.Repository, pr.Number)
					}
					if err != nil {
						logger.Log(fmt.Sprintf("Error getting initial changes: %v", err))
						reviewsMutex.Unlock()
						continue
					}
					newReview := CodeReview{
						ID:           fmt.Sprintf("github-%s-%d", pr.Repository, pr.Number),
						Title:        pr.Title,
						URL:          pr.HTMLURL,
						LastCommit:   currentCommit,
						ReviewedAt:   time.Now(),
						Source:       "github",
						Repository:   pr.Repository,
						PullReqID:    pr.Number,
						IsInProgress: true,
					}
					reviews = append(reviews, newReview)
					reviewsMutex.Unlock()
					reviewText, err := openai.CodeReview(cfg, changes, state.IsFirstGitHubReview(pr.Repository))
					if err != nil {
						logger.Log(fmt.Sprintf("Error generating review: %v", err))
						markReviewNotInProgress(newReview.ID)
						continue
					}
					reviewsMutex.Lock()
					for i := range reviews {
						if reviews[i].ID == newReview.ID {
							reviews[i].ReviewText = reviewText
							reviews[i].IsInProgress = false
						}
					}
					reviewsMutex.Unlock()
					state.UpdateGitHubRepoState(pr.Repository, currentCommit, time.Now().Unix())
					logger.Log(fmt.Sprintf("Added new review for PR #%d in %s", pr.Number, pr.Repository))
				} else {
					if !exists {
						reviewsMutex.Unlock()
					}
				}
			}
		}
	}
}

func markReviewNotInProgress(reviewID string) {
	reviewsMutex.Lock()
	defer reviewsMutex.Unlock()
	for i, r := range reviews {
		if r.ID == reviewID {
			reviews[i].IsInProgress = false
			break
		}
	}
}

func StartMonitoring(cfg *config.Config) {
	stopChan = make(chan struct{})
	go monitorMergeRequests(cfg)
	go monitorReviewRequests(cfg)
}

func StopMonitoring() {
	close(stopChan)
}

func GetCodeReviews() []CodeReview {
	reviewsMutex.Lock()
	defer reviewsMutex.Unlock()
	return reviews
}
