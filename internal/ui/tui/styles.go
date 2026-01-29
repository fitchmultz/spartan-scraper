// Package tui provides Lipgloss style constants and helper functions for the TUI.
// It defines the styling used across all TUI rendering.
package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

var (
	headerStyle  = lipgloss.NewStyle().Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
)

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
