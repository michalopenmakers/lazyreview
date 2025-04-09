package darwin

import (
	"log/slog"
	"os/exec"
	"runtime"
)

// ActivateApplication próbuje aktywować aplikację na poziomie systemu macOS
// używając AppleScript
func ActivateApplication() {
	// Ta funkcja działa tylko na macOS
	if runtime.GOOS != "darwin" {
		return
	}

	// Próba użycia AppleScript do aktywacji aplikacji
	cmd := exec.Command("osascript", "-e", `
tell application "System Events"
    set frontApp to name of first application process whose frontmost is true
    if frontApp is not "LazyReview" then
        try
            tell application "LazyReview" to activate
        on error
            # Jeśli aplikacja nie ma nazwy "LazyReview", spróbuj znaleźć proces
            set appList to application processes where name contains "LazyReview"
            if appList is not {} then
                set appProc to item 1 of appList
                set frontmost of appProc to true
            end if
        end try
    end if
end tell
`)

	err := cmd.Run()
	if err != nil {
		slog.Error("Error activating application via AppleScript", "error", err)
	} else {
		slog.Info("Application activation via AppleScript attempted")
	}
}
