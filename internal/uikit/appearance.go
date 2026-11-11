package uikit

import "fyne.io/fyne/v2"

// AppearanceControl owns the common selector state and preference update.
// apply retains the platform's theme and native-font implementation.
func AppearanceControl(app fyne.App, key, description string, apply func(string)) fyne.CanvasObject {
	options := []string{"System", "Light", "Dark"}
	selected := app.Preferences().StringWithFallback(key, options[0])
	control := NewSegmented(options, func(index int) {
		value := options[index]
		app.Preferences().SetString(key, value)
		apply(value)
	})
	for index, option := range options {
		if option == selected {
			control.SetSelected(index, false)
			break
		}
	}
	return SettingsRow("Theme", description, control.Object())
}
