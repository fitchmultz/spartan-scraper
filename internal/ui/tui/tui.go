package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"spartan-scraper/internal/auth"
	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/jobs"
	"spartan-scraper/internal/model"
	"spartan-scraper/internal/scheduler"
	"spartan-scraper/internal/store"
)

type appModel struct {
	ctx          context.Context
	store        *store.Store
	manager      *jobs.Manager
	tab          string
	statusFilter *model.Status
	jobs         []string
	fullJobs     []model.Job
	cursor       int
	pageOffset   int
	pageLimit    int
	viewMode     string
	selectedJob  model.Job
	profiles     []string
	schedules    []string
	templates    []string
	crawlStates  []string
	err          error
	success      string
}

type tickMsg time.Time

type jobsMsg struct {
	jobs     []string
	fullJobs []model.Job
	err      error
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

type jobDetailMsg struct {
	job model.Job
	err error
}

type jobCancelMsg struct {
	jobID string
	err   error
}

type errorMsg struct {
	err error
}

type successMsg struct {
	message string
}

type Options struct {
	Smoke bool
}

var (
	headerStyle  = lipgloss.NewStyle().Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
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

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
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
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.jobs)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.fullJobs) > m.cursor {
				m.selectedJob = m.fullJobs[m.cursor]
				m.viewMode = "detail"
				return m, fetchJobDetail(m.ctx, m.store, m.selectedJob.ID)
			}
		case "left":
			if m.pageOffset > 0 {
				newOffset := m.pageOffset - m.pageLimit
				if newOffset < 0 {
					newOffset = 0
				}
				m.pageOffset = newOffset
				m.cursor = 0
				return m, fetchJobs(m.ctx, m.store, m.statusFilter, m.pageLimit, m.pageOffset)
			}
		case "right":
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
	return m, nil
}

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
