package ui

import (
	_ "embed"
	"fmt"
	"image/color"
	"strconv"
	"strings"
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
	"github.com/michalopenmakers/lazyreview/logger"
	"github.com/michalopenmakers/lazyreview/notifications"
)

//go:embed icon.png
var iconPng []byte

var currentConfig *config.Config
var mainWindow fyne.Window
var mainApp fyne.App
var statusInfo *widget.Label
var currentReviewIndex = -1

func updateReviewsList(reviewsList *fyne.Container, reviewDetails *widget.Entry) {
	reviews := business.GetReviews()
	reviewsList.RemoveAll()

	if len(reviews) == 0 {
		emptyLabel := widget.NewLabel("No reviews available")
		emptyLabel.Alignment = fyne.TextAlignCenter
		reviewsList.Add(emptyLabel)

		if reviewDetails != nil {
			reviewDetails.SetText("")
		}
		currentReviewIndex = -1
	} else {
		for i, review := range reviews {
			reviewIndex := i
			reviewButton := widget.NewButton(review.Title, func() {
				if reviewDetails != nil {
					reviewDetails.SetText(review.ReviewText)
					currentReviewIndex = reviewIndex
					setStatus(fmt.Sprintf("Showing review: %s", review.Title))
				}
			})
			reviewsList.Add(reviewButton)
		}

		if currentReviewIndex >= 0 && currentReviewIndex < len(reviews) {
			if reviewDetails != nil {
				reviewDetails.SetText(reviews[currentReviewIndex].ReviewText)
			}
		} else if reviewDetails != nil && len(reviews) > 0 {
			currentReviewIndex = 0
			reviewDetails.SetText(reviews[0].ReviewText)
		}
	}

	reviewsList.Refresh()
}

func setStatus(text string) {
	if statusInfo != nil {
		statusInfo.SetText(text)
	}
}

var updateLogs func()

func StartUI() {
	mainApp = app.New()
	mainWindow = mainApp.NewWindow("LazyReview")
	mainWindow.Resize(fyne.NewSize(1920, 1080))
	mainWindow.CenterOnScreen()

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

	logsDisplay := widget.NewRichText()

	copyLogsButton := widget.NewButtonWithIcon("Copy logs", theme.ContentCopyIcon(), func() {
		logs := logger.GetLogs()
		formattedLogs := make([]string, len(logs))
		for i, log := range logs {
			formattedLogs[i] = " " + log
		}
		mainWindow.Clipboard().SetContent(strings.Join(formattedLogs, "\n"))
		setStatus("Logs copied to clipboard.")
	})

	updateLogs = func() {
		logs := logger.GetLogs()
		baseColor := theme.Color("text")
		lighterColor := lightenColor(baseColor, 0.5)
		hexColor := colorToHex(lighterColor)

		var markupLogs strings.Builder
		for _, log := range logs {
			markupLogs.WriteString(fmt.Sprintf("<span style='font-family:monospace; color:%s;'> %s</span>\n\n", hexColor, log))
		}
		newRich := widget.NewRichTextFromMarkdown(markupLogs.String())
		logsDisplay.Segments = newRich.Segments
		logsDisplay.Refresh()
	}

	logsScrollContainer := container.NewScroll(logsDisplay)
	logsScrollContainer.SetMinSize(fyne.NewSize(0, 150))

	logsContainer := container.NewVBox(
		container.NewBorder(nil, nil, nil, copyLogsButton, widget.NewLabel("Logs:")),
		logsScrollContainer,
	)

	bottomContainer := container.NewVBox(
		container.New(layout.NewHBoxLayout(), statusInfo),
		widget.NewSeparator(),
		logsContainer,
	)

	content := container.NewBorder(
		toolbar,
		bottomContainer,
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
			updateLogs()
		}
	}()
	updateReviewsList(reviewsListContainer, reviewDetails)
	updateLogs()

	mainWindow.ShowAndRun()
}

func buildToolbar(refreshAction func(), settingsAction func()) *widget.Toolbar {
	title := widget.NewLabel("LazyReview - AI Code Review")
	title.TextStyle = fyne.TextStyle{Bold: true}
	return widget.NewToolbar(
		widget.NewToolbarAction(theme.ViewRefreshIcon(), func() { refreshAction() }),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.SettingsIcon(), func() { settingsAction() }),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(theme.InfoIcon(), func() {
			dialog.NewInformation("About", "LazyReview - AI Code Review\nVersion: 1.0\n© 2023 - 2025 MichalOpenmakers", mainWindow).Show()
		}),
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
		quitItem := fyne.NewMenuItem("Quit", func() { mainApp.Quit() })
		menu := fyne.NewMenu("LazyReview", showItem, hideItem, fyne.NewMenuItemSeparator(), settingsItem, fyne.NewMenuItemSeparator(), quitItem)
		desk.SetSystemTrayMenu(menu)
		res := fyne.NewStaticResource("icon.png", iconPng)
		desk.SetSystemTrayIcon(res)
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
	gitlabUrlEntry.PlaceHolder = "e.g. gitlab.com or gitlab.hlag.altemista.cloud"

	gitlabTokenEntry := widget.NewPasswordEntry()
	gitlabTokenEntry.SetText(currentConfig.GitLabConfig.ApiToken)
	gitlabTokenEntry.PlaceHolder = "Personal Access Token"

	githubEnabledCheck := widget.NewCheck("Enable GitHub", func(enabled bool) {
		currentConfig.GitHubConfig.Enabled = enabled
	})
	githubEnabledCheck.Checked = currentConfig.GitHubConfig.Enabled

	githubTokenEntry := widget.NewPasswordEntry()
	githubTokenEntry.SetText(currentConfig.GitHubConfig.ApiToken)
	githubTokenEntry.PlaceHolder = "Personal Access Token"

	githubApiInfo := widget.NewLabel("GitHub API uses the standard URL: https://api.github.com")
	githubApiInfo.TextStyle = fyne.TextStyle{Italic: true}
	githubApiInfo.Alignment = fyne.TextAlignLeading

	githubContainer := container.NewVBox(
		githubTokenEntry,
		githubApiInfo,
	)

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
	openaiModelEntry.PlaceHolder = "OpenAI Model (e.g. o3-mini-high)"

	gitlabUrlInfo := widget.NewLabel("Enter only the GitLab domain; 'https://' and '/api/v4' will be added automatically")
	gitlabUrlInfo.TextStyle = fyne.TextStyle{Italic: true}
	gitlabUrlInfo.Alignment = fyne.TextAlignLeading

	gitlabUrlContainer := container.NewVBox(
		gitlabUrlEntry,
		gitlabUrlInfo,
	)

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "GitLab", Widget: gitlabEnabledCheck},
			{Text: "GitLab URL", Widget: gitlabUrlContainer},
			{Text: "GitLab Token", Widget: gitlabTokenEntry},
			{Text: "GitHub", Widget: githubEnabledCheck},
			{Text: "GitHub Token", Widget: githubContainer},
			{Text: "OpenAI API Token", Widget: openaiTokenEntry},
			{Text: "OpenAI Model", Widget: openaiModelEntry},
			{Text: "MR/PR polling interval", Widget: mergeRequestsLayout},
			{Text: "Review polling interval", Widget: reviewRequestsLayout},
		},
	}

	saveButton := widget.NewButton("Save", func() {
		currentConfig.GitLabConfig.ApiUrl = gitlabUrlEntry.Text
		currentConfig.GitLabConfig.ApiToken = gitlabTokenEntry.Text
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
		dialog.NewInformation("Settings saved", "Settings saved", mainWindow).Show()
		setStatus("Settings saved.")
	})

	content := container.NewVBox(
		form,
		saveButton,
	)
	dialog.ShowCustom("Settings", "Close", content, mainWindow)
}

func lightenColor(c color.Color, factor float64) color.Color {
	r, g, b, a := c.RGBA() // wartości 16-bitowe
	r8 := uint8(r >> 8)
	g8 := uint8(g >> 8)
	b8 := uint8(b >> 8)
	newR := uint8(float64(r8) + (255-float64(r8))*factor)
	newG := uint8(float64(g8) + (255-float64(g8))*factor)
	newB := uint8(float64(b8) + (255-float64(b8))*factor)
	return &color.NRGBA{R: newR, G: newG, B: newB, A: uint8(a >> 8)}
}

func colorToHex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
}
