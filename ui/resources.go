package ui

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed app_icon.png
var appIconData []byte

// resourceAppIconPng returns the app icon as a fyne.Resource
func resourceAppIconPng() fyne.Resource {
	resource := fyne.NewStaticResource("app_icon.png", appIconData)
	return resource
}
