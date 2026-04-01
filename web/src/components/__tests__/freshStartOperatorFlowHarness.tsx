/**
 * Purpose: Provide shared mocks and render helpers for fresh-start operator flow route tests.
 * Responsibilities: Centralize app-shell mock wiring, route render helpers, and per-test state resets.
 * Scope: Test-only support for FreshStartOperatorFlow route suites.
 * Usage: Import setup helpers into focused FreshStartOperatorFlow Vitest files and call `setupFreshStartOperatorFlowTest()` inside the suite module.
 * Invariants/Assumptions: App-shell hook mocks stay package-local, mocked route containers remain deterministic, and callers read mutable state through the exported getters after setup runs.
 */

import { render } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeAll, beforeEach, vi } from "vitest";
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
        : "Watches"}
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

vi.mock("../../components/jobs/JobSubmissionContainer", async () => {
  const React = await import("react");
  return {
    JobSubmissionContainer: React.forwardRef(
      function MockJobSubmissionContainer(
        {
          onSubmitScrape,
        }: {
          onSubmitScrape: (input: {
            url: string;
            headless?: boolean;
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
  } as JobEntry;
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

let appDataState = createAppDataState();
let keyboardState = createKeyboardState();

export function setupFreshStartOperatorFlowTest() {
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
}

export function getAppDataState() {
  return appDataState;
}

export function getKeyboardState() {
  return keyboardState;
}

export function createJob(
  id: string,
  overrides: Partial<JobEntry> = {},
): JobEntry {
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

export function makeDetailJob(overrides: Partial<JobEntry> = {}) {
  return makeJobEntry(overrides);
}

export function setSettingsPanelMessages(options: {
  proxyMessage?: string | null;
  retentionMessage?: string | null;
}) {
  hoisted.settingsPanels.proxyMessage = options.proxyMessage ?? null;
  hoisted.settingsPanels.retentionMessage = options.retentionMessage ?? null;
}

export async function loadAppModule() {
  return import("../../App");
}

export async function loadRoutingHelpers() {
  return import("../../hooks/useAppShellRouting");
}

export async function loadApiConfigHelpers() {
  return import("../../lib/api-config");
}

export async function renderAppAt(path: string) {
  window.history.replaceState({}, "", path);
  const { App } = await loadAppModule();
  return render(<App />);
}
