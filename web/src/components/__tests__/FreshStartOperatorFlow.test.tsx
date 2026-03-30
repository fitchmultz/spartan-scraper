/**
 * Purpose: Lock the fresh-start-to-daily-use operator journey into deterministic browser regression coverage.
 * Responsibilities: Verify first-run onboarding retirement, Settings overview gating, route-help persistence, command-palette path navigation, first-job redirect flow, and exported route helpers.
 * Scope: App-shell integration coverage in Vitest/jsdom only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: The shell can be exercised deterministically with mocked data hooks and lightweight route-container doubles while onboarding persistence and route helpers remain real.
 */

import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import {
  afterEach,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from "vitest";
import type { JobEntry } from "../../types";

const hoisted = vi.hoisted(() => ({
  useAppData: vi.fn(),
  useFormState: vi.fn(),
  useResultsState: vi.fn(),
  useTheme: vi.fn(),
  usePresets: vi.fn(),
  useKeyboard: vi.fn(),
  postV1Scrape: vi.fn(),
  postV1Crawl: vi.fn(),
  postV1Research: vi.fn(),
  deleteV1JobsById: vi.fn(),
  toast: {
    show: vi.fn(() => "toast-1"),
    update: vi.fn(),
    confirm: vi.fn(async () => true),
  },
  aiAssistant: {
    open: vi.fn(),
  },
  inventory: {
    renderProfiles: 0,
    pipelineScripts: 0,
  },
  settingsPanels: {
    proxyMessage: null as string | null,
    retentionMessage: null as string | null,
  },
}));

const storage = new Map<string, string>();

vi.mock("../../hooks/useAppData", () => ({
  useAppData: hoisted.useAppData,
}));

vi.mock("../../hooks/useFormState", () => ({
  useFormState: hoisted.useFormState,
}));

vi.mock("../../hooks/useResultsState", () => ({
  useResultsState: hoisted.useResultsState,
}));

vi.mock("../../hooks/useTheme", () => ({
  useTheme: hoisted.useTheme,
}));

vi.mock("../../hooks/usePresets", () => ({
  usePresets: hoisted.usePresets,
}));

vi.mock("../../hooks/useKeyboard", () => ({
  useKeyboard: hoisted.useKeyboard,
}));

vi.mock("../../api", () => ({
  postV1Scrape: hoisted.postV1Scrape,
  postV1Crawl: hoisted.postV1Crawl,
  postV1Research: hoisted.postV1Research,
  deleteV1JobsById: hoisted.deleteV1JobsById,
}));

vi.mock("../../components/toast", () => ({
  useToast: () => hoisted.toast,
}));

vi.mock("../../components/ai-assistant", () => ({
  AIAssistantProvider: ({ children }: { children: ReactNode }) => children,
  useAIAssistant: () => hoisted.aiAssistant,
}));

vi.mock("../../components/InfoSections", () => ({
  InfoSections: () => <div data-testid="info-sections" />,
}));

vi.mock("../../components/KeyboardShortcutsHelp", () => ({
  KeyboardShortcutsHelp: ({ isOpen }: { isOpen: boolean }) =>
    isOpen ? <div data-testid="keyboard-help">Keyboard help</div> : null,
}));

vi.mock("../../components/OnboardingFlow", () => ({
  OnboardingFlow: () => null,
}));

vi.mock("../../components/SystemStatusPanel", () => ({
  SystemStatusPanel: () => <div data-testid="system-status-panel" />,
}));

vi.mock("../../components/automation/AutomationLayout", () => ({
  AutomationLayout: ({
    activeSection,
    renderSection,
  }: {
    activeSection: string;
    renderSection: (section: string) => ReactNode;
  }) => (
    <section data-testid="automation-layout">
      <div data-testid="automation-active-section">{activeSection}</div>
      {renderSection(activeSection)}
    </section>
  ),
}));

vi.mock("../../components/automation/AutomationSubnav", () => ({
  AutomationSubnav: ({ activeSection }: { activeSection: string }) => (
    <div data-testid="automation-subnav">{activeSection}</div>
  ),
}));

vi.mock("../../components/watches/WatchContainer", () => ({
  WatchContainer: ({
    promotionSeed,
  }: {
    promotionSeed?: { source?: { jobId?: string } } | null;
  }) => (
    <div>
      {promotionSeed?.source?.jobId
        ? `Watch seed ${promotionSeed.source.jobId}`
        : "Watch container"}
    </div>
  ),
}));

vi.mock("../../components/export-schedules/ExportScheduleContainer", () => ({
  ExportScheduleContainer: ({
    promotionSeed,
  }: {
    promotionSeed?: { source?: { jobId?: string } } | null;
  }) => (
    <div>
      {promotionSeed?.source?.jobId
        ? `Export seed ${promotionSeed.source.jobId}`
        : "Export schedules"}
    </div>
  ),
}));

vi.mock("../../components/webhooks/WebhookDeliveryContainer", () => ({
  WebhookDeliveryContainer: () => <div>Webhook deliveries</div>,
}));

vi.mock("../../components/RetentionStatusPanel", () => ({
  RetentionStatusPanel: () => (
    <div data-testid="retention-status">
      {hoisted.settingsPanels.retentionMessage ? (
        <div className="error">{hoisted.settingsPanels.retentionMessage}</div>
      ) : null}
    </div>
  ),
}));

vi.mock("../../components/ProxyPoolStatusPanel", () => ({
  ProxyPoolStatusPanel: () => (
    <div data-testid="proxy-pool-status">
      {hoisted.settingsPanels.proxyMessage ? (
        <div className="error">{hoisted.settingsPanels.proxyMessage}</div>
      ) : null}
    </div>
  ),
}));

vi.mock("../../components/chains/ChainContainer", () => ({
  ChainContainer: () => <div>Chain container</div>,
}));

vi.mock("../../components/batches/BatchContainer", () => ({
  BatchContainer: () => <div>Batch container</div>,
}));

vi.mock("../../components/templates/TemplateManager", () => ({
  TemplateManager: ({
    promotionSeed,
  }: {
    promotionSeed?: { source?: { jobId?: string } } | null;
  }) => (
    <div data-testid="templates-workspace">
      {promotionSeed?.source?.jobId
        ? `Template seed ${promotionSeed.source.jobId}`
        : "Template manager"}
    </div>
  ),
}));

vi.mock("../../components/presets/PresetContainer", () => ({
  PresetContainer: () => <aside data-testid="preset-container" />,
}));

vi.mock("../../components/jobs/JobMonitoringDashboard", () => ({
  JobMonitoringDashboard: ({ jobs }: { jobs: JobEntry[] }) => (
    <section data-testid="jobs-dashboard" data-tour="jobs-dashboard">
      <h2>Jobs dashboard</h2>
      {jobs.map((job) => (
        <div key={job.id}>{job.id}</div>
      ))}
    </section>
  ),
}));

vi.mock("../../components/results/ResultsContainer", () => ({
  ResultsContainer: ({
    currentJob,
    onPromote,
  }: {
    currentJob?: { id?: string } | null;
    onPromote?: (
      destination: "template" | "watch" | "export-schedule",
      options?: {
        preferredExportFormat?: "json" | "jsonl" | "md" | "csv" | "xlsx";
      },
    ) => void;
  }) => (
    <div data-testid="results-container">
      <div>{currentJob?.id ?? "no-job"}</div>
      <button type="button" onClick={() => onPromote?.("template")}>
        Promote template
      </button>
      <button type="button" onClick={() => onPromote?.("watch")}>
        Promote watch
      </button>
      <button
        type="button"
        onClick={() =>
          onPromote?.("export-schedule", { preferredExportFormat: "md" })
        }
      >
        Promote export
      </button>
    </div>
  ),
}));

vi.mock("../../components/render-profiles", async () => {
  const React = await import("react");

  return {
    RenderProfileEditor: ({
      onInventoryChange,
    }: {
      onInventoryChange?: (count: number) => void;
    }) => {
      React.useEffect(() => {
        onInventoryChange?.(hoisted.inventory.renderProfiles);
      }, [onInventoryChange]);

      return <div data-testid="render-profile-editor" />;
    },
  };
});

vi.mock("../../components/pipeline-js/PipelineJSEditor", async () => {
  const React = await import("react");

  return {
    PipelineJSEditor: ({
      onInventoryChange,
    }: {
      onInventoryChange?: (count: number) => void;
    }) => {
      React.useEffect(() => {
        onInventoryChange?.(hoisted.inventory.pipelineScripts);
      }, [onInventoryChange]);

      return <div data-testid="pipeline-js-editor" />;
    },
  };
});

vi.mock("../../components/shell/ShellPrimitives", () => ({
  AppTopBar: ({
    navItems,
    onNavigate,
    globalAction,
    utilities,
  }: {
    navItems: Array<{ label: string; path: string }>;
    onNavigate: (path: string) => void;
    globalAction?: ReactNode;
    utilities?: ReactNode;
  }) => (
    <header>
      <nav aria-label="Primary">
        {navItems.map((item) => (
          <button
            key={item.path}
            type="button"
            onClick={() => onNavigate(item.path)}
          >
            {item.label}
          </button>
        ))}
      </nav>
      {globalAction}
      {utilities}
    </header>
  ),
  RouteHeader: ({
    title,
    description,
    actions,
    subnav,
  }: {
    title: string;
    description?: string;
    actions?: ReactNode;
    subnav?: ReactNode;
  }) => (
    <section>
      <h1>{title}</h1>
      {description ? <p>{description}</p> : null}
      {actions}
      {subnav}
    </section>
  ),
  RouteSignals: () => null,
}));

vi.mock("../../components/ThemeToggle", () => ({
  ThemeToggle: () => <button type="button">Theme</button>,
}));

vi.mock("../../components/TutorialTooltip", () => ({
  TutorialTooltip: () => null,
}));

vi.mock("../../components/jobs/JobSubmissionContainer", async () => {
  const React = await import("react");

  return {
    JobSubmissionContainer: React.forwardRef(
      function MockJobSubmissionContainer(
        {
          onSubmitScrape,
        }: {
          onSubmitScrape: (request: {
            url: string;
            headless: boolean;
          }) => Promise<void>;
        },
        ref,
      ) {
        React.useImperativeHandle(ref, () => ({
          submitScrape: async () => {},
          submitCrawl: async () => {},
          submitResearch: async () => {},
          setScrapeUrl: () => {},
          setCrawlUrl: () => {},
          setResearchQuery: () => {},
          getScrapeUrl: () => "",
          getCrawlUrl: () => "",
          getCurrentConfig: () => ({ jobType: "scrape" }),
          applyPreset: () => {},
          clearDraft: () => {},
        }));

        return (
          <section
            data-testid="job-submission-container"
            data-tour="job-wizard-header"
          >
            <div data-tour="wizard-steps">Wizard steps</div>
            <button
              type="button"
              onClick={() =>
                void onSubmitScrape({
                  url: "https://fixture.example.test/page",
                  headless: false,
                })
              }
            >
              Submit first scrape
            </button>
          </section>
        );
      },
    ),
  };
});

import { App } from "../../App";
import { normalizePath, parseRoute } from "../../hooks/useAppShellRouting";
import { getApiBaseUrl } from "../../lib/api-config";

function createJob(id: string, overrides: Partial<JobEntry> = {}): JobEntry {
  return {
    id,
    kind: "scrape",
    status: "succeeded",
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    specVersion: 1,
    spec: {},
    run: {
      waitMs: 0,
      runMs: 0,
      totalMs: 0,
    },
    ...overrides,
  } as JobEntry;
}

function buildHealth() {
  return {
    status: "ok",
    version: "test",
    components: {
      browser: { status: "ok", message: "Browser automation is ready." },
      ai: {
        status: "disabled",
        message: "AI helpers are optional and currently disabled.",
      },
      proxy_pool: {
        status: "disabled",
        message: "Proxy pooling is disabled.",
      },
      retention: {
        status: "disabled",
        message: "Retention is disabled.",
      },
    },
    notices: [],
  };
}

function makeJobEntry(overrides: Partial<JobEntry> = {}): JobEntry {
  return {
    id: "job-1",
    kind: "scrape",
    status: "succeeded",
    createdAt: "2026-03-20T10:00:00Z",
    updatedAt: "2026-03-20T10:05:00Z",
    finishedAt: "2026-03-20T10:05:00Z",
    specVersion: 1,
    spec: {
      version: 1,
      url: "https://example.com/pricing",
      execution: {
        headless: true,
        playwright: true,
        screenshot: { enabled: true, fullPage: true, format: "png" },
      },
    },
    run: {
      waitMs: 0,
      runMs: 1000,
      totalMs: 1000,
    },
    ...overrides,
  };
}

function createAppDataState() {
  return {
    jobs: [] as JobEntry[],
    failedJobs: [] as JobEntry[],
    jobStatusFilter: "",
    profiles: [] as Array<{ name: string; parents: string[] }>,
    schedules: [] as Array<{
      id: string;
      kind: string;
      intervalSeconds: number;
    }>,
    templates: [] as Array<{ name?: string }>,
    crawlStates: [] as unknown[],
    managerStatus: null,
    jobsTotal: 0,
    jobsPage: 1,
    crawlStatesTotal: 0,
    crawlStatesPage: 1,
    error: null,
    loading: false,
    connectionState: "connected" as const,
    health: buildHealth(),
    setupRequired: false,
    detailJob: null as JobEntry | null,
    detailJobLoading: false,
    detailJobError: null as string | null,
    refreshHealth: vi.fn(async () => buildHealth()),
    refreshJobs: vi.fn(async () => {}),
    refreshTemplates: vi.fn(async () => {}),
    refreshJobDetail: vi.fn(async () => null),
    clearJobDetail: vi.fn(),
    setJobsPage: vi.fn(),
    setCrawlStatesPage: vi.fn(),
    setJobStatusFilter: vi.fn(),
  };
}

function createKeyboardState() {
  return {
    isCommandPaletteOpen: false,
    isHelpOpen: false,
    openCommandPalette: vi.fn(),
    closeCommandPalette: vi.fn(),
    openHelp: vi.fn(),
    closeHelp: vi.fn(),
    shortcuts: {
      commandPalette: "mod+k",
      submitForm: "mod+enter",
      search: "/",
      help: "?",
      escape: "escape",
      navigateJobs: "g j",
      navigateResults: "g r",
      navigateForms: "g f",
    },
    isMac: false,
  };
}

function renderAppAt(path: string) {
  window.history.replaceState({}, "", path);
  return render(<App />);
}

describe("FreshStartOperatorFlow", () => {
  beforeAll(() => {
    const localStorageMock = {
      getItem: (key: string) => storage.get(key) ?? null,
      setItem: (key: string, value: string) => {
        storage.set(key, value);
      },
      removeItem: (key: string) => {
        storage.delete(key);
      },
      clear: () => {
        storage.clear();
      },
    };

    vi.stubGlobal("localStorage", localStorageMock);
    Object.defineProperty(window, "localStorage", {
      value: localStorageMock,
      configurable: true,
    });
  });

  let appDataState: ReturnType<typeof createAppDataState>;
  let keyboardState: ReturnType<typeof createKeyboardState>;

  beforeEach(() => {
    storage.clear();
    vi.clearAllMocks();

    vi.stubGlobal("requestAnimationFrame", (cb: FrameRequestCallback) => {
      cb(0);
      return 1;
    });
    vi.stubGlobal("cancelAnimationFrame", vi.fn());
    vi.stubGlobal(
      "ResizeObserver",
      class {
        observe() {}
        unobserve() {}
        disconnect() {}
      },
    );
    Object.defineProperty(Element.prototype, "scrollIntoView", {
      configurable: true,
      value: vi.fn(),
    });

    hoisted.inventory.renderProfiles = 0;
    hoisted.inventory.pipelineScripts = 0;
    hoisted.settingsPanels.proxyMessage = null;
    hoisted.settingsPanels.retentionMessage = null;

    appDataState = createAppDataState();
    keyboardState = createKeyboardState();

    hoisted.useAppData.mockImplementation(() => appDataState);
    hoisted.useFormState.mockImplementation(() => ({
      extractTemplate: "",
    }));
    hoisted.useResultsState.mockImplementation(() => ({
      selectedJobId: null,
      loadResults: vi.fn(async () => {}),
      totalResults: 0,
      resultFormat: "json",
    }));
    hoisted.useTheme.mockImplementation(() => ({
      theme: "system",
      resolvedTheme: "system",
      setTheme: vi.fn(),
      toggleTheme: vi.fn(),
    }));
    hoisted.usePresets.mockImplementation(() => ({
      presets: [],
      savePreset: vi.fn(),
    }));
    hoisted.useKeyboard.mockImplementation(() => keyboardState);

    hoisted.postV1Scrape.mockResolvedValue({});
    hoisted.postV1Crawl.mockResolvedValue({});
    hoisted.postV1Research.mockResolvedValue({});
    hoisted.deleteV1JobsById.mockResolvedValue({});
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows the first-run nudge on a pristine workspace and retires it after work starts", async () => {
    const rendered = renderAppAt("/jobs");

    expect(
      screen.getByRole("heading", { name: /start with one working job/i }),
    ).toBeInTheDocument();

    appDataState.jobsTotal = 1;
    rendered.rerender(<App />);

    await waitFor(() => {
      expect(
        screen.queryByRole("heading", { name: /start with one working job/i }),
      ).not.toBeInTheDocument();
    });

    rendered.unmount();
  });

  it("loads authoritative job detail for direct result routes outside the paged jobs list", async () => {
    appDataState.detailJob = makeJobEntry({ id: "job-direct" });

    renderAppAt("/jobs/job-direct");

    await waitFor(() => {
      expect(appDataState.refreshJobDetail).toHaveBeenCalledWith("job-direct");
    });

    expect(screen.getByText("job-direct")).toBeInTheDocument();
  });

  it("keeps saved results ahead of secondary framing on the results route", async () => {
    appDataState.detailJob = makeJobEntry({ id: "job-results-first" });

    renderAppAt("/jobs/job-results-first");

    const results = await screen.findByTestId("results-container");
    const routeHelp = screen.getByLabelText(
      /what can i do here\? for this route/i,
    );

    expect(
      screen.queryByText(
        /read saved output first, then open comparison, transform, and export tools only when needed/i,
      ),
    ).not.toBeInTheDocument();
    expect(
      results.compareDocumentPosition(routeHelp) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
  });

  it("keeps job detail errors focused on the recovery path", async () => {
    appDataState.detailJobError = "invalid job id format: bad-job";

    renderAppAt("/jobs/bad-job");

    expect(
      await screen.findByRole("heading", {
        name: /unable to load this saved job/i,
      }),
    ).toBeInTheDocument();
    expect(screen.queryByLabelText(/result context/i)).not.toBeInTheDocument();
    expect(
      screen.queryByLabelText(/what can i do here\? for this route/i),
    ).not.toBeInTheDocument();
    expect(
      screen.getAllByRole("button", { name: /back to jobs/i }).length,
    ).toBeGreaterThan(0);
  });

  it("hands off promotion drafts from results into the canonical destination workspaces", async () => {
    const user = userEvent.setup();
    appDataState.detailJob = makeJobEntry({ id: "job-promote" });

    renderAppAt("/jobs/job-promote");

    await waitFor(() => {
      expect(screen.getByText("job-promote")).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: /promote watch/i }));

    await waitFor(() => {
      expect(window.location.pathname).toBe("/automation/watches");
    });
    expect(screen.getByText("Watch seed job-promote")).toBeInTheDocument();
  });

  it("hands template promotion drafts off to the templates workspace", async () => {
    const user = userEvent.setup();
    appDataState.detailJob = makeJobEntry({ id: "job-template" });

    renderAppAt("/jobs/job-template");

    await waitFor(() => {
      expect(screen.getByText("job-template")).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: /promote template/i }));

    await waitFor(() => {
      expect(window.location.pathname).toBe("/templates");
    });
    expect(screen.getByText("Template seed job-template")).toBeInTheDocument();
  });

  it("keeps the new job workspace ahead of first-run framing", async () => {
    renderAppAt("/jobs/new");

    expect(
      screen.queryByRole("heading", { name: /start with one working job/i }),
    ).not.toBeInTheDocument();

    const wizard = screen.getByTestId("job-submission-container");
    const firstRunNotice = screen.getByText(/start with a single page scrape/i);
    const routeHelp = screen.getByLabelText(
      /what can i do here\? for this route/i,
    );

    expect(
      wizard.compareDocumentPosition(firstRunNotice) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    expect(
      firstRunNotice.compareDocumentPosition(routeHelp) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
  });

  it("shows first-visit Settings guidance, then retires the overview after the first job", async () => {
    const user = userEvent.setup();

    const firstRender = renderAppAt("/settings");

    await waitFor(() => {
      expect(window.location.pathname).toBe("/settings/authoring");
    });

    expect(
      screen.queryByRole("heading", { name: /start with one working job/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /show details/i }),
    ).toBeInTheDocument();
    const settingsOverview = await screen.findByText(
      /most settings controls can wait until a workflow proves it needs them/i,
    );
    expect(settingsOverview).toBeInTheDocument();
    expect(
      screen.getByRole("navigation", { name: /settings sections/i }),
    ).toBeInTheDocument();
    const authoringHeading = screen.getByRole("heading", {
      name: /authoring tools/i,
    });
    expect(authoringHeading).toBeInTheDocument();
    expect(
      screen.queryByRole("heading", { name: /saved state and history/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("heading", { name: /operational controls/i }),
    ).not.toBeInTheDocument();
    expect(screen.getByTestId("render-profile-editor")).toBeInTheDocument();
    expect(screen.getByTestId("pipeline-js-editor")).toBeInTheDocument();
    expect(screen.queryByTestId("proxy-pool-status")).not.toBeInTheDocument();
    expect(screen.queryByTestId("retention-status")).not.toBeInTheDocument();

    const routeHelp = screen.getByLabelText(
      /what can i do here\? for this route/i,
    );
    expect(
      authoringHeading.compareDocumentPosition(settingsOverview) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    expect(
      settingsOverview.compareDocumentPosition(routeHelp) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();

    await user.click(
      within(routeHelp).getByRole("button", { name: /^create job$/i }),
    );

    await waitFor(() => {
      expect(window.location.pathname).toBe("/jobs/new");
    });

    firstRender.unmount();

    const secondRender = renderAppAt("/settings/authoring");

    expect(
      screen.getByLabelText(/what can i do here\? for this route/i),
    ).toBeInTheDocument();

    appDataState.jobsTotal = 1;
    secondRender.rerender(<App />);

    await waitFor(() => {
      expect(
        screen.queryByText(
          /most settings controls can wait until a workflow proves it needs them/i,
        ),
      ).not.toBeInTheDocument();
    });
  });

  it("binds Settings section selection to the canonical URL and browser history", async () => {
    const user = userEvent.setup();

    renderAppAt("/settings/authoring");

    const authoringButton = screen.getByRole("button", {
      name: /authoring tools/i,
    });
    const operationsButton = screen.getByRole("button", {
      name: /operations/i,
    });

    expect(authoringButton).toHaveAttribute("aria-current", "page");
    expect(operationsButton).not.toHaveAttribute("aria-current");
    expect(screen.getByTestId("render-profile-editor")).toBeInTheDocument();
    expect(screen.queryByTestId("proxy-pool-status")).not.toBeInTheDocument();

    await user.click(operationsButton);

    await waitFor(() => {
      expect(window.location.pathname).toBe("/settings/operations");
    });

    expect(operationsButton).toHaveAttribute("aria-current", "page");
    expect(authoringButton).not.toHaveAttribute("aria-current");
    expect(
      screen.getByRole("heading", { name: /operational controls/i }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("proxy-pool-status")).toBeInTheDocument();
    expect(
      screen.queryByTestId("render-profile-editor"),
    ).not.toBeInTheDocument();

    window.history.back();

    await waitFor(() => {
      expect(window.location.pathname).toBe("/settings/authoring");
    });

    expect(authoringButton).toHaveAttribute("aria-current", "page");
    expect(operationsButton).not.toHaveAttribute("aria-current");
    expect(screen.getByTestId("render-profile-editor")).toBeInTheDocument();
    expect(screen.queryByTestId("proxy-pool-status")).not.toBeInTheDocument();
  });

  it("keeps shared Settings chrome visible when Operations panels fail locally", () => {
    hoisted.settingsPanels.proxyMessage =
      "Proxy pool metadata could not be loaded.";
    hoisted.settingsPanels.retentionMessage =
      "Retention metadata could not be loaded.";

    renderAppAt("/settings/operations");

    expect(
      screen.getByRole("heading", { name: /operational controls/i }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("proxy-pool-status")).toHaveTextContent(
      "Proxy pool metadata could not be loaded.",
    );
    expect(screen.getByTestId("retention-status")).toHaveTextContent(
      "Retention metadata could not be loaded.",
    );
    expect(
      screen.getByRole("heading", { name: /^settings$/i }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText(/settings sections/i)).toBeInTheDocument();
    expect(
      screen.getByLabelText(/what can i do here\? for this route/i),
    ).toBeInTheDocument();
  });

  it("navigates to every major route from the command palette", async () => {
    const user = userEvent.setup();

    appDataState.jobs = [createJob("job-12345678")];
    appDataState.jobsTotal = 1;
    keyboardState.isCommandPaletteOpen = true;

    renderAppAt("/jobs");

    const palette = screen.getByRole("dialog", { name: /command palette/i });

    const routeCases = [
      ["Open Jobs", "/jobs", "Jobs"],
      ["Create Job", "/jobs/new", "Create Job"],
      ["Open Templates", "/templates", "Templates"],
      ["Open Settings", "/settings/authoring", "Settings"],
      ["Open Automation / Batches", "/automation/batches", "Automation"],
      ["Open Automation / Chains", "/automation/chains", "Automation"],
      ["Open Automation / Watches", "/automation/watches", "Automation"],
      ["Open Automation / Exports", "/automation/exports", "Automation"],
      ["Open Automation / Webhooks", "/automation/webhooks", "Automation"],
    ] as const;

    for (const [label, path, heading] of routeCases) {
      await user.click(within(palette).getByText(label));

      await waitFor(() => {
        expect(window.location.pathname).toBe(path);
      });

      expect(
        screen.getByRole("heading", { name: heading }),
      ).toBeInTheDocument();
    }

    expect(screen.getByTestId("automation-active-section")).toHaveTextContent(
      "webhooks",
    );
  });

  it("reopens the command palette with a fresh search input", async () => {
    const user = userEvent.setup();

    keyboardState.isCommandPaletteOpen = true;
    const rendered = renderAppAt("/jobs");

    await user.type(screen.getByLabelText("Search commands"), "template");
    expect(screen.getByLabelText("Search commands")).toHaveValue("template");

    keyboardState.isCommandPaletteOpen = false;
    rendered.rerender(<App />);

    await waitFor(() => {
      expect(
        screen.queryByRole("dialog", { name: /command palette/i }),
      ).not.toBeInTheDocument();
    });

    keyboardState.isCommandPaletteOpen = true;
    rendered.rerender(<App />);

    await waitFor(() => {
      expect(screen.getByLabelText("Search commands")).toHaveValue("");
    });
  });

  it("submits the first job from the New Job route, redirects to Jobs, and shows the new run", async () => {
    const user = userEvent.setup();

    appDataState.refreshJobs = vi.fn(async () => {
      appDataState.jobs = [createJob("job-first-run-0001")];
      appDataState.jobsTotal = 1;
    });

    renderAppAt("/jobs/new");

    await user.click(
      screen.getByRole("button", { name: /submit first scrape/i }),
    );

    await waitFor(() => {
      expect(window.location.pathname).toBe("/jobs");
    });

    expect(appDataState.refreshJobs).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("jobs-dashboard")).toHaveTextContent(
      "job-first-run-0001",
    );
  });

  it("keeps browser API base URL resolution deterministic in test mode", () => {
    expect(getApiBaseUrl()).toBe("");
  });

  it("normalizes paths, resolves Settings sections, and falls back unknown routes to Jobs", () => {
    expect(normalizePath("")).toBe("/jobs");
    expect(normalizePath("/")).toBe("/jobs");
    expect(normalizePath("/settings///")).toBe("/settings");
    expect(normalizePath("/jobs/new/")).toBe("/jobs/new");

    expect(parseRoute("/settings/operations")).toMatchObject({
      kind: "settings",
      path: "/settings/operations",
      settingsSection: "operations",
    });
    expect(parseRoute("/settings/unknown")).toMatchObject({
      kind: "settings",
      path: "/settings/unknown",
      settingsSection: "authoring",
    });
    expect(parseRoute("/mystery-path")).toMatchObject({
      kind: "jobs",
      path: "/jobs",
    });
  });
});
