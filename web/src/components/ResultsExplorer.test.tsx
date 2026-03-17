/**
 * Purpose: Verify the result explorer cutover keeps one dominant reader while demoting secondary tools and exports.
 * Responsibilities: Assert first-paint reader priority, explicit secondary-tool activation, and guided export behavior.
 * Scope: Component coverage for `ResultsExplorer`.
 * Usage: Run with Vitest.
 * Invariants/Assumptions: The primary reader always renders, secondary tools stay hidden until explicitly opened, and export uses the saved-job endpoint.
 */

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { Job, ResultItem } from "../types";
import { AIAssistantProvider } from "./ai-assistant";
import { ResultsExplorer } from "./ResultsExplorer";

const exportResultsMock = vi.fn();
const loadResultsMock = vi.fn();

vi.mock("../lib/results", () => ({
  exportResults: (...args: unknown[]) => exportResultsMock(...args),
  loadResults: (...args: unknown[]) => loadResultsMock(...args),
}));

vi.mock("./ResultsViewer", () => ({
  ResultsViewer: ({ resultItems }: { resultItems: unknown[] }) => (
    <div data-testid="results-viewer">
      {resultItems.length} visible result(s)
    </div>
  ),
}));

vi.mock("./TreeView", () => ({
  TreeView: () => <div data-testid="tree-view">tree view</div>,
}));

vi.mock("./DiffViewer", () => ({
  DiffViewer: () => <div data-testid="diff-viewer">diff viewer</div>,
}));

vi.mock("./EvidenceChart", () => ({
  EvidenceChart: () => <div data-testid="evidence-chart">evidence chart</div>,
}));

vi.mock("./ClusterGraph", () => ({
  ClusterGraph: () => <div data-testid="cluster-graph">cluster graph</div>,
}));

vi.mock("./TransformPreview", () => ({
  TransformPreview: ({
    onApply,
  }: {
    onApply?: (
      format: "jsonl" | "json" | "md" | "csv" | "xlsx",
      expression: string,
      language: "jmespath" | "jsonata",
    ) => void;
  }) => (
    <div data-testid="transform-preview">
      <div>transform preview</div>
      <button
        type="button"
        onClick={() => onApply?.("json", "[].url", "jmespath")}
      >
        Apply transform export
      </button>
    </div>
  ),
}));

const jobs: Job[] = [
  {
    id: "job-1",
    status: "succeeded",
    kind: "crawl",
    createdAt: "2026-03-16T12:00:00.000Z",
    updatedAt: "2026-03-16T12:05:00.000Z",
    specVersion: 1,
    spec: {},
    run: { waitMs: 0, runMs: 1000, totalMs: 1000 },
  },
  {
    id: "job-2",
    status: "succeeded",
    kind: "crawl",
    createdAt: "2026-03-16T12:10:00.000Z",
    updatedAt: "2026-03-16T12:15:00.000Z",
    specVersion: 1,
    spec: {},
    run: { waitMs: 0, runMs: 1000, totalMs: 1000 },
  },
];

const items: ResultItem[] = [
  {
    url: "https://example.com/a",
    status: 200,
    title: "Article A",
    text: "alpha page",
    links: [],
    normalized: { title: "Article A" },
  },
  {
    url: "https://example.com/b",
    status: 404,
    title: "Article B",
    text: "beta page",
    links: [],
  },
];

function renderExplorer() {
  return render(
    <AIAssistantProvider>
      <ResultsExplorer
        jobId="job-1"
        resultItems={items}
        selectedResultIndex={0}
        setSelectedResultIndex={vi.fn()}
        resultSummary={null}
        resultConfidence={null}
        resultEvidence={[]}
        resultClusters={[]}
        resultCitations={[]}
        resultAgentic={null}
        rawResult={JSON.stringify(items)}
        resultFormat="jsonl"
        currentPage={1}
        totalResults={12}
        resultsPerPage={100}
        onLoadPage={vi.fn()}
        availableJobs={jobs}
        jobType="crawl"
      />
    </AIAssistantProvider>,
  );
}

describe("ResultsExplorer", () => {
  beforeEach(() => {
    exportResultsMock.mockReset();
    loadResultsMock.mockReset();

    exportResultsMock.mockResolvedValue({
      content: "{}",
      filename: "job-1.json",
      contentType: "application/json",
      isBinary: false,
    });

    loadResultsMock.mockResolvedValue({
      data: [],
      raw: "[]",
    });

    vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:mock");
    vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
    vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});
  });

  it("keeps the primary reader dominant on first paint", () => {
    renderExplorer();

    expect(screen.getByTestId("results-viewer")).toHaveTextContent(
      "2 visible result(s)",
    );
    expect(screen.getByRole("button", { name: "Tools" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Export" })).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Compare runs/i }),
    ).not.toBeInTheDocument();
  });

  it("opens secondary tools from an explicit drawer", async () => {
    const user = userEvent.setup();
    renderExplorer();

    await user.click(screen.getByRole("button", { name: "Tools" }));
    await user.click(screen.getByRole("button", { name: /Compare runs/i }));

    expect(
      screen.getByText(/Secondary tool: Compare runs/i),
    ).toBeInTheDocument();
    expect(screen.getByLabelText(/Compare with:/i)).toBeInTheDocument();
  });

  it("uses a guided export flow with scope guidance", async () => {
    const user = userEvent.setup();
    renderExplorer();

    await user.type(
      screen.getByPlaceholderText(/Search by URL, title, or content/i),
      "alpha",
    );
    await user.click(screen.getByRole("button", { name: "Export" }));

    expect(
      screen.getByText(/Exports the full saved job output/i),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        /Current search and status filters only narrow the on-screen reader/i,
      ),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /^Export JSON$/i }));

    expect(exportResultsMock).toHaveBeenCalledWith("job-1", {
      format: "json",
    });
  });

  it("surfaces transform export failures without leaking object coercions", async () => {
    const user = userEvent.setup();
    exportResultsMock.mockRejectedValueOnce({
      message: "Transform export failed cleanly.",
    });
    renderExplorer();

    await user.click(screen.getByRole("button", { name: "Tools" }));
    await user.click(screen.getByRole("button", { name: /Transform output/i }));
    await user.click(
      screen.getByRole("button", { name: /Apply transform export/i }),
    );

    expect(
      await screen.findByText(/Transform export failed cleanly\./i),
    ).toBeInTheDocument();
    expect(screen.queryByText("[object Object]")).not.toBeInTheDocument();
  });
});
