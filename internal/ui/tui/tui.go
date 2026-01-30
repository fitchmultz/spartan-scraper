// Package tui provides a terminal user interface for viewing and managing jobs, profiles, schedules, and templates.
// It handles interactive TUI rendering, user input, and view navigation.
// It does NOT handle backend operations (delegates to other packages).
package tui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m appModel) Init() tea.Cmd {
	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorSecondary)
	m.spinner = s

	// Initialize progress bar
	m.progress = progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)

	// Initialize help model
	m.help = help.New()

	return tea.Batch(
		fetchJobs(m.ctx, m.store, m.statusFilter, m.pageLimit, m.pageOffset),
		fetchProfiles(m.ctx, m.store),
		fetchSchedules(m.ctx, m.store),
		fetchTemplates(m.ctx, m.store),
		fetchCrawlStates(m.ctx, m.store),
		tick(),
		m.spinner.Tick,
	)
}
