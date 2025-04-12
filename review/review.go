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
	Accepted     bool
	Commented    bool
}

func monitorMergeRequests(cfg *config.Config) {
	if !cfg.GitLabConfig.Enabled {
		logger.Log("GitLab integration is disabled, not monitoring MRs")
		return
	}

	// Natychmiastowe sprawdzenie przy starcie, bez czekania na ticker
	logger.Log("Starting immediate GitLab merge requests check")
	if cfg.GitLabConfig.ApiToken != "" {
		checkGitLabMergeRequests(cfg)
	} else {
		logger.Log("GitLab API token not configured")
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
			checkGitLabMergeRequests(cfg)
		}
	}
}

func checkGitLabMergeRequests(cfg *config.Config) {
	mergeRequests, err := gitlab.GetMergeRequestsToReview(cfg)
	if err != nil {
		logger.Log(fmt.Sprintf("Error fetching merge requests: %v", err))
		return
	}
	for _, mr := range mergeRequests {
		projectID := fmt.Sprintf("%d", mr.ProjectID)

		// Od razu sprawdzamy, czy merge request został skomentowany
		hasMyComment, err := gitlab.HasMyComment(cfg, projectID, mr.IID)
		if err != nil {
			logger.Log(fmt.Sprintf("Error checking if MR #%d has my comment: %v", mr.IID, err))
		}

		hasReply := false

		if hasMyComment {
			hasReply, err = gitlab.HasReplyOnMyComment(cfg, projectID, mr.IID)
			if err != nil {
				logger.Log(fmt.Sprintf("Error checking for replies on MR #%d: %v", mr.IID, err))
			}
			if !hasReply {
				logger.Log(fmt.Sprintf("MR #%d already has my comment with no reply, skipping", mr.IID))
				continue // Przejdź do następnego MR, bez przetwarzania tego
			}
			logger.Log(fmt.Sprintf("MR #%d has a reply to my comment, will process", mr.IID))
		}

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

				if currentCommit == review.LastCommit && hasMyComment && !hasReply {
					logger.Log(fmt.Sprintf("MR #%d: same commit and already commented with no reply, skipping", mr.IID))
					reviewsMutex.Unlock()
					goto nextMR
				} else if currentCommit != review.LastCommit && !review.IsInProgress {
					logger.Log(fmt.Sprintf("New commit detected for MR #%d, generating review", mr.IID))
					reviews[i].IsInProgress = true
					reviewsMutex.Unlock()
					changes, err := gitlab.GetMergeRequestChanges(cfg, projectID, mr.IID)
					if err != nil {
						logger.Log(fmt.Sprintf("Error getting changes: %v", err))
						markReviewNotInProgress(review.ID)
						goto nextMR
					}
					reviewText, err := openai.CodeReview(cfg, changes, false)
					if err != nil {
						logger.Log(fmt.Sprintf("Error generating review: %v", err))
						markReviewNotInProgress(review.ID)
						goto nextMR
					}
					reviewsMutex.Lock()
					reviews[i].LastCommit = currentCommit
					reviews[i].ReviewText = reviewText
					reviews[i].ReviewedAt = time.Now()
					reviews[i].IsInProgress = false
					reviews[i].Commented = false
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
				goto nextMR
			}
			changes, err := gitlab.GetMergeRequestChanges(cfg, projectID, mr.IID)
			if err != nil {
				logger.Log(fmt.Sprintf("Error getting initial changes: %v", err))
				reviewsMutex.Unlock()
				goto nextMR
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
				Commented:    false,
			}
			reviews = append(reviews, newReview)
			reviewsMutex.Unlock()
			reviewText, err := openai.CodeReview(cfg, changes, false)
			if err != nil {
				logger.Log(fmt.Sprintf("Error generating review: %v", err))
				markReviewNotInProgress(newReview.ID)
				goto nextMR
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
		}
	nextMR:
		continue
	}
}

func monitorReviewRequests(cfg *config.Config) {
	if !cfg.GitHubConfig.Enabled {
		logger.Log("GitHub integration is disabled, not monitoring PRs")
		return
	}

	logger.Log("Starting immediate GitHub pull requests check")
	if cfg.GitHubConfig.ApiToken != "" {
		checkGitHubPullRequests(cfg)
	} else {
		logger.Log("GitHub API token not configured")
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
			checkGitHubPullRequests(cfg)
		}
	}
}

func checkGitHubPullRequests(cfg *config.Config) {
	pullRequests, err := github.GetPullRequestsToReview(cfg)
	if err != nil {
		logger.Log(fmt.Sprintf("Error fetching pull requests: %v", err))
		return
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
					changes, err := github.GetPullRequestChanges(cfg, pr.Repository, pr.Number)
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
			changes, err := github.GetPullRequestChanges(cfg, pr.Repository, pr.Number)
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
			reviewText, err := openai.CodeReview(cfg, changes, false)
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
			reviewsMutex.Unlock()
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

func AcceptReview(reviewID string) {
	reviewsMutex.Lock()
	defer reviewsMutex.Unlock()
	for i, r := range reviews {
		if r.ID == reviewID {
			reviews[i].Accepted = true
			logger.Log(fmt.Sprintf("Review accepted: %s", r.Title))
			if r.Source == "gitlab" {
				cfg := config.LoadConfig()
				if err := gitlab.AcceptMergeRequestReview(cfg, r.ProjectID, r.MergeReqID, r.ReviewText); err != nil {
					logger.Log(fmt.Sprintf("Error accepting review in GitLab: %v", err))
				} else {
					reviews[i].Commented = true
					// Zapisujemy w stanie, że MR został skomentowany
					state.MarkGitLabProjectCommented(r.ProjectID)
				}
			} else if r.Source == "github" {
				cfg := config.LoadConfig()
				reviewMessage := "Review accepted: chore: add comment to EC2 example configuration\n" + r.ReviewText
				err := github.AcceptPullRequest(cfg, r.Repository, r.PullReqID, reviewMessage)
				if err != nil {
					logger.Log(fmt.Sprintf("Error accepting review in GitHub: %v", err))
				} else {
					reviews[i].Commented = true
					// (Jeśli chcesz, możesz też dodać funkcję stanu dla GitHub)
				}
			}
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
