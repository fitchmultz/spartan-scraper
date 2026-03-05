// Package tui provides TUI rendering logic (View method).
// It handles all rendering logic for the TUI, including tab switching and different view modes.
package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Key bindings for help modal
type keyMap struct {
	Navigation []keyBinding
	Actions    []keyBinding
	View       []keyBinding
}

type keyBinding struct {
	key  string
	desc string
}

var defaultKeyMap = keyMap{
	Navigation: []keyBinding{
		{key: "↑/k", desc: "Move up"},
		{key: "↓/j", desc: "Move down"},
		{key: "←/h", desc: "Previous page"},
		{key: "→/l", desc: "Next page"},
		{key: "g", desc: "First page"},
		{key: "G", desc: "Last page"},
		{key: "1-5", desc: "Switch tabs"},
	},
	Actions: []keyBinding{
		{key: "enter/d", desc: "View details"},
		{key: "c", desc: "Cancel job"},
		{key: "e", desc: "Export hint"},
		{key: "f", desc: "Filter status"},
		{key: "r", desc: "Refresh"},
	},
	View: []keyBinding{
		{key: "?", desc: "Toggle help"},
		{key: "q", desc: "Quit"},
		{key: "esc", desc: "Back to list"},
	},
}

func (m appModel) View() string {
	if m.showHelp {
		return m.renderHelpModal()
	}

	var sections []string

	// Header with title
	sections = append(sections, titleStyle.Render("⚡ Spartan Scraper"))

	// Tab bar
	sections = append(sections, m.renderTabs())

	// Error/success messages
	if m.err != nil {
		sections = append(sections, errorStyle.Render("✗ "+m.err.Error()))
	} else if m.success != "" {
		sections = append(sections, successStyle.Render("✓ "+m.success))
	}

	// Main content based on tab
	switch m.tab {
	case "jobs":
		sections = append(sections, m.renderJobsTab())
	case "profiles":
		sections = append(sections, m.renderProfilesTab())
	case "schedules":
		sections = append(sections, m.renderSchedulesTab())
	case "templates":
		sections = append(sections, m.renderTemplatesTab())
	case "crawl-states":
		sections = append(sections, m.renderCrawlStatesTab())
	}

	// Status bar
	sections = append(sections, m.renderStatusBar())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m appModel) renderTabs() string {
	tabs := []struct {
		id   string
		num  string
		name string
	}{
		{"jobs", "1", "Jobs"},
		{"profiles", "2", "Profiles"},
		{"schedules", "3", "Schedules"},
		{"templates", "4", "Templates"},
		{"crawl-states", "5", "Crawl States"},
	}

	var rendered []string
	for _, t := range tabs {
		label := fmt.Sprintf("[%s] %s", t.num, t.name)
		if m.tab == t.id {
			rendered = append(rendered, tabActiveStyle.Render(label))
		} else {
			rendered = append(rendered, tabInactiveStyle.Render(label))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, rendered...)
}

func (m appModel) renderJobsTab() string {
	if m.viewMode == "detail" {
		return m.renderJobDetail()
	}
	return m.renderJobList()
}

func (m appModel) renderJobList() string {
	var content strings.Builder

	// Header with filter info
	header := headerStyle.Render("Jobs")
	if m.statusFilter != nil {
		header += fmt.Sprintf(" (filter: %s)", *m.statusFilter)
	}
	if len(m.fullJobs) > 0 {
		start := m.pageOffset + 1
		end := min(m.pageOffset+m.pageLimit, len(m.fullJobs))
		pageNum := (m.pageOffset / m.pageLimit) + 1
		header += fmt.Sprintf(" — Page %d (%d-%d of %d)", pageNum, start, end, len(m.fullJobs))
	}
	content.WriteString(header + "\n\n")

	if len(m.jobs) == 0 {
		content.WriteString("No jobs found.\n")
	} else {
		for i, job := range m.fullJobs {
			cursor := "  "
			if i == m.cursor {
				cursor = cursorStyle.Render("> ")
			}

			// Status badge with icon
			statusBadge := RenderStatusBadge(job.Status)

			// Format job line with columns
			idStr := truncateString(job.ID, 12)
			kindStr := string(job.Kind)
			timeStr := formatRelativeTime(job.UpdatedAt)

			line := fmt.Sprintf("%s%-12s %-10s %-12s %s",
				cursor, idStr, kindStr, statusBadge, timeStr)

			if i == m.cursor {
				line = selectedItemStyle.Render(line)
			} else {
				line = itemStyle.Render(line)
			}
			content.WriteString(line + "\n")
		}
	}

	return panelStyle.Render(content.String())
}

func (m appModel) renderJobDetail() string {
	if m.selectedJob.ID == "" {
		return panelStyle.Render("No job selected.")
	}

	var content strings.Builder
	content.WriteString(headerStyle.Render("Job Details") + "\n\n")

	job := m.selectedJob

	// Job info in a grid-like format
	content.WriteString(renderDetailRow("ID:", job.ID))
	content.WriteString(renderDetailRow("Kind:", string(job.Kind)))
	content.WriteString(renderDetailRow("Status:", RenderStatusBadge(job.Status)))
	content.WriteString(renderDetailRow("Created:", formatRelativeTime(job.CreatedAt)))
	content.WriteString(renderDetailRow("Updated:", formatRelativeTime(job.UpdatedAt)))

	if job.ResultPath != "" {
		content.WriteString(renderDetailRow("Result:", job.ResultPath))
	}

	// Progress bar for running jobs
	if job.Status == "running" {
		content.WriteString("\n")
		content.WriteString(infoStyle.Render("Progress:") + "\n")
		content.WriteString(m.progress.View() + "\n")
	}

	// Params as formatted JSON
	if len(job.Params) > 0 {
		content.WriteString("\n")
		content.WriteString(detailLabelStyle.Render("Params:") + "\n")
		paramsJSON, _ := json.MarshalIndent(job.Params, "", "  ")
		paramsStr := string(paramsJSON)
		if len(paramsStr) > 500 {
			paramsStr = paramsStr[:500] + "\n..."
		}
		content.WriteString(jsonStyle.Render(paramsStr) + "\n")
	}

	// Error display
	if job.Error != "" {
		content.WriteString("\n")
		content.WriteString(errorStyle.Render("Error: "+job.Error) + "\n")
	}

	// Action hints
	content.WriteString("\n")
	content.WriteString(statusBarKeyStyle.Render("[esc]") + " Back  ")
	content.WriteString(statusBarKeyStyle.Render("[c]") + " Cancel  ")
	content.WriteString(statusBarKeyStyle.Render("[e]") + " Export")

	return panelActiveStyle.Render(content.String())
}

func (m appModel) renderProfilesTab() string {
	var content strings.Builder
	content.WriteString(headerStyle.Render("Auth Profiles") + "\n\n")

	if len(m.profiles) == 0 {
		content.WriteString("No profiles found.\n")
	} else {
		for _, name := range m.profiles {
			content.WriteString(fmt.Sprintf("  • %s\n", name))
		}
	}

	return panelStyle.Render(content.String())
}

func (m appModel) renderSchedulesTab() string {
	var content strings.Builder
	content.WriteString(headerStyle.Render("Schedules") + "\n\n")

	if len(m.schedules) == 0 {
		content.WriteString("No schedules found.\n")
	} else {
		for _, line := range m.schedules {
			content.WriteString("  " + line + "\n")
		}
	}

	return panelStyle.Render(content.String())
}

func (m appModel) renderTemplatesTab() string {
	var content strings.Builder
	content.WriteString(headerStyle.Render("Extraction Templates") + "\n\n")

	if len(m.templates) == 0 {
		content.WriteString("No templates found.\n")
	} else {
		for _, name := range m.templates {
			content.WriteString(fmt.Sprintf("  • %s\n", name))
		}
	}

	return panelStyle.Render(content.String())
}

func (m appModel) renderCrawlStatesTab() string {
	var content strings.Builder
	content.WriteString(headerStyle.Render("Crawl States") + "\n\n")

	if len(m.crawlStates) == 0 {
		content.WriteString("No crawl states found.\n")
	} else {
		for _, line := range m.crawlStates {
			content.WriteString("  " + line + "\n")
		}
	}

	return panelStyle.Render(content.String())
}

func (m appModel) renderStatusBar() string {
	var parts []string

	// Current context
	parts = append(parts, statusBarKeyStyle.Render(m.tab))

	if m.viewMode == "detail" {
		parts = append(parts, "detail view")
	} else if m.statusFilter != nil {
		parts = append(parts, fmt.Sprintf("filter: %s", *m.statusFilter))
	}

	// Key hints
	parts = append(parts, "")
	parts = append(parts, statusBarKeyStyle.Render("?")+" help")
	parts = append(parts, statusBarKeyStyle.Render("q")+" quit")

	content := strings.Join(parts, " │ ")
	return statusBarStyle.Width(m.width).Render(content)
}

func (m appModel) renderHelpModal() string {
	var content strings.Builder
	content.WriteString(headerStyle.Render("Keyboard Shortcuts") + "\n\n")

	// Navigation section
	content.WriteString(infoStyle.Render("Navigation") + "\n")
	for _, kb := range defaultKeyMap.Navigation {
		content.WriteString(fmt.Sprintf("  %s %s\n",
			helpKeyStyle.Render(padRight(kb.key, 8)),
			helpDescStyle.Render(kb.desc)))
	}
	content.WriteString("\n")

	// Actions section
	content.WriteString(infoStyle.Render("Actions") + "\n")
	for _, kb := range defaultKeyMap.Actions {
		content.WriteString(fmt.Sprintf("  %s %s\n",
			helpKeyStyle.Render(padRight(kb.key, 8)),
			helpDescStyle.Render(kb.desc)))
	}
	content.WriteString("\n")

	// View section
	content.WriteString(infoStyle.Render("View") + "\n")
	for _, kb := range defaultKeyMap.View {
		content.WriteString(fmt.Sprintf("  %s %s\n",
			helpKeyStyle.Render(padRight(kb.key, 8)),
			helpDescStyle.Render(kb.desc)))
	}

	content.WriteString("\n" + mutedStyle.Render("Press ? or any key to close"))

	return helpModalStyle.Render(content.String())
}

// Helper functions
func renderDetailRow(label, value string) string {
	return fmt.Sprintf("%s %s\n", detailLabelStyle.Render(label), value)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatRelativeTime(t time.Time) string {
	duration := time.Since(t)
	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	case duration < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	default:
		return t.Format("Jan 02")
	}
}

func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(s))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
