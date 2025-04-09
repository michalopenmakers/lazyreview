package main

import (
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/michalopenmakers/lazyreview/business"
	"github.com/michalopenmakers/lazyreview/config"
	"github.com/michalopenmakers/lazyreview/ui"
)

func main() {
	// Jeśli uruchomiono z terminala na macOS, uruchamiamy aplikację przez "open"
	if runtime.GOOS == "darwin" && os.Getenv("TERM_PROGRAM") != "" {
		exePath, err := os.Executable()
		if err == nil {
			exec.Command("open", "-n", exePath).Start()
			return
		}
	}

	setupLogging()

	slog.Info("Starting LazyReview application", "os", runtime.GOOS)

	if runtime.GOOS == "darwin" {
		slog.Info("Running on macOS - using enhanced window activation")
	}

	cfg := config.LoadConfig()

	business.StartMonitoring(cfg)

	ui.StartUI()
}

func setupLogging() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		slog.SetDefault(slog.New(logHandler))
		return
	}

	logDir := filepath.Join(homeDir, "Library", "Logs", "LazyReview")
	if runtime.GOOS != "darwin" {
		logDir = filepath.Join(homeDir, ".lazyreview", "logs")
	}

	err = os.MkdirAll(logDir, 0755)
	if err != nil {
		logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		slog.SetDefault(slog.New(logHandler))
		return
	}

	logPath := filepath.Join(logDir, "lazyreview.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		slog.SetDefault(slog.New(logHandler))
		return
	}

	logHandler := slog.NewTextHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(logHandler))
}
