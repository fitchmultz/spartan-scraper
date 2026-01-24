// Unit tests for loadResults error handling logic.
// These tests verify the error handling behavior by testing the logic in isolation.
// Note: loadResults is internal to the App component, so these tests duplicate the logic
// to verify correct error handling behavior. The actual App.tsx implementation should match
// this logic for the tests to remain valid.
import { describe, it, expect, vi, beforeEach } from "vitest";

const validNDJSON = JSON.stringify({
  url: "https://example.com",
  status: 200,
  title: "Test",
  text: "Content",
  links: [],
});

async function loadResults(jobId: string) {
  try {
    const response = await fetch(`/v1/jobs/${jobId}/results`);

    if (!response.ok) {
      let errorMessage = `Failed to load results (${response.status} ${response.statusText})`;
      try {
        const errorData = (await response.json()) as { error?: string };
        if (errorData.error) {
          errorMessage = errorData.error;
        }
      } catch {
        // If parsing error body fails, use default message
      }
      return { error: errorMessage };
    }

    const text = await response.text();
    const lines = text.split("\n").filter((line) => line.trim());

    const parsedItems: unknown[] = [];
    for (const line of lines) {
      try {
        const parsed = JSON.parse(line);
        parsedItems.push(parsed);
      } catch {
        // Skip malformed JSON lines
      }
    }

    // Check if we had input but failed to parse anything
    if (parsedItems.length === 0 && lines.length > 0) {
      return {
        error: "No valid results found. Results file may be corrupted.",
      };
    }

    return { data: parsedItems, raw: text };
  } catch (err) {
    return { error: String(err) };
  }
}

describe("loadResults error handling", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("should display server error message on 404", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: false,
      status: 404,
      statusText: "Not Found",
      json: async () => ({ error: "job not found" }),
    });
    vi.stubGlobal("fetch", mockFetch);

    const result = await loadResults("test-id");

    expect(result).toEqual({ error: "job not found" });
    expect(mockFetch).toHaveBeenCalledWith("/v1/jobs/test-id/results");
  });

  it("should display server error message on 500", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: false,
      status: 500,
      statusText: "Internal Server Error",
      json: async () => ({ error: "internal server error" }),
    });
    vi.stubGlobal("fetch", mockFetch);

    const result = await loadResults("test-id");

    expect(result).toEqual({ error: "internal server error" });
    expect(mockFetch).toHaveBeenCalledWith("/v1/jobs/test-id/results");
  });

  it("should display server error message on 400", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: false,
      status: 400,
      statusText: "Bad Request",
      json: async () => ({ error: "invalid job id" }),
    });
    vi.stubGlobal("fetch", mockFetch);

    const result = await loadResults("test-id");

    expect(result).toEqual({ error: "invalid job id" });
    expect(mockFetch).toHaveBeenCalledWith("/v1/jobs/test-id/results");
  });

  it("should display default error message when error body cannot be parsed", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: false,
      status: 404,
      statusText: "Not Found",
      json: async () => {
        throw new Error("Invalid JSON");
      },
    });
    vi.stubGlobal("fetch", mockFetch);

    const result = await loadResults("test-id");

    expect(result).toEqual({ error: "Failed to load results (404 Not Found)" });
    expect(mockFetch).toHaveBeenCalledWith("/v1/jobs/test-id/results");
  });

  it("should handle network errors", async () => {
    const mockFetch = vi.fn().mockRejectedValueOnce(new Error("Network error"));
    vi.stubGlobal("fetch", mockFetch);

    const result = await loadResults("test-id");

    expect(result.error).toContain("Network error");
    expect(mockFetch).toHaveBeenCalledWith("/v1/jobs/test-id/results");
  });

  it("should display 'No valid results found' when results file is corrupted", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 200,
      statusText: "OK",
      text: async () => "invalid json\nmore invalid json",
    });
    vi.stubGlobal("fetch", mockFetch);

    const result = await loadResults("test-id");

    expect(result).toEqual({
      error: "No valid results found. Results file may be corrupted.",
    });
    expect(mockFetch).toHaveBeenCalledWith("/v1/jobs/test-id/results");
  });

  it("should parse and display valid results", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 200,
      statusText: "OK",
      text: async () => validNDJSON,
    });
    vi.stubGlobal("fetch", mockFetch);

    const result = await loadResults("test-id");

    expect(result.error).toBeUndefined();
    expect(result.data).toEqual([JSON.parse(validNDJSON)]);
    expect(result.raw).toEqual(validNDJSON);
    expect(mockFetch).toHaveBeenCalledWith("/v1/jobs/test-id/results");
  });

  it("should handle empty results", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 200,
      statusText: "OK",
      text: async () => "",
    });
    vi.stubGlobal("fetch", mockFetch);

    const result = await loadResults("test-id");

    expect(result.error).toBeUndefined();
    expect(result.data).toEqual([]);
    expect(result.raw).toEqual("");
    expect(mockFetch).toHaveBeenCalledWith("/v1/jobs/test-id/results");
  });

  it("should skip malformed JSON lines in results", async () => {
    const mixedNDJSON = `invalid line\n${validNDJSON}\nalso invalid`;
    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 200,
      statusText: "OK",
      text: async () => mixedNDJSON,
    });
    vi.stubGlobal("fetch", mockFetch);

    const result = await loadResults("test-id");

    expect(result.error).toBeUndefined();
    expect(result.data).toEqual([JSON.parse(validNDJSON)]);
    expect(result.raw).toEqual(mixedNDJSON);
    expect(mockFetch).toHaveBeenCalledWith("/v1/jobs/test-id/results");
  });
});
