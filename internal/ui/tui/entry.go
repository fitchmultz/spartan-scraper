// Package tui provides entry points for running the TUI.
// It defines Run, RunWithOptions, and Smoke functions for launching the interface.
package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func Run(ctx context.Context, store *store.Store) int {
	return RunWithOptions(ctx, store, nil, Options{})
}

func RunWithOptions(ctx context.Context, store *store.Store, manager *jobs.Manager, opts Options) int {
	if opts.Smoke {
		if err := Smoke(ctx, store); err != nil {
			fmt.Println(err)
			return 1
		}
		return 0
	}
	m := appModel{ctx: ctx, store: store, manager: manager, tab: "jobs", pageLimit: 20}
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		return 1
	}
	return 0
}

func Smoke(ctx context.Context, store *store.Store) error {
	m := appModel{ctx: ctx, store: store, tab: "jobs", pageLimit: 20}

	msg := fetchJobs(ctx, store, m.statusFilter, m.pageLimit, m.pageOffset)()
	if jobMsg, ok := msg.(jobsMsg); ok {
		m.jobs = jobMsg.jobs
		m.fullJobs = jobMsg.fullJobs
		if jobMsg.err != nil {
			m.err = jobMsg.err
		}
	}

	if len(m.fullJobs) > 0 {
		m.selectedJob = m.fullJobs[0]
		msg = fetchJobDetail(ctx, store, m.selectedJob.ID)()
		if detailMsg, ok := msg.(jobDetailMsg); ok {
			if detailMsg.err != nil {
				return fmt.Errorf("failed to fetch job detail: %w", detailMsg.err)
			}
			if detailMsg.job.ID != m.selectedJob.ID {
				return fmt.Errorf("job ID mismatch")
			}
		}
	}

	msg = fetchProfiles(ctx, store)()
	if profileMsg, ok := msg.(profilesMsg); ok {
		m.profiles = profileMsg.profiles
		if profileMsg.err != nil {
			m.err = profileMsg.err
		}
	}

	msg = fetchSchedules(ctx, store)()
	if scheduleMsg, ok := msg.(schedulesMsg); ok {
		m.schedules = scheduleMsg.schedules
		if scheduleMsg.err != nil {
			m.err = scheduleMsg.err
		}
	}

	msg = fetchTemplates(ctx, store)()
	if templateMsg, ok := msg.(templatesMsg); ok {
		m.templates = templateMsg.templates
		if templateMsg.err != nil {
			m.err = templateMsg.err
		}
	}

	msg = fetchCrawlStates(ctx, store)()
	if crawlStateMsg, ok := msg.(crawlStatesMsg); ok {
		m.crawlStates = crawlStateMsg.crawlStates
		if crawlStateMsg.err != nil {
			m.err = crawlStateMsg.err
		}
	}

	_ = m.View()
	return m.err
}
