// Package tui provides Lipgloss style constants and helper functions for the TUI.
// It defines the styling used across all TUI rendering.
package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

var (
	// Base colors - using a modern, accessible palette
	colorPrimary   = lipgloss.Color("#7C3AED") // Violet 600
	colorSecondary = lipgloss.Color("#06B6D4") // Cyan 500
	colorSuccess   = lipgloss.Color("#10B981") // Emerald 500
	colorWarning   = lipgloss.Color("#F59E0B") // Amber 500
	colorError     = lipgloss.Color("#EF4444") // Red 500
	colorInfo      = lipgloss.Color("#3B82F6") // Blue 500
	colorMuted     = lipgloss.Color("#6B7280") // Gray 500
	colorDark      = lipgloss.Color("#1F2937") // Gray 800
	colorLight     = lipgloss.Color("#F9FAFB") // Gray 50

	// Status-specific colors
	colorQueued    = lipgloss.Color("#6B7280") // Gray
	colorRunning   = lipgloss.Color("#3B82F6") // Blue
	colorSucceeded = lipgloss.Color("#10B981") // Green
	colorFailed    = lipgloss.Color("#EF4444") // Red
	colorCanceled  = lipgloss.Color("#F59E0B") // Amber

	// Layout styles
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			Padding(0, 1).
			MarginBottom(1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorLight).
			Background(colorPrimary).
			Padding(0, 2).
			MarginBottom(1)

	tabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorLight).
			Background(colorPrimary).
			Padding(0, 2)

	tabInactiveStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Padding(0, 2)

	// Status badge styles
	statusQueuedStyle    = lipgloss.NewStyle().Foreground(colorQueued)
	statusRunningStyle   = lipgloss.NewStyle().Foreground(colorRunning)
	statusSucceededStyle = lipgloss.NewStyle().Foreground(colorSucceeded)
	statusFailedStyle    = lipgloss.NewStyle().Foreground(colorFailed)
	statusCanceledStyle  = lipgloss.NewStyle().Foreground(colorCanceled)

	// Message styles
	errorStyle   = lipgloss.NewStyle().Foreground(colorError)
	successStyle = lipgloss.NewStyle().Foreground(colorSuccess)
	infoStyle    = lipgloss.NewStyle().Foreground(colorInfo)
	mutedStyle   = lipgloss.NewStyle().Foreground(colorMuted)

	// List styles
	cursorStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	itemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true).
				PaddingLeft(2)

	// Panel/box styles
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			Padding(1, 2)

	panelActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Padding(1, 2)

	// Status bar style
	statusBarStyle = lipgloss.NewStyle().
			Foreground(colorLight).
			Background(colorDark).
			Padding(0, 1)

	statusBarKeyStyle = lipgloss.NewStyle().
				Foreground(colorSecondary).
				Bold(true)

	// Help modal style
	helpModalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Background(colorDark).
			Padding(2, 4)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(colorLight)

	// Detail view styles
	detailLabelStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Width(12)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(colorLight)

	jsonStyle = lipgloss.NewStyle().
			Foreground(colorSecondary)
)

// GetStatusStyle returns the appropriate style for a job status
func GetStatusStyle(status model.Status) lipgloss.Style {
	switch status {
	case model.StatusQueued:
		return statusQueuedStyle
	case model.StatusRunning:
		return statusRunningStyle
	case model.StatusSucceeded:
		return statusSucceededStyle
	case model.StatusFailed:
		return statusFailedStyle
	case model.StatusCanceled:
		return statusCanceledStyle
	default:
		return lipgloss.NewStyle()
	}
}

// RenderStatusBadge returns a styled status badge with optional icon
func RenderStatusBadge(status model.Status) string {
	icon := ""
	switch status {
	case model.StatusQueued:
		icon = "○"
	case model.StatusRunning:
		icon = "◐"
	case model.StatusSucceeded:
		icon = "✓"
	case model.StatusFailed:
		icon = "✗"
	case model.StatusCanceled:
		icon = "⊘"
	}
	return GetStatusStyle(status).Render(fmt.Sprintf("%s %s", icon, status))
}

func cycleStatusFilter(current *model.Status) *model.Status {
	statuses := model.ValidStatuses()
	if current == nil {
		return &statuses[0]
	}
	for i, s := range statuses {
		if s == *current {
			if i < len(statuses)-1 {
				return &statuses[i+1]
			}
			return nil
		}
	}
	return nil
}
