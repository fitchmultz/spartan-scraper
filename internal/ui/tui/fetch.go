// Package tui provides data fetching commands for Bubble Tea.
// It defines all commands used to fetch data from the store and other backends.
package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/scheduler"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func fetchJobs(ctx context.Context, st *store.Store, statusFilter *model.Status, limit int, offset int) tea.Cmd {
	return func() tea.Msg {
		var jobs []model.Job
		var err error

		if statusFilter != nil {
			opts := store.ListByStatusOptions{Limit: limit, Offset: offset}
			jobs, err = st.ListByStatus(ctx, *statusFilter, opts)
		} else {
			opts := store.ListOptions{Limit: limit, Offset: offset}
			jobs, err = st.ListOpts(ctx, opts)
		}

		if err != nil {
			return jobsMsg{err: err}
		}
		lines := make([]string, 0, len(jobs))
		for _, job := range jobs {
			line := fmt.Sprintf("%s  %s  %s  %s", job.ID, job.Kind, job.Status, job.UpdatedAt.Format(time.RFC3339))
			lines = append(lines, line)
		}
		return jobsMsg{jobs: lines, fullJobs: jobs}
	}
}

func fetchProfiles(ctx context.Context, st *store.Store) tea.Cmd {
	return func() tea.Msg {
		vault, err := auth.LoadVault(st.DataDir())
		if err != nil {
			return profilesMsg{err: err}
		}
		names := make([]string, 0, len(vault.Profiles))
		for _, profile := range vault.Profiles {
			names = append(names, profile.Name)
		}
		return profilesMsg{profiles: names}
	}
}

func fetchSchedules(ctx context.Context, st *store.Store) tea.Cmd {
	return func() tea.Msg {
		schedules, err := scheduler.List(st.DataDir())
		if err != nil {
			return schedulesMsg{err: err}
		}
		lines := make([]string, 0, len(schedules))
		for _, sched := range schedules {
			line := fmt.Sprintf("%s  %s  next=%s  interval=%ds",
				sched.ID, sched.Kind, sched.NextRun.Format(time.RFC3339), sched.IntervalSeconds)
			lines = append(lines, line)
		}
		return schedulesMsg{schedules: lines}
	}
}

func fetchTemplates(ctx context.Context, st *store.Store) tea.Cmd {
	return func() tea.Msg {
		names, err := extract.ListTemplateNames(st.DataDir())
		if err != nil {
			return templatesMsg{err: err}
		}
		return templatesMsg{templates: names}
	}
}

func fetchCrawlStates(ctx context.Context, st *store.Store) tea.Cmd {
	return func() tea.Msg {
		opts := store.ListCrawlStatesOptions{Limit: 100, Offset: 0}
		states, err := st.ListCrawlStates(ctx, opts)
		if err != nil {
			return crawlStatesMsg{err: err}
		}
		lines := make([]string, 0, len(states))
		for _, state := range states {
			lastScraped := "never"
			if !state.LastScraped.IsZero() {
				lastScraped = state.LastScraped.Format(time.RFC3339)
			}
			line := fmt.Sprintf("%s  %s  %s", state.URL, state.ETag, lastScraped)
			lines = append(lines, line)
		}
		return crawlStatesMsg{crawlStates: lines}
	}
}

func fetchJobDetail(ctx context.Context, st *store.Store, jobID string) tea.Cmd {
	return func() tea.Msg {
		job, err := st.Get(ctx, jobID)
		return jobDetailMsg{job: job, err: err}
	}
}

func cancelJobCmd(ctx context.Context, mgr *jobs.Manager, jobID string) tea.Cmd {
	return func() tea.Msg {
		err := mgr.CancelJob(ctx, jobID)
		return jobCancelMsg{jobID: jobID, err: err}
	}
}

func tick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
