package ui

import (
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/michalopenmakers/lazyreview/business"
	"github.com/michalopenmakers/lazyreview/config"
	"github.com/michalopenmakers/lazyreview/notifications"
)

var currentConfig *config.Config
var mainWindow fyne.Window
var mainApp fyne.App
var statusInfo *widget.Label

func updateReviewsList(reviewsList *fyne.Container, reviewDetails *widget.Entry) {
	// ...existing code to update review list...
	reviewsList.Refresh()
}

func setStatus(text string) {
	if statusInfo != nil {
		statusInfo.SetText(text)
	}
}

func StartUI() {
	mainApp = app.New()
	mainWindow = mainApp.NewWindow("LazyReview")
	mainWindow.Resize(fyne.NewSize(900, 600))

	setupSystemTray()
	currentConfig = config.LoadConfig()

	reviewDetails := widget.NewMultiLineEntry()
	reviewDetails.Disable()

	reviewsListContainer := container.NewVBox()
	scrollContainer := container.NewScroll(reviewsListContainer)
	split := container.NewHSplit(
		scrollContainer,
		buildDetailsSection(reviewDetails),
	)
	split.Offset = 0.3

	toolbar := buildToolbar(func() {
		updateReviewsList(reviewsListContainer, reviewDetails)
		setStatus("Review list refreshed.")
	}, showSettingsDialog)

	statusInfo = widget.NewLabel("")
	statusBar := container.New(layout.NewHBoxLayout(), statusInfo)
	setStatus("Application started.")

	content := container.NewBorder(
		toolbar,
		statusBar,
		nil,
		nil,
		split,
	)

	mainWindow.SetContent(content)

	mainWindow.SetCloseIntercept(func() {
		hideWindow()
	})

	go func() {
		for {
			time.Sleep(5 * time.Second)
			updateReviewsList(reviewsListContainer, reviewDetails)
		}
	}()
	updateReviewsList(reviewsListContainer, reviewDetails)

	mainWindow.ShowAndRun()
}

func buildToolbar(refreshAction func(), settingsAction func()) *widget.Toolbar {
	title := widget.NewLabel("LazyReview - AI Code Review")
	title.TextStyle = fyne.TextStyle{Bold: true}

	return widget.NewToolbar(
		widget.NewToolbarAction(theme.ViewRefreshIcon(), func() {
			refreshAction()
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.SettingsIcon(), func() {
			settingsAction()
		}),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(theme.InfoIcon(), func() {
			dialog.NewInformation("About", "LazyReview - AI Code Review\nVersion: 1.0\n© 2023 - 2025 MichalOpenmakers", mainWindow).Show()
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(nil, nil),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.FyneLogo(), func() {}),
	)
}

func buildDetailsSection(reviewDetails *widget.Entry) fyne.CanvasObject {
	detailsLabel := widget.NewLabel("Review Details:")
	detailsLabel.TextStyle = fyne.TextStyle{Bold: true}

	return container.NewVBox(
		detailsLabel,
		widget.NewSeparator(),
		reviewDetails,
	)
}

func setupSystemTray() {
	if desk, ok := mainApp.(desktop.App); ok {
		showItem := fyne.NewMenuItem("Show", showWindow)
		hideItem := fyne.NewMenuItem("Hide", hideWindow)
		settingsItem := fyne.NewMenuItem("Settings", showSettingsDialog)
		quitItem := fyne.NewMenuItem("Quit", func() {
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
	setStatus("Window restored.")
}

func hideWindow() {
	mainWindow.Hide()
	notifications.SendNotification("LazyReview is running in the background.")
	setStatus("Window hidden, application running in the background.")
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

	openaiTokenEntry := widget.NewPasswordEntry()
	openaiTokenEntry.SetText(currentConfig.AIModelConfig.ApiKey)
	openaiTokenEntry.PlaceHolder = "OpenAI API Token"

	openaiModelEntry := widget.NewEntry()
	openaiModelEntry.SetText(currentConfig.AIModelConfig.Model)
	openaiModelEntry.PlaceHolder = "OpenAI Model (np. o3-mini-high)"

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "GitLab", Widget: gitlabEnabledCheck},
			{Text: "GitLab API URL", Widget: gitlabUrlEntry},
			{Text: "GitLab Token", Widget: gitlabTokenEntry},
			{Text: "GitHub", Widget: githubEnabledCheck},
			{Text: "GitHub API URL", Widget: githubUrlEntry},
			{Text: "GitHub Token", Widget: githubTokenEntry},
			{Text: "OpenAI API Token", Widget: openaiTokenEntry},
			{Text: "OpenAI Model", Widget: openaiModelEntry},
			{Text: "MR/PR polling interval", Widget: mergeRequestsLayout},
			{Text: "Review polling interval", Widget: reviewRequestsLayout},
		},
	}

	saveButton := widget.NewButton("Save", func() {
		currentConfig.GitLabConfig.ApiUrl = gitlabUrlEntry.Text
		currentConfig.GitLabConfig.ApiToken = gitlabTokenEntry.Text
		currentConfig.GitHubConfig.ApiUrl = githubUrlEntry.Text
		currentConfig.GitHubConfig.ApiToken = githubTokenEntry.Text
		currentConfig.AIModelConfig.ApiKey = openaiTokenEntry.Text
		currentConfig.AIModelConfig.Model = openaiModelEntry.Text

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
		setStatus("Settings saved.")
	})

	content := container.NewVBox(
		form,
		saveButton,
	)
	dialog.ShowCustom("Settings", "Close", content, mainWindow)
}
