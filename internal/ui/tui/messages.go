// Package tui provides Bubble Tea message type definitions for the TUI.
// It defines all message types used for communication between TUI model and commands.
package tui

import (
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

type tickMsg time.Time

type jobsMsg struct {
	jobs     []string
	fullJobs []model.Job
	err      error
}

type profilesMsg struct {
	profiles []string
	err      error
}

type schedulesMsg struct {
	schedules []string
	err       error
}

type templatesMsg struct {
	templates []string
	err       error
}

type crawlStatesMsg struct {
	crawlStates []string
	err         error
}

type jobDetailMsg struct {
	job model.Job
	err error
}

type jobCancelMsg struct {
	jobID string
	err   error
}

type errorMsg struct {
	err error
}

type successMsg struct {
	message string
}
