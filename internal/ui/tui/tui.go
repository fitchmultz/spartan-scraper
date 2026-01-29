// Package tui provides a terminal user interface for viewing and managing jobs, profiles, schedules, and templates.
// It handles interactive TUI rendering, user input, and view navigation.
// It does NOT handle backend operations (delegates to other packages).
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m appModel) Init() tea.Cmd {
	return tea.Batch(
		fetchJobs(m.ctx, m.store, m.statusFilter, m.pageLimit, m.pageOffset),
		fetchProfiles(m.ctx, m.store),
		fetchSchedules(m.ctx, m.store),
		fetchTemplates(m.ctx, m.store),
		fetchCrawlStates(m.ctx, m.store),
		tick(),
	)
}
