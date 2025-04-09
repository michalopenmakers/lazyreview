package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"runtime"
	"strconv"
	"time"

	"github.com/michalopenmakers/lazyreview/business"
	"github.com/michalopenmakers/lazyreview/config"
	"github.com/michalopenmakers/lazyreview/notifications"
	"github.com/michalopenmakers/lazyreview/review"
)

type ReviewListItem struct {
	review review.CodeReview
	item   *fyne.Container
}

var currentConfig *config.Config
var mainWindow fyne.Window
var mainApp fyne.App

func StartUI() {
	a := app.NewWithID("com.michalopenmakers.lazyreview")
	mainApp = a

	// Ustawienie właściwości aplikacji dla macOS
	if runtime.GOOS == "darwin" {
		a.SetIcon(resourceAppIconPng())
	}

	w := a.NewWindow("LazyReview")
	w.Resize(fyne.NewSize(800, 600))
	mainWindow = w

	// Rejestrujemy funkcję przywracania okna, która będzie wywoływana po kliknięciu w powiadomienie
	notifications.RegisterShowWindowCallback(func() {
		if mainWindow != nil {
			if runtime.GOOS == "darwin" {
				mainWindow.Show()
				mainWindow.RequestFocus()

				// Usunięto wysyłanie dodatkowego powiadomienia, aby przerwać pętlę aktywacji
				go func() {
					time.Sleep(200 * time.Millisecond)
					// Druga próba po krótkim opóźnieniu
					time.Sleep(300 * time.Millisecond)
					mainWindow.Show()
					mainWindow.RequestFocus()
					if desk, ok := mainApp.(desktop.App); ok {
						desk.SetSystemTrayIcon(resourceAppIconPng())
					}
				}()
			} else {
				mainWindow.Show()
				mainWindow.RequestFocus()
			}
		}
	})

	// Okno nie zamyka się całkowicie przy kliknięciu "X", tylko się ukrywa
	w.SetCloseIntercept(func() {
		notifications.SendNotification("Application minimized to system tray")
		w.Hide()
	})

	// Tworzymy menu systemowe
	if desk, ok := a.(desktop.App); ok {
		systrayMenu := fyne.NewMenu("LazyReview",
			fyne.NewMenuItem("Show", func() {
				mainWindow.Show()
				mainWindow.RequestFocus()
			}),
			fyne.NewMenuItem("Exit", func() {
				a.Quit()
			}),
		)
		desk.SetSystemTrayMenu(systrayMenu)

		// Ustawiamy ikonę w tacce systemowej
		if runtime.GOOS == "darwin" {
			desk.SetSystemTrayIcon(resourceAppIconPng())
		} else {
			desk.SetSystemTrayIcon(theme.InfoIcon())
		}
	}

	currentConfig = config.LoadConfig()

	title := widget.NewLabel("LazyReview - AI Code Review")
	title.TextStyle = fyne.TextStyle{Bold: true}

	settingsButton := widget.NewButtonWithIcon("Settings", theme.SettingsIcon(), showSettingsDialog)
	header := container.NewBorder(nil, nil, nil, settingsButton, title)

	reviewsList := container.NewVBox()
	scrollContainer := container.NewScroll(reviewsList)
	reviewDetails := widget.NewMultiLineEntry()
	reviewDetails.Disable()

	split := container.NewHSplit(
		scrollContainer,
		container.NewVBox(
			widget.NewLabel("Review details:"),
			reviewDetails,
		),
	)
	split.Offset = 0.3

	content := container.NewVBox(
		header,
		widget.NewSeparator(),
		split,
	)

	w.SetContent(content)

	go func() {
		for {
			time.Sleep(5 * time.Second)
			updateReviewsList(reviewsList, reviewDetails)
		}
	}()
	updateReviewsList(reviewsList, reviewDetails)

	w.ShowAndRun()
}

// ShowMainWindow przywraca główne okno aplikacji
func ShowMainWindow() {
	if mainWindow != nil {
		if runtime.GOOS == "darwin" {
			// Bardziej rozbudowane podejście do przywracania okna na macOS
			// Pierwsza próba
			mainWindow.Show()
			mainWindow.RequestFocus()

			// Dodatkowe działania w osobnej goroutinie
			go func() {
				// Dodatkowe opóźnienie
				time.Sleep(200 * time.Millisecond)

				// Druga próba
				mainWindow.Show()
				mainWindow.RequestFocus()

				if mainApp != nil {
					mainApp.SendNotification(&fyne.Notification{
						Title:   "LazyReview",
						Content: "Application activated",
					})

					if desk, ok := mainApp.(desktop.App); ok {
						desk.SetSystemTrayIcon(theme.InfoIcon())
					}
				}
			}()
		} else {
			mainWindow.Show()
			mainWindow.RequestFocus()
		}
	}
}

func showSettingsDialog() {
	gitlabEnabledCheck := widget.NewCheck("Enable GitLab", func(enabled bool) {
		currentConfig.GitLabConfig.Enabled = enabled
	})
	gitlabEnabledCheck.Checked = currentConfig.GitLabConfig.Enabled

	gitlabUrlEntry := widget.NewEntry()
	gitlabUrlEntry.SetText(currentConfig.GitLabConfig.ApiUrl)
	gitlabUrlEntry.PlaceHolder = "https://gitlab.com/api/v4"

	gitlabTokenEntry := widget.NewPasswordEntry()
	gitlabTokenEntry.SetText(currentConfig.GitLabConfig.ApiToken)
	gitlabTokenEntry.PlaceHolder = "Personal Access Token"

	githubEnabledCheck := widget.NewCheck("Enable GitHub", func(enabled bool) {
		currentConfig.GitHubConfig.Enabled = enabled
	})
	githubEnabledCheck.Checked = currentConfig.GitHubConfig.Enabled

	githubUrlEntry := widget.NewEntry()
	githubUrlEntry.SetText(currentConfig.GitHubConfig.ApiUrl)
	githubUrlEntry.PlaceHolder = "https://api.github.com"

	githubTokenEntry := widget.NewPasswordEntry()
	githubTokenEntry.SetText(currentConfig.GitHubConfig.ApiToken)
	githubTokenEntry.PlaceHolder = "Personal Access Token"

	mergeRequestsIntervalEntry := widget.NewEntry()
	mergeRequestsIntervalEntry.SetText(strconv.Itoa(currentConfig.MergeRequestsPollingInterval))
	mergeRequestsIntervalUnit := widget.NewLabel("seconds")
	mergeRequestsLayout := container.NewHBox(mergeRequestsIntervalEntry, mergeRequestsIntervalUnit)

	reviewRequestsIntervalEntry := widget.NewEntry()
	reviewRequestsIntervalEntry.SetText(strconv.Itoa(currentConfig.ReviewRequestsPollingInterval))
	reviewRequestsIntervalUnit := widget.NewLabel("seconds")
	reviewRequestsLayout := container.NewHBox(reviewRequestsIntervalEntry, reviewRequestsIntervalUnit)

	aiModelSelect := widget.NewSelect([]string{"o3-mini-high", "o3-mini"}, func(selected string) {
		currentConfig.AIModelConfig.Model = selected
	})
	aiModelSelect.Selected = currentConfig.AIModelConfig.Model

	aiApiKeyEntry := widget.NewPasswordEntry()
	aiApiKeyEntry.SetText(currentConfig.AIModelConfig.ApiKey)
	aiApiKeyEntry.PlaceHolder = "AI API Key"

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "GitLab", Widget: gitlabEnabledCheck},
			{Text: "GitLab API URL", Widget: gitlabUrlEntry},
			{Text: "GitLab Token", Widget: gitlabTokenEntry},
			{Text: "GitHub", Widget: githubEnabledCheck},
			{Text: "GitHub API URL", Widget: githubUrlEntry},
			{Text: "GitHub Token", Widget: githubTokenEntry},
			{Text: "MR/PR polling interval", Widget: mergeRequestsLayout},
			{Text: "Review polling interval", Widget: reviewRequestsLayout},
			{Text: "AI Model", Widget: aiModelSelect},
			{Text: "AI API Key", Widget: aiApiKeyEntry},
		},
		OnSubmit: func() {
			currentConfig.GitLabConfig.ApiUrl = gitlabUrlEntry.Text
			currentConfig.GitLabConfig.ApiToken = gitlabTokenEntry.Text
			currentConfig.GitHubConfig.ApiUrl = githubUrlEntry.Text
			currentConfig.GitHubConfig.ApiToken = githubTokenEntry.Text
			mrInterval, err := strconv.Atoi(mergeRequestsIntervalEntry.Text)
			if err == nil && mrInterval > 0 {
				currentConfig.MergeRequestsPollingInterval = mrInterval
			}
			rrInterval, err := strconv.Atoi(reviewRequestsIntervalEntry.Text)
			if err == nil && rrInterval > 0 {
				currentConfig.ReviewRequestsPollingInterval = rrInterval
			}
			// Aktualizacja konfiguracji AI
			currentConfig.AIModelConfig.Model = aiModelSelect.Selected
			currentConfig.AIModelConfig.ApiKey = aiApiKeyEntry.Text

			err = config.SaveConfig(currentConfig)
			if err != nil {
				dialog.ShowError(err, mainWindow)
				return
			}
			notifications.SendNotification("Configuration saved.")
			business.RestartMonitoring(currentConfig)
			dialog.NewInformation("Saved", "Settings saved.", mainWindow).Show()
		},
		OnCancel: func() {
		},
	}

	dialog.ShowCustom("Settings", "Save", form, mainWindow)
}

func updateReviewsList(reviewsList *fyne.Container, reviewDetails *widget.Entry) {
	reviews := business.GetReviews()
	reviewsList.Objects = nil
	for _, r := range reviews {
		codeReview := r
		var statusIcon *widget.Icon
		switch codeReview.Status {
		case "completed":
			statusIcon = widget.NewIcon(theme.ConfirmIcon())
		case "error":
			statusIcon = widget.NewIcon(theme.ErrorIcon())
		default:
			statusIcon = widget.NewIcon(theme.InfoIcon())
		}
		var titleLabel *widget.Label
		if codeReview.Source == "gitlab" {
			titleLabel = widget.NewLabel(fmt.Sprintf("GitLab MR #%d: %s", codeReview.MRID, codeReview.Title))
		} else {
			titleLabel = widget.NewLabel(fmt.Sprintf("GitHub PR #%d: %s", codeReview.PRID, codeReview.Title))
		}
		viewButton := widget.NewButton("Show", func() {
			reviewDetails.SetText(codeReview.Review)
		})
		item := container.NewHBox(
			statusIcon,
			titleLabel,
			layout.NewSpacer(),
			viewButton,
		)
		reviewsList.Add(item)
	}
	reviewsList.Refresh()
}
