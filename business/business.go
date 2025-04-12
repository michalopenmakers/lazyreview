package business

import (
	"fmt"
	"github.com/michalopenmakers/lazyreview/config"
	"github.com/michalopenmakers/lazyreview/review"
	"github.com/michalopenmakers/lazyreview/store"
)

func InitializeApplication(cfg *config.Config) {
	review.StartMonitoring(cfg)
}

func GetReviews() []review.CodeReview {
	reviews := review.GetCodeReviews()
	dataStore := store.GetStore()
	dataStore.Data["reviewCount"] = fmt.Sprintf("%d", len(reviews))
	return reviews
}

func RestartMonitoring(cfg *config.Config) {
	review.StopMonitoring()
	review.StartMonitoring(cfg)
}
