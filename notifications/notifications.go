package notifications

import (
	"log/slog"
	"runtime"
	"time"

	"github.com/gen2brain/beeep"
)

var ShowWindowCallback func()

func SendNotification(message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	slog.Info("Notification", "timestamp", timestamp, "message", message)

	err := beeep.Notify("LazyReview", message, "")
	if err != nil {
		slog.Error("Failed to send system notification", "error", err)
	}

	if message != "Application minimized to system tray" {
		ShowWindow()
	}
}

func RegisterShowWindowCallback(callback func()) {
	ShowWindowCallback = callback
	slog.Info("Window callback registered")
}

func ShowWindow() {
	if ShowWindowCallback != nil {
		slog.Info("Showing window via callback", "platform", runtime.GOOS)
		// Wywołujemy callback w goroutine, aby uniknąć blokowania
		go ShowWindowCallback()

		// Dodatkowa próba dla macOS - czasem pierwsze wywołanie nie zadziała
		if runtime.GOOS == "darwin" {
			time.Sleep(100 * time.Millisecond)
			go ShowWindowCallback()
		}
	} else {
		slog.Warn("No callback registered for showing window")
	}
}
