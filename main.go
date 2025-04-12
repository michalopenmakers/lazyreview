package main

import (
	"github.com/michalopenmakers/lazyreview/business"
	"github.com/michalopenmakers/lazyreview/config"
	"github.com/michalopenmakers/lazyreview/logger"
	"github.com/michalopenmakers/lazyreview/state"
	"github.com/michalopenmakers/lazyreview/ui"
)

func main() {
	logger.Log("Uruchamianie aplikacji LazyReview")
	state.Init()
	cfg := config.LoadConfig()
	business.InitializeApplication(cfg)
	ui.StartUI()
}
