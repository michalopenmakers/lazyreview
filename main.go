package main

import (
	"github.com/michalopenmakers/lazyreview/business"
	"github.com/michalopenmakers/lazyreview/config"
	"github.com/michalopenmakers/lazyreview/notifications"
	"github.com/michalopenmakers/lazyreview/ui"
)

func main() {
	cfg := config.LoadConfig()
	business.InitializeApplication(cfg)
	notifications.SendNotification("LazyReview started! Merge requests monitoring initiated.")
	ui.StartUI()
}
