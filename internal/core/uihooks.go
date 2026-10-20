//go:build windows

package core

import "goampp/internal/packages"

// UIHooks is what the application core needs from a front end. The GUI fills
// it in at start-up; with no front end attached every hook is a no-op, so the
// core never imports GUI code.
type UIHooks struct {
	// Do runs fn on the UI goroutine.
	Do func(fn func())
	// Status shows a short message to the user.
	Status func(text string)
	// RefreshServices re-renders service state. Called via Do.
	RefreshServices func()
	// Progress reports download/install progress from any goroutine.
	Progress packages.ProgressFunc
	// UpdateCheckDone and UpdateApplyDone end the busy state of the update
	// buttons. Called via Do.
	UpdateCheckDone func()
	UpdateApplyDone func()
}

var UI UIHooks

func UiDo(fn func()) {
	if UI.Do != nil {
		UI.Do(fn)
	}
}

func SetStatus(text string) {
	if UI.Status != nil {
		UI.Status(text)
	}
}

func UiRefreshServices() {
	if UI.RefreshServices != nil {
		UiDo(UI.RefreshServices)
	}
}

func uiDownloadProgress(stage, name string, done, total int64) {
	if UI.Progress != nil {
		UI.Progress(stage, name, done, total)
	}
}

func uiUpdateCheckDone() {
	if UI.UpdateCheckDone != nil {
		UiDo(UI.UpdateCheckDone)
	}
}

func uiUpdateApplyDone() {
	if UI.UpdateApplyDone != nil {
		UiDo(UI.UpdateApplyDone)
	}
}
