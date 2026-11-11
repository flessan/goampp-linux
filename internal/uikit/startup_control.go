package uikit

import "fyne.io/fyne/v2/widget"

// StartupCheck owns the state and rollback behavior for a startup checkbox.
// change must invoke done on the UI thread, synchronously or asynchronously;
// the OS adapter decides how to perform the operation.
func StartupCheck(label string, enabled, known bool, change func(bool, func(error)), finished func(bool, error)) *widget.Check {
	check := widget.NewCheck(label, nil)
	check.SetChecked(enabled)
	if !known {
		check.Disable()
		return check
	}
	updating := false
	check.OnChanged = func(value bool) {
		if updating {
			return
		}
		check.Disable()
		change(value, func(err error) {
			if err != nil {
				updating = true
				check.SetChecked(!value)
				updating = false
			}
			check.Enable()
			finished(value, err)
		})
	}
	return check
}
