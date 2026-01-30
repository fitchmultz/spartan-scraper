// Package tui provides TUI event handling (Update method).
// It handles all message and key event processing for the TUI.
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle spinner tick
	if m.showHelp {
		// Update help model when visible
		helpModel, cmd := m.help.Update(msg)
		m.help = helpModel
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.width > 40 {
			m.progress.Width = min(60, m.width-40) // Responsive progress bar
		}
		return m, nil

	case tea.KeyMsg:
		// Global help toggle (works in any mode)
		if msg.String() == "?" {
			m.showHelp = !m.showHelp
			return m, nil
		}

		if m.showHelp {
			// Any key closes help modal except ?
			if msg.String() != "?" {
				m.showHelp = false
				return m, nil
			}
		}

		if m.viewMode == "detail" {
			switch msg.String() {
			case "escape", "q":
				m.viewMode = "list"
				m.selectedJob = model.Job{}
				return m, nil
			case "c":
				if m.manager != nil {
					return m, cancelJobCmd(m.ctx, m.manager, m.selectedJob.ID)
				}
			case "e":
				return m, tea.Cmd(func() tea.Msg {
					return errorMsg{err: fmt.Errorf("use 'spartan export --job-id %s --format <format>' to export", m.selectedJob.ID)}
				})
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			return m, tea.Batch(
				fetchJobs(m.ctx, m.store, m.statusFilter, m.pageLimit, m.pageOffset),
				fetchProfiles(m.ctx, m.store),
				fetchSchedules(m.ctx, m.store),
				fetchTemplates(m.ctx, m.store),
				fetchCrawlStates(m.ctx, m.store),
			)
		// Vim-style navigation (add alongside existing arrow keys)
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "j", "down":
			if m.cursor < len(m.jobs)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.fullJobs) > m.cursor {
				m.selectedJob = m.fullJobs[m.cursor]
				m.viewMode = "detail"
				return m, fetchJobDetail(m.ctx, m.store, m.selectedJob.ID)
			}
		// Vim-style pagination
		case "h", "left":
			if m.pageOffset > 0 {
				newOffset := m.pageOffset - m.pageLimit
				if newOffset < 0 {
					newOffset = 0
				}
				m.pageOffset = newOffset
				m.cursor = 0
				return m, fetchJobs(m.ctx, m.store, m.statusFilter, m.pageLimit, m.pageOffset)
			}
		case "l", "right":
			m.pageOffset += m.pageLimit
			m.cursor = 0
			return m, fetchJobs(m.ctx, m.store, m.statusFilter, m.pageLimit, m.pageOffset)
		case "g":
			if m.pageOffset > 0 {
				m.pageOffset = 0
				m.cursor = 0
				return m, fetchJobs(m.ctx, m.store, m.statusFilter, m.pageLimit, m.pageOffset)
			}
		case "G":
			if len(m.fullJobs) > 0 {
				newOffset := ((len(m.fullJobs) - 1) / m.pageLimit) * m.pageLimit
				if newOffset > m.pageOffset {
					m.pageOffset = newOffset
					m.cursor = 0
					return m, fetchJobs(m.ctx, m.store, m.statusFilter, m.pageLimit, m.pageOffset)
				}
			}
		case "pageup":
			newOffset := m.pageOffset - m.pageLimit
			if newOffset < 0 {
				newOffset = 0
			}
			if newOffset != m.pageOffset {
				m.pageOffset = newOffset
				m.cursor = 0
				return m, fetchJobs(m.ctx, m.store, m.statusFilter, m.pageLimit, m.pageOffset)
			}
		case "pagedown":
			newOffset := m.pageOffset + m.pageLimit
			if newOffset != m.pageOffset {
				m.pageOffset = newOffset
				m.cursor = 0
				return m, fetchJobs(m.ctx, m.store, m.statusFilter, m.pageLimit, m.pageOffset)
			}
		case "1":
			m.tab = "jobs"
			m.viewMode = "list"
			return m, nil
		case "2":
			m.tab = "profiles"
			m.viewMode = "list"
			return m, nil
		case "3":
			m.tab = "schedules"
			m.viewMode = "list"
			return m, nil
		case "4":
			m.tab = "templates"
			m.viewMode = "list"
			return m, nil
		case "5":
			m.tab = "crawl-states"
			m.viewMode = "list"
			return m, nil
		case "f":
			m.statusFilter = cycleStatusFilter(m.statusFilter)
			m.pageOffset = 0
			m.cursor = 0
			return m, fetchJobs(m.ctx, m.store, m.statusFilter, m.pageLimit, m.pageOffset)
		case "c":
			if len(m.fullJobs) > m.cursor && m.manager != nil {
				return m, cancelJobCmd(m.ctx, m.manager, m.fullJobs[m.cursor].ID)
			}
		case "e":
			if len(m.fullJobs) > m.cursor {
				return m, tea.Cmd(func() tea.Msg {
					return errorMsg{err: fmt.Errorf("use 'spartan export --job-id %s --format <format>' to export", m.fullJobs[m.cursor].ID)}
				})
			}
		case "d":
			if len(m.fullJobs) > m.cursor {
				m.selectedJob = m.fullJobs[m.cursor]
				m.viewMode = "detail"
				return m, fetchJobDetail(m.ctx, m.store, m.selectedJob.ID)
			}
		}
	case tickMsg:
		return m, tea.Batch(fetchJobs(m.ctx, m.store, m.statusFilter, m.pageLimit, m.pageOffset), tick())
	case jobsMsg:
		m.jobs = msg.jobs
		m.fullJobs = msg.fullJobs
		if msg.err != nil {
			m.err = msg.err
		}
		if len(m.jobs) == 0 && m.pageOffset > 0 {
			newOffset := m.pageOffset - m.pageLimit
			if newOffset < 0 {
				newOffset = 0
			}
			m.pageOffset = newOffset
			m.cursor = 0
			return m, fetchJobs(m.ctx, m.store, m.statusFilter, m.pageLimit, m.pageOffset)
		}
		if m.cursor >= len(m.jobs) && len(m.jobs) > 0 {
			m.cursor = len(m.jobs) - 1
		}
		return m, tick()
	case profilesMsg:
		m.profiles = msg.profiles
		if msg.err != nil {
			m.err = msg.err
		}
	case schedulesMsg:
		m.schedules = msg.schedules
		if msg.err != nil {
			m.err = msg.err
		}
	case templatesMsg:
		m.templates = msg.templates
		if msg.err != nil {
			m.err = msg.err
		}
	case crawlStatesMsg:
		m.crawlStates = msg.crawlStates
		if msg.err != nil {
			m.err = msg.err
		}
	case jobDetailMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.selectedJob = msg.job
		}
	case jobCancelMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.success = fmt.Sprintf("Canceled job %s", msg.jobID)
		}
		return m, fetchJobs(m.ctx, m.store, m.statusFilter, m.pageLimit, m.pageOffset)
	case errorMsg:
		m.err = msg.err
	case successMsg:
		m.success = msg.message
		return m, fetchJobs(m.ctx, m.store, m.statusFilter, m.pageLimit, m.pageOffset)
	}

	// Update spinner
	if m.tab == "jobs" && m.viewMode == "list" {
		newSpinner, cmd := m.spinner.Update(msg)
		m.spinner = newSpinner
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}
