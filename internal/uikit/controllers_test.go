package uikit

import (
	"errors"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestProjectCreatorValidationAndBusyLifecycle(t *testing.T) {
	queue := make(chan func(), 1)
	unblock := make(chan struct{})
	created := make(chan ProjectRequest, 1)
	var invalid error
	var started, finished int
	c := &ProjectCreator{
		Prepare: func(r ProjectRequest) (ProjectRequest, error) { r.Name = "normalized"; return r, nil },
		Create: func(r ProjectRequest) (string, error) {
			created <- r
			<-unblock
			return "partial", errors.New("failed")
		},
		Dispatch: func(fn func()) { queue <- fn },
		Invalid:  func(err error) { invalid = err },
		Started:  func(ProjectRequest) { started++ },
		Finished: func(r ProjectRequest, result string, err error) {
			finished++
			if r.Name != "normalized" || result != "partial" || err == nil {
				t.Error("completion lost adapter result")
			}
		},
	}
	c.Submit(ProjectRequest{Name: "name"})
	if invalid == nil || c.Busy || started != 0 {
		t.Fatal("missing framework should not start work")
	}
	c.Submit(ProjectRequest{Name: "name", Framework: "Go"})
	if !c.Busy || started != 1 {
		t.Fatal("expected busy operation")
	}
	c.Submit(ProjectRequest{Name: "second", Framework: "Go"})
	if started != 1 {
		t.Fatal("duplicate operation started")
	}
	select {
	case r := <-created:
		if r.Name != "normalized" {
			t.Fatal("prepared request not used")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("creation did not start")
	}
	close(unblock)
	select {
	case fn := <-queue:
		fn()
	case <-time.After(5 * time.Second):
		t.Fatal("completion was not dispatched")
	}
	if c.Busy || finished != 1 {
		t.Fatal("operation did not finish")
	}
}

func TestStartupCheckRollbackDoesNotRepeatOperation(t *testing.T) {
	test.NewTempApp(t)
	var done func(error)
	calls, completed := 0, 0
	check := StartupCheck("Service", false, true, func(value bool, callback func(error)) {
		calls++
		done = callback
	}, func(value bool, err error) { completed++ })
	check.SetChecked(true)
	if calls != 1 || !check.Disabled() {
		t.Fatal("change was not started and disabled")
	}
	done(errors.New("denied"))
	if check.Checked || check.Disabled() || calls != 1 || completed != 1 {
		t.Fatal("failed change was not rolled back safely")
	}
	unknown := StartupCheck("Unknown", false, false, nil, nil)
	if !unknown.Disabled() {
		t.Fatal("unknown state should be disabled")
	}
}

func TestAppearanceControlUpdatesPreference(t *testing.T) {
	app := test.NewTempApp(t)
	app.Preferences().SetString("theme", "Dark")
	var applied string
	root := AppearanceControl(app, "theme", "description", func(v string) { applied = v })
	var light *widget.Button
	var find func(fyne.CanvasObject)
	find = func(obj fyne.CanvasObject) {
		if b, ok := obj.(*widget.Button); ok && b.Text == "Light" {
			light = b
		}
		if c, ok := obj.(*fyne.Container); ok {
			for _, child := range c.Objects {
				find(child)
			}
		}
	}
	find(root)
	if light == nil {
		t.Fatal("Light control missing")
	}
	test.Tap(light)
	if app.Preferences().String("theme") != "Light" || applied != "Light" {
		t.Fatal("appearance change was not applied and persisted")
	}
}

func TestProjectCreatorPreparationFailure(t *testing.T) {
	want := errors.New("bad name")
	var got error
	c := &ProjectCreator{
		Prepare: func(r ProjectRequest) (ProjectRequest, error) { return r, want },
		Invalid: func(err error) { got = err },
	}
	c.Submit(ProjectRequest{Name: "bad", Framework: "Go"})
	if got != want || c.Busy {
		t.Fatal("invalid request started an operation")
	}
}

func TestStartupCheckSuccessfulChange(t *testing.T) {
	test.NewTempApp(t)
	calls := 0
	check := StartupCheck("Service", false, true, func(value bool, done func(error)) {
		calls++
		done(nil)
	}, func(value bool, err error) {
		if !value || err != nil {
			t.Fatal("wrong completion")
		}
	})
	check.SetChecked(true)
	if !check.Checked || check.Disabled() || calls != 1 {
		t.Fatal("successful change did not retain state")
	}
}
