package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"spartan-scraper/internal/auth"
	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/model"
	"spartan-scraper/internal/scheduler"
	"spartan-scraper/internal/store"
)

type appModel struct {
	ctx          context.Context
	store        *store.Store
	tab          string
	statusFilter *model.Status
	jobs         []string
	profiles     []string
	schedules    []string
	templates    []string
	crawlStates  []string
	err          error
}

type tickMsg time.Time

type jobsMsg struct {
	jobs []string
	err  error
}

type profilesMsg struct {
	profiles []string
	err      error
}

type schedulesMsg struct {
	schedules []string
	err       error
}

type templatesMsg struct {
	templates []string
	err       error
}

type crawlStatesMsg struct {
	crawlStates []string
	err         error
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
	m := appModel{ctx: ctx, store: store, tab: "jobs"}
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		return 1
	}
	return 0
}

func Smoke(ctx context.Context, store *store.Store) error {
	m := appModel{ctx: ctx, store: store, tab: "jobs"}

	msg := fetchJobs(ctx, store, m.statusFilter)()
	if jobMsg, ok := msg.(jobsMsg); ok {
		m.jobs = jobMsg.jobs
		if jobMsg.err != nil {
			m.err = jobMsg.err
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

func (m appModel) Init() tea.Cmd {
	return tea.Batch(
		fetchJobs(m.ctx, m.store, m.statusFilter),
		fetchProfiles(m.ctx, m.store),
		fetchSchedules(m.ctx, m.store),
		fetchTemplates(m.ctx, m.store),
		fetchCrawlStates(m.ctx, m.store),
		tick(),
	)
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			return m, tea.Batch(
				fetchJobs(m.ctx, m.store, m.statusFilter),
				fetchProfiles(m.ctx, m.store),
				fetchSchedules(m.ctx, m.store),
				fetchTemplates(m.ctx, m.store),
				fetchCrawlStates(m.ctx, m.store),
			)
		case "1":
			m.tab = "jobs"
			return m, nil
		case "2":
			m.tab = "profiles"
			return m, nil
		case "3":
			m.tab = "schedules"
			return m, nil
		case "4":
			m.tab = "templates"
			return m, nil
		case "5":
			m.tab = "crawl-states"
			return m, nil
		case "f":
			m.statusFilter = cycleStatusFilter(m.statusFilter)
			return m, fetchJobs(m.ctx, m.store, m.statusFilter)
		}
	case tickMsg:
		return m, tea.Batch(fetchJobs(m.ctx, m.store, m.statusFilter), tick())
	case jobsMsg:
		m.jobs = msg.jobs
		if msg.err != nil {
			m.err = msg.err
		}
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
	}
	return m, nil
}

func (m appModel) View() string {
	out := headerStyle.Render("Spartan Scraper") + "\n\n"
	out += "Tabs: [1] Jobs [2] Profiles [3] Schedules [4] Templates [5] Crawl States | r=refresh, f=filter, q=quit\n\n"

	if m.err != nil {
		out += errorStyle.Render(m.err.Error()) + "\n\n"
	}

	switch m.tab {
	case "jobs":
		out += headerStyle.Render("Jobs")
		if m.statusFilter != nil {
			out += fmt.Sprintf(" (filter: %s)", *m.statusFilter)
		}
		out += "\n\n"
		if len(m.jobs) == 0 {
			out += "No jobs found.\n"
			return out
		}
		for _, line := range m.jobs {
			out += line + "\n"
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

func fetchJobs(ctx context.Context, st *store.Store, statusFilter *model.Status) tea.Cmd {
	return func() tea.Msg {
		var jobs []model.Job
		var err error

		if statusFilter != nil {
			opts := store.ListByStatusOptions{Limit: 100, Offset: 0}
			jobs, err = st.ListByStatus(ctx, *statusFilter, opts)
		} else {
			opts := store.ListOptions{Limit: 100, Offset: 0}
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
		return jobsMsg{jobs: lines}
	}
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

func tick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
