package business

import (
	"github.com/michalopenmakers/lazyreview/config"
	"github.com/michalopenmakers/lazyreview/review"
)

func InitializeApplication(cfg *config.Config) {
	review.StartMonitoring(cfg)
}

func StartMonitoring(cfg *config.Config) {
	InitializeApplication(cfg)
}

func GetReviews() []review.CodeReview {
	return review.GetCodeReviews()
}

func RestartMonitoring(cfg *config.Config) {
	review.StopMonitoring()
	review.StartMonitoring(cfg)
}
