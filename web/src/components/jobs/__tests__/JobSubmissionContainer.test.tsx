/**
 * Purpose: Verify guided job-wizard orchestration, draft persistence, and expert-mode continuity without depending on the full downstream form implementations.
 * Responsibilities: Mock expert forms, exercise wizard navigation and persistence, and assert that guided mode remains the default while expert mode preserves data.
 * Scope: Component coverage for `JobSubmissionContainer`.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Drafts persist in localStorage per job type, expert mode shares the same underlying data as guided mode, and the mocked forms only need imperative parity with the real form refs.
 */

import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState, type ForwardedRef } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AIAssistantProvider } from "../../ai-assistant";
import { useFormState } from "../../../hooks/useFormState";
import { JobSubmissionContainer } from "../JobSubmissionContainer";
import type { JobType } from "../../../types/presets";

interface MockScrapeFormProps {
  surface?: "full" | "headless";
  url: string;
  setUrl: (value: string) => void;
}

interface MockCrawlFormProps {
  surface?: "full" | "headless";
  url: string;
  setUrl: (value: string) => void;
}

interface MockResearchFormProps {
  surface?: "full" | "headless";
  query: string;
  urls: string;
  setQuery: (value: string) => void;
  setUrls: (value: string) => void;
}

const DEFAULT_VIEWPORT_HEIGHT = 900;

function installMockLocalStorage() {
  const store = new Map<string, string>();
  const mockStorage: Storage = {
    get length() {
      return store.size;
    },
    clear() {
      store.clear();
    },
    getItem(key) {
      return store.has(key) ? (store.get(key) ?? null) : null;
    },
    key(index) {
      return Array.from(store.keys())[index] ?? null;
    },
    removeItem(key) {
      store.delete(key);
    },
    setItem(key, value) {
      store.set(key, String(value));
    },
  };

  Object.defineProperty(window, "localStorage", {
    configurable: true,
    value: mockStorage,
  });
  Object.defineProperty(globalThis, "localStorage", {
    configurable: true,
    value: mockStorage,
  });
}

vi.mock("../../ScrapeForm", async () => {
  const React = await import("react");
  return {
    ScrapeForm: React.forwardRef(function MockScrapeForm(
      props: MockScrapeFormProps,
      ref: ForwardedRef<unknown>,
    ) {
      React.useImperativeHandle(ref, () => ({
        submit: vi.fn().mockResolvedValue(undefined),
        getUrl: () => props.url,
        setUrl: props.setUrl,
        getConfig: () => ({ url: props.url }),
      }));

      if (props.surface === "headless") {
        return null;
      }

      return (
        <label>
          Target URL
          <input
            value={props.url}
            onChange={(event) => props.setUrl(event.target.value)}
          />
        </label>
      );
    }),
  };
});

vi.mock("../../CrawlForm", async () => {
  const React = await import("react");
  return {
    CrawlForm: React.forwardRef(function MockCrawlForm(
      props: MockCrawlFormProps,
      ref: ForwardedRef<unknown>,
    ) {
      React.useImperativeHandle(ref, () => ({
        submit: vi.fn().mockResolvedValue(undefined),
        getUrl: () => props.url,
        setUrl: props.setUrl,
        getConfig: () => ({ url: props.url }),
      }));

      if (props.surface === "headless") {
        return null;
      }

      return (
        <label>
          Root URL
          <input
            value={props.url}
            onChange={(event) => props.setUrl(event.target.value)}
          />
        </label>
      );
    }),
  };
});

vi.mock("../../ResearchForm", async () => {
  const React = await import("react");
  return {
    ResearchForm: React.forwardRef(function MockResearchForm(
      props: MockResearchFormProps,
      ref: ForwardedRef<unknown>,
    ) {
      React.useImperativeHandle(ref, () => ({
        submit: vi.fn().mockResolvedValue(undefined),
        getQuery: () => props.query,
        setQuery: props.setQuery,
        getConfig: () => ({ query: props.query, urls: props.urls }),
      }));

      if (props.surface === "headless") {
        return null;
      }

      return (
        <>
          <label>
            Research query
            <input
              value={props.query}
              onChange={(event) => props.setQuery(event.target.value)}
            />
          </label>
          <label>
            Source URLs
            <textarea
              value={props.urls}
              onChange={(event) => props.setUrls(event.target.value)}
            />
          </label>
        </>
      );
    }),
  };
});

function renderHarness(
  initialTab: JobType = "scrape",
  { includePresets = false }: { includePresets?: boolean } = {},
) {
  const onSubmitScrape = vi.fn();
  const onSubmitCrawl = vi.fn();
  const onSubmitResearch = vi.fn();
  const savePreset = vi.fn();
  const onSelectPreset = vi.fn();

  function Harness() {
    const formState = useFormState();
    const [activeTab, setActiveTab] = useState<JobType>(initialTab);

    return (
      <AIAssistantProvider>
        <JobSubmissionContainer
          activeTab={activeTab}
          setActiveTab={setActiveTab}
          formState={formState}
          onSubmitScrape={onSubmitScrape}
          onSubmitCrawl={onSubmitCrawl}
          onSubmitResearch={onSubmitResearch}
          loading={false}
          profiles={[]}
          presets={
            includePresets
              ? [
                  {
                    id: "preset-scrape-static-page",
                    name: "Static Page",
                    description: "Fast HTTP fetch for static HTML content",
                    icon: "🧪",
                    jobType: "scrape",
                    config: { url: "https://example.com" },
                    resources: {
                      timeSeconds: 10,
                      cpu: "low",
                      memory: "low",
                    },
                    useCases: ["Static marketing pages"],
                    isBuiltIn: true,
                  },
                ]
              : undefined
          }
          savePreset={includePresets ? savePreset : undefined}
          onSelectPreset={includePresets ? onSelectPreset : undefined}
        />
      </AIAssistantProvider>
    );
  }

  return {
    ...render(<Harness />),
    onSubmitScrape,
    onSubmitCrawl,
    onSubmitResearch,
  };
}

function setViewportHeight(height: number) {
  Object.defineProperty(window, "innerHeight", {
    configurable: true,
    writable: true,
    value: height,
  });
  window.dispatchEvent(new Event("resize"));
}

describe("JobSubmissionContainer wizard", () => {
  beforeEach(() => {
    installMockLocalStorage();
    window.localStorage.clear();
    setViewportHeight(DEFAULT_VIEWPORT_HEIGHT);
  });

  it("blocks advancement when basics are invalid", async () => {
    const user = userEvent.setup();
    renderHarness();

    await user.click(screen.getByRole("button", { name: /next/i }));

    expect(
      screen.getByText(/a target url is required before continuing/i),
    ).toBeInTheDocument();
  });

  it("preserves entered data when toggling into expert mode", async () => {
    const user = userEvent.setup();
    renderHarness();

    await user.type(
      screen.getAllByLabelText(/target url/i)[0],
      "https://example.com",
    );

    await user.click(screen.getByRole("checkbox", { name: /guided mode/i }));

    expect(
      (await screen.findAllByDisplayValue("https://example.com"))[0],
    ).toBeInTheDocument();
  });

  it("autosaves draft state to localStorage", async () => {
    const user = userEvent.setup();
    renderHarness();

    await user.type(
      screen.getAllByLabelText(/target url/i)[0],
      "https://example.com",
    );

    await waitFor(() => {
      expect(
        JSON.parse(
          window.localStorage.getItem("spartan.job-draft.scrape") ?? "{}",
        ),
      ).toMatchObject({
        url: "https://example.com",
      });
    });
  });

  it("restores the saved scrape draft when the wizard remounts", async () => {
    const view = renderHarness();
    view.unmount();

    window.localStorage.setItem(
      "spartan.job-draft.scrape",
      JSON.stringify({ url: "https://example.com/docs" }),
    );

    renderHarness();

    await waitFor(() => {
      expect(document.getElementById("wizard-scrape-url")).toHaveValue(
        "https://example.com/docs",
      );
    });
  });

  it("renders stable onboarding targets for the new-job tour", () => {
    const { container } = renderHarness();

    expect(
      container.querySelector('[data-tour="job-wizard-header"]'),
    ).not.toBeNull();
    expect(
      container.querySelector('[data-tour="wizard-steps"]'),
    ).not.toBeNull();
  });

  it("marks the workspace compact on short viewports", () => {
    setViewportHeight(780);
    const { container } = renderHarness("scrape", { includePresets: true });

    expect(container.querySelector(".job-wizard__workspace")).toHaveAttribute(
      "data-compact",
      "true",
    );
    expect(container.querySelector(".job-wizard__workspace")).toHaveAttribute(
      "data-sidebar-mode",
      "assistant",
    );
    expect(
      screen.getByRole("heading", { name: /scrape presets/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: /job submission assistant/i }),
    ).toBeInTheDocument();
  });

  it("keeps presets visible when the assistant is hidden", () => {
    window.localStorage.setItem("spartan.ai-assistant.open", "false");
    const { container } = renderHarness("scrape", { includePresets: true });

    expect(container.querySelector(".job-wizard__workspace")).toHaveAttribute(
      "data-sidebar-mode",
      "presets",
    );
    expect(
      screen.getByRole("heading", { name: /scrape presets/i }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("heading", { name: /job submission assistant/i }),
    ).not.toBeInTheDocument();
    expect(container.querySelector(".job-wizard__sidebar")).not.toBeNull();
  });

  it("drops the sidebar when both presets and the assistant are unavailable", () => {
    window.localStorage.setItem("spartan.ai-assistant.open", "false");
    const { container } = renderHarness();

    expect(container.querySelector(".job-wizard__workspace")).toHaveAttribute(
      "data-sidebar-mode",
      "none",
    );
    expect(container.querySelector(".job-wizard__sidebar")).toBeNull();
  });

  it("keeps the workspace in standard mode on taller viewports", () => {
    const { container } = renderHarness("scrape", { includePresets: true });

    expect(container.querySelector(".job-wizard__workspace")).toHaveAttribute(
      "data-compact",
      "false",
    );
  });
});
