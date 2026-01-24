package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"spartan-scraper/internal/store"
)

type model struct {
	ctx   context.Context
	store *store.Store
	jobs  []string
	err   error
}

type tickMsg time.Time

type jobsMsg struct {
	jobs []string
	err  error
}

type Options struct {
	Smoke bool
}

var (
	headerStyle = lipgloss.NewStyle().Bold(true)
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

func Run(ctx context.Context, store *store.Store) int {
	return RunWithOptions(ctx, store, Options{})
}

func RunWithOptions(ctx context.Context, store *store.Store, opts Options) int {
	if opts.Smoke {
		if err := Smoke(ctx, store); err != nil {
			fmt.Println(err)
			return 1
		}
		return 0
	}
	m := model{ctx: ctx, store: store}
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		return 1
	}
	return 0
}

func Smoke(ctx context.Context, store *store.Store) error {
	m := model{ctx: ctx, store: store}
	msg := fetchJobs(ctx, store)()
	if jobMsg, ok := msg.(jobsMsg); ok {
		m.jobs = jobMsg.jobs
		m.err = jobMsg.err
	}
	_ = m.View()
	return m.err
}

func (m model) Init() tea.Cmd {
	return tea.Batch(fetchJobs(m.ctx, m.store), tick())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			return m, fetchJobs(m.ctx, m.store)
		}
	case tickMsg:
		return m, tea.Batch(fetchJobs(m.ctx, m.store), tick())
	case jobsMsg:
		m.jobs = msg.jobs
		m.err = msg.err
	}
	return m, nil
}

func (m model) View() string {
	out := headerStyle.Render("Spartan Scraper Jobs") + "\n\n"
	out += "Press r to refresh, q to quit.\n\n"
	if m.err != nil {
		out += errorStyle.Render(m.err.Error()) + "\n"
	}
	if len(m.jobs) == 0 {
		out += "No jobs yet.\n"
		return out
	}
	for _, line := range m.jobs {
		out += line + "\n"
	}
	return out
}

func fetchJobs(ctx context.Context, st *store.Store) tea.Cmd {
	return func() tea.Msg {
		opts := store.ListOptions{Limit: 100, Offset: 0}
		jobs, err := st.ListOpts(ctx, opts)
		if err != nil {
			return jobsMsg{err: err}
		}
		lines := make([]string, 0, len(jobs))
		for _, job := range jobs {
			line := fmt.Sprintf("%s  %s  %s  %s", job.ID, job.Kind, job.Status, job.UpdatedAt.Format(time.RFC3339))
			lines = append(lines, line)
		}
		return jobsMsg{jobs: lines}
	}
}

func tick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
