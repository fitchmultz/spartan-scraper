import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { TransformPreview } from "./TransformPreview";

describe("TransformPreview", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.spyOn(globalThis, "fetch").mockImplementation(
      async (input: RequestInfo | URL, init?: RequestInit) => {
        const url =
          typeof input === "string"
            ? input
            : input instanceof URL
              ? input.toString()
              : input.url;
        if (url === "/v1/ai/transform-generate") {
          return new Response(
            JSON.stringify({
              issues: [
                "current transform is invalid: invalid transform.expression",
              ],
              inputStats: {
                sampleRecordCount: 2,
                fieldPathCount: 5,
                currentTransformProvided: true,
              },
              transform: {
                expression: "{title: title, url: url}",
                language: "jmespath",
              },
              preview: [
                {
                  title: "Example",
                  url: "https://example.com",
                },
              ],
              explanation: "Projected the title and URL fields for export.",
              route_id: "openai/gpt-5.4",
              provider: "openai",
              model: "gpt-5.4",
            }),
            {
              status: 200,
              headers: { "Content-Type": "application/json" },
            },
          );
        }
        if (url === "/v1/transform/validate") {
          return new Response(
            JSON.stringify({ valid: true, message: "Expression is valid" }),
            {
              status: 200,
              headers: { "Content-Type": "application/json" },
            },
          );
        }
        if (url === "/v1/jobs/job-123/preview-transform") {
          return new Response(
            JSON.stringify({
              results: [{ title: "Example", url: "https://example.com" }],
              resultCount: 1,
            }),
            {
              status: 200,
              headers: { "Content-Type": "application/json" },
            },
          );
        }
        throw new Error(
          `unexpected fetch: ${url} ${(init?.method || "GET") as string}`,
        );
      },
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("generates a bounded transform with AI and applies it to the editor", async () => {
    render(<TransformPreview jobId="job-123" onApply={vi.fn()} />);

    fireEvent.click(screen.getByRole("button", { name: /generate with ai/i }));
    fireEvent.change(screen.getByLabelText(/ai transform instructions/i), {
      target: { value: "Project the URL and title for export." },
    });
    fireEvent.click(
      screen.getByRole("button", { name: /generate transform/i }),
    );

    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalledWith(
        "/v1/ai/transform-generate",
        expect.objectContaining({
          method: "POST",
        }),
      );
    });

    expect(
      await screen.findByDisplayValue("{title: title, url: url}"),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/projected the title and url fields for export/i),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/current transform is invalid/i),
    ).toBeInTheDocument();
    expect(screen.getByText(/sample records 2/i)).toBeInTheDocument();
    expect(screen.getByText(/field paths 5/i)).toBeInTheDocument();
    expect(screen.getByText(/https:\/\/example.com/i)).toBeInTheDocument();
  });
});
