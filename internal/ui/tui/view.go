// Package tui provides TUI rendering logic (View method).
// It handles all rendering logic for the TUI, including tab switching and different view modes.
package tui

import (
	"encoding/json"
	"fmt"
	"time"
)

func (m appModel) View() string {
	out := headerStyle.Render("Spartan Scraper") + "\n\n"
	out += "Tabs: [1] Jobs [2] Profiles [3] Schedules [4] Templates [5] Crawl States | r=refresh, f=filter, q=quit\n\n"

	if m.err != nil {
		out += errorStyle.Render(m.err.Error()) + "\n\n"
	} else if m.success != "" {
		out += successStyle.Render(m.success) + "\n\n"
	}

	switch m.tab {
	case "jobs":
		if m.viewMode == "detail" {
			out += headerStyle.Render("Job Details") + "\n\n"
			if m.selectedJob.ID == "" {
				out += "No job selected.\n"
				return out
			}
			out += fmt.Sprintf("ID:         %s\n", m.selectedJob.ID)
			out += fmt.Sprintf("Kind:       %s\n", m.selectedJob.Kind)
			out += fmt.Sprintf("Status:     %s\n", m.selectedJob.Status)
			out += fmt.Sprintf("Created:    %s\n", m.selectedJob.CreatedAt.Format(time.RFC3339))
			out += fmt.Sprintf("Updated:    %s\n", m.selectedJob.UpdatedAt.Format(time.RFC3339))
			out += fmt.Sprintf("Result:     %s\n", m.selectedJob.ResultPath)

			paramsJSON, _ := json.Marshal(m.selectedJob.Params)
			paramsStr := string(paramsJSON)
			if len(paramsStr) > 200 {
				paramsStr = paramsStr[:200] + "..."
			}
			out += fmt.Sprintf("Params:     %s\n", paramsStr)

			if m.selectedJob.Error != "" {
				errStr := m.selectedJob.Error
				if len(errStr) > 100 {
					errStr = errStr[:100] + "..."
				}
				out += fmt.Sprintf("Error:      %s\n", errStr)
			}

			out += "\n"
			out += "[Esc] Back to list | [c] Cancel | [e] Export\n"
		} else {
			out += headerStyle.Render("Jobs")
			if m.statusFilter != nil {
				out += fmt.Sprintf(" (filter: %s)", *m.statusFilter)
			}
			if len(m.fullJobs) > 0 {
				start := m.pageOffset + 1
				end := min(m.pageOffset+m.pageLimit, len(m.fullJobs))
				pageNum := (m.pageOffset / m.pageLimit) + 1
				out += fmt.Sprintf(" (Page %d: showing %d-%d)", pageNum, start, end)
			}
			out += "\n\n"
			if len(m.jobs) == 0 {
				out += "No jobs found.\n"
			} else {
				for i, line := range m.jobs {
					if i == m.cursor {
						out += "> " + line + "\n"
					} else {
						out += "  " + line + "\n"
					}
				}
			}
			out += "\n"
			out += "[↑/↓] Navigate [Enter/d] Details [←/→] Page [c] Cancel [e] Export [g/G] First/Last [r]efresh [f]ilter [q]uit\n"
		}
	case "profiles":
		out += headerStyle.Render("Auth Profiles") + "\n\n"
		if len(m.profiles) == 0 {
			out += "No profiles found.\n"
			return out
		}
		for _, name := range m.profiles {
			out += name + "\n"
		}
	case "schedules":
		out += headerStyle.Render("Schedules") + "\n\n"
		if len(m.schedules) == 0 {
			out += "No schedules found.\n"
			return out
		}
		for _, line := range m.schedules {
			out += line + "\n"
		}
	case "templates":
		out += headerStyle.Render("Extraction Templates") + "\n\n"
		if len(m.templates) == 0 {
			out += "No templates found.\n"
			return out
		}
		for _, name := range m.templates {
			out += name + "\n"
		}
	case "crawl-states":
		out += headerStyle.Render("Crawl States") + "\n\n"
		if len(m.crawlStates) == 0 {
			out += "No crawl states found.\n"
			return out
		}
		for _, line := range m.crawlStates {
			out += line + "\n"
		}
	}
	return out
}
