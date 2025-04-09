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
	mainApp = app.New()
	w := mainApp.NewWindow("LazyReview")
	w.Resize(fyne.NewSize(800, 600))
	mainWindow = w

	setupSystemTray()
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

	w.SetCloseIntercept(func() {
		hideWindow()
	})

	go func() {
		for {
			time.Sleep(5 * time.Second)
			updateReviewsList(reviewsList, reviewDetails)
		}
	}()
	updateReviewsList(reviewsList, reviewDetails)

	w.ShowAndRun()
}

func setupSystemTray() {
	if desk, ok := mainApp.(desktop.App); ok {
		showItem := fyne.NewMenuItem("Pokaż", showWindow)
		hideItem := fyne.NewMenuItem("Ukryj", hideWindow)
		settingsItem := fyne.NewMenuItem("Ustawienia", showSettingsDialog)
		quitItem := fyne.NewMenuItem("Zakończ", func() {
			mainApp.Quit()
		})

		menu := fyne.NewMenu("LazyReview", showItem, hideItem, fyne.NewMenuItemSeparator(), settingsItem, fyne.NewMenuItemSeparator(), quitItem)
		desk.SetSystemTrayMenu(menu)

		desk.SetSystemTrayIcon(theme.FyneLogo())
	}
}

func showWindow() {
	mainWindow.Show()
	mainWindow.RequestFocus()
}

func hideWindow() {
	mainWindow.Hide()
	notifications.SendNotification("LazyReview działa w tle. Kliknij ikonę w zasobniku, aby pokazać.")
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
