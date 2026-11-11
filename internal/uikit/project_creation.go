package uikit

import (
	"errors"
	"strings"
)

// ProjectRequest is the captured form state for one creation operation.
type ProjectRequest struct {
	Name, Framework, Domain string
}

// ProjectCreator centralizes submission, validation, busy state and background
// completion. Submit and callbacks run on the UI thread; Create runs in a
// goroutine, and Dispatch must schedule its callback on the UI thread.
// Prepare and Create are platform adapters: they retain OS-specific validation,
// scaffolding and filesystem behavior.
type ProjectCreator struct {
	Prepare  func(ProjectRequest) (ProjectRequest, error)
	Create   func(ProjectRequest) (string, error)
	Dispatch func(func())
	Invalid  func(error)
	Started  func(ProjectRequest)
	Finished func(ProjectRequest, string, error)
	Busy     bool
}

// Submit rejects duplicate operations and validates before starting work.
func (c *ProjectCreator) Submit(request ProjectRequest) {
	if c.Busy {
		return
	}
	if strings.TrimSpace(request.Framework) == "" {
		c.Invalid(errors.New("Pick a framework first."))
		return
	}
	request, err := c.Prepare(request)
	if err != nil {
		c.Invalid(err)
		return
	}
	if strings.TrimSpace(request.Name) == "" {
		c.Invalid(errors.New("A project name is required."))
		return
	}
	c.Busy = true
	c.Started(request)
	go func() {
		result, err := c.Create(request)
		c.Dispatch(func() {
			c.Busy = false
			c.Finished(request, result, err)
		})
	}()
}
