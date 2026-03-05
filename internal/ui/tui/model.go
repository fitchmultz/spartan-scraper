// Package tui provides the TUI model struct and configuration options.
// It defines the appModel which holds the entire TUI state and Options for configuration.
package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

type appModel struct {
	ctx          context.Context
	store        *store.Store
	manager      *jobs.Manager
	tab          string
	statusFilter *model.Status
	jobs         []string
	fullJobs     []model.Job
	cursor       int
	pageOffset   int
	pageLimit    int
	viewMode     string
	selectedJob  model.Job
	profiles     []string
	schedules    []string
	templates    []string
	crawlStates  []string
	err          error
	success      string

	// New fields for modernization
	spinner  spinner.Model  // Loading animation
	progress progress.Model // Progress bar for running jobs
	help     help.Model     // Help component
	showHelp bool           // Help modal visibility
	width    int            // Terminal width for responsive layout
	height   int            // Terminal height
}

type Options struct {
	Smoke bool
}
