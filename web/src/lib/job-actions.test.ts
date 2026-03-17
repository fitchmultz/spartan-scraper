/**
 * Tests for job action functions.
 *
 * Tests job submission and management logic in isolation.
 */
import { describe, it, expect, vi } from "vitest";
import type { ScrapeRequest, CrawlRequest, ResearchRequest } from "../api";
import {
  submitScrapeJob,
  submitCrawlJob,
  submitResearchJob,
  cancelJob,
  deleteJob,
} from "./job-actions";

const mockPostV1Scrape = vi.fn();
const mockPostV1Crawl = vi.fn();
const mockPostV1Research = vi.fn();
const mockDeleteV1JobsById = vi.fn();

describe("submitScrapeJob", () => {
  it("should submit scrape job successfully", async () => {
    mockPostV1Scrape.mockResolvedValue({ error: undefined });
    const mockRefreshJobs = vi.fn().mockResolvedValue(undefined);
    const mockSetLoading = vi.fn();
    const mockSetError = vi.fn();
    const getApiBaseUrl = vi.fn(() => "http://localhost:8080");

    const request: ScrapeRequest = {
      url: "https://example.com",
      headless: true,
      timeoutSeconds: 30,
    };

    await submitScrapeJob(mockPostV1Scrape, {
      request,
      setLoading: mockSetLoading,
      setError: mockSetError,
      refreshJobs: mockRefreshJobs,
      getApiBaseUrl,
    });

    expect(mockSetLoading).toHaveBeenCalledWith(true);
    expect(mockPostV1Scrape).toHaveBeenCalledWith({
      baseUrl: "http://localhost:8080",
      body: request,
    });
    expect(mockSetError).toHaveBeenCalledWith(null);
    expect(mockRefreshJobs).toHaveBeenCalled();
    expect(mockSetLoading).toHaveBeenCalledWith(false);
  });

  it("should handle API error", async () => {
    mockPostV1Scrape.mockResolvedValue({ error: "API error" });
    const mockRefreshJobs = vi.fn().mockResolvedValue(undefined);
    const mockSetLoading = vi.fn();
    const mockSetError = vi.fn();
    const getApiBaseUrl = vi.fn(() => "http://localhost:8080");

    const request: ScrapeRequest = {
      url: "https://example.com",
      headless: true,
      timeoutSeconds: 30,
    };

    await submitScrapeJob(mockPostV1Scrape, {
      request,
      setLoading: mockSetLoading,
      setError: mockSetError,
      refreshJobs: mockRefreshJobs,
      getApiBaseUrl,
    });

    expect(mockSetError).toHaveBeenCalledWith("API error");
    expect(mockRefreshJobs).not.toHaveBeenCalled();
    expect(mockSetLoading).toHaveBeenCalledWith(false);
  });

  it("should handle network error", async () => {
    mockPostV1Scrape.mockRejectedValue(new Error("Network error"));
    const mockRefreshJobs = vi.fn().mockResolvedValue(undefined);
    const mockSetLoading = vi.fn();
    const mockSetError = vi.fn();
    const getApiBaseUrl = vi.fn(() => "http://localhost:8080");

    const request: ScrapeRequest = {
      url: "https://example.com",
      headless: true,
      timeoutSeconds: 30,
    };

    await submitScrapeJob(mockPostV1Scrape, {
      request,
      setLoading: mockSetLoading,
      setError: mockSetError,
      refreshJobs: mockRefreshJobs,
      getApiBaseUrl,
    });

    expect(mockSetError).toHaveBeenCalledWith("Network error");
    expect(mockSetLoading).toHaveBeenCalledWith(false);
  });
});

describe("submitCrawlJob", () => {
  it("should submit crawl job successfully", async () => {
    mockPostV1Crawl.mockResolvedValue({ error: undefined });
    const mockRefreshJobs = vi.fn().mockResolvedValue(undefined);
    const mockSetLoading = vi.fn();
    const mockSetError = vi.fn();
    const getApiBaseUrl = vi.fn(() => "http://localhost:8080");

    const request: CrawlRequest = {
      url: "https://example.com",
      maxDepth: 2,
      maxPages: 100,
      headless: true,
      timeoutSeconds: 30,
    };

    await submitCrawlJob(mockPostV1Crawl, {
      request,
      setLoading: mockSetLoading,
      setError: mockSetError,
      refreshJobs: mockRefreshJobs,
      getApiBaseUrl,
    });

    expect(mockPostV1Crawl).toHaveBeenCalledWith({
      baseUrl: "http://localhost:8080",
      body: request,
    });
    expect(mockSetError).toHaveBeenCalledWith(null);
    expect(mockRefreshJobs).toHaveBeenCalled();
  });

  it("should handle API error", async () => {
    mockPostV1Crawl.mockResolvedValue({ error: "Invalid request" });
    const mockRefreshJobs = vi.fn().mockResolvedValue(undefined);
    const mockSetLoading = vi.fn();
    const mockSetError = vi.fn();
    const getApiBaseUrl = vi.fn(() => "http://localhost:8080");

    const request: CrawlRequest = {
      url: "https://example.com",
      maxDepth: 2,
      maxPages: 100,
      headless: true,
      timeoutSeconds: 30,
    };

    await submitCrawlJob(mockPostV1Crawl, {
      request,
      setLoading: mockSetLoading,
      setError: mockSetError,
      refreshJobs: mockRefreshJobs,
      getApiBaseUrl,
    });

    expect(mockSetError).toHaveBeenCalledWith("Invalid request");
    expect(mockRefreshJobs).not.toHaveBeenCalled();
  });
});

describe("submitResearchJob", () => {
  it("should submit research job successfully", async () => {
    mockPostV1Research.mockResolvedValue({ error: undefined });
    const mockRefreshJobs = vi.fn().mockResolvedValue(undefined);
    const mockSetLoading = vi.fn();
    const mockSetError = vi.fn();
    const getApiBaseUrl = vi.fn(() => "http://localhost:8080");

    const request: ResearchRequest = {
      query: "test query",
      urls: ["https://example.com"],
      maxDepth: 2,
      maxPages: 100,
      headless: true,
      timeoutSeconds: 30,
    };

    await submitResearchJob(mockPostV1Research, {
      request,
      setLoading: mockSetLoading,
      setError: mockSetError,
      refreshJobs: mockRefreshJobs,
      getApiBaseUrl,
    });

    expect(mockPostV1Research).toHaveBeenCalledWith({
      baseUrl: "http://localhost:8080",
      body: request,
    });
    expect(mockSetError).toHaveBeenCalledWith(null);
    expect(mockRefreshJobs).toHaveBeenCalled();
  });

  it("should handle API error", async () => {
    mockPostV1Research.mockResolvedValue({ error: "Rate limit exceeded" });
    const mockRefreshJobs = vi.fn().mockResolvedValue(undefined);
    const mockSetLoading = vi.fn();
    const mockSetError = vi.fn();
    const getApiBaseUrl = vi.fn(() => "http://localhost:8080");

    const request: ResearchRequest = {
      query: "test query",
      urls: ["https://example.com"],
      maxDepth: 2,
      maxPages: 100,
      headless: true,
      timeoutSeconds: 30,
    };

    await submitResearchJob(mockPostV1Research, {
      request,
      setLoading: mockSetLoading,
      setError: mockSetError,
      refreshJobs: mockRefreshJobs,
      getApiBaseUrl,
    });

    expect(mockSetError).toHaveBeenCalledWith("Rate limit exceeded");
    expect(mockRefreshJobs).not.toHaveBeenCalled();
  });
});

describe("cancelJob", () => {
  it("should cancel job successfully", async () => {
    mockDeleteV1JobsById.mockResolvedValue({ error: undefined });
    const mockRefreshJobs = vi.fn().mockResolvedValue(undefined);
    const mockSetLoading = vi.fn();
    const mockSetError = vi.fn();
    const getApiBaseUrl = vi.fn(() => "http://localhost:8080");

    await cancelJob(
      mockDeleteV1JobsById,
      "job-123",
      mockSetLoading,
      mockSetError,
      mockRefreshJobs,
      getApiBaseUrl,
    );

    expect(mockDeleteV1JobsById).toHaveBeenCalledWith({
      baseUrl: "http://localhost:8080",
      path: { id: "job-123" },
    });
    expect(mockSetError).toHaveBeenCalledWith(null);
    expect(mockRefreshJobs).toHaveBeenCalled();
  });

  it("should handle API error", async () => {
    mockDeleteV1JobsById.mockResolvedValue({ error: "Job not found" });
    const mockRefreshJobs = vi.fn().mockResolvedValue(undefined);
    const mockSetLoading = vi.fn();
    const mockSetError = vi.fn();
    const getApiBaseUrl = vi.fn(() => "http://localhost:8080");

    await cancelJob(
      mockDeleteV1JobsById,
      "job-123",
      mockSetLoading,
      mockSetError,
      mockRefreshJobs,
      getApiBaseUrl,
    );

    expect(mockSetError).toHaveBeenCalledWith("Job not found");
    expect(mockRefreshJobs).not.toHaveBeenCalled();
  });

  it("should handle network error", async () => {
    mockDeleteV1JobsById.mockRejectedValue(new Error("Network error"));
    const mockRefreshJobs = vi.fn().mockResolvedValue(undefined);
    const mockSetLoading = vi.fn();
    const mockSetError = vi.fn();
    const getApiBaseUrl = vi.fn(() => "http://localhost:8080");

    await cancelJob(
      mockDeleteV1JobsById,
      "job-123",
      mockSetLoading,
      mockSetError,
      mockRefreshJobs,
      getApiBaseUrl,
    );

    expect(mockSetError).toHaveBeenCalledWith("Network error");
    expect(mockSetLoading).toHaveBeenCalledWith(false);
  });
});

describe("deleteJob", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("should delete job successfully when confirmed", async () => {
    const confirmDelete = vi.fn(() => true);

    mockDeleteV1JobsById.mockResolvedValue({ error: undefined });
    const mockRefreshJobs = vi.fn().mockResolvedValue(undefined);
    const mockSetLoading = vi.fn();
    const mockSetError = vi.fn();
    const mockOnJobDeleted = vi.fn();
    const getApiBaseUrl = vi.fn(() => "http://localhost:8080");

    const result = await deleteJob(
      mockDeleteV1JobsById,
      "job-123",
      mockSetLoading,
      mockSetError,
      mockRefreshJobs,
      getApiBaseUrl,
      confirmDelete,
      "job-123",
      mockOnJobDeleted,
    );

    expect(confirmDelete).toHaveBeenCalled();
    expect(result).toEqual({ status: "success" });
    expect(mockDeleteV1JobsById).toHaveBeenCalledWith({
      baseUrl: "http://localhost:8080",
      path: { id: "job-123" },
      query: { force: true },
    });
    expect(mockSetError).toHaveBeenCalledWith(null);
    expect(mockRefreshJobs).toHaveBeenCalled();
    expect(mockOnJobDeleted).toHaveBeenCalledWith("job-123");
  });

  it("should not delete job when not confirmed", async () => {
    const confirmDelete = vi.fn(() => false);

    mockDeleteV1JobsById.mockResolvedValue({ error: undefined });
    const mockRefreshJobs = vi.fn().mockResolvedValue(undefined);
    const mockSetLoading = vi.fn();
    const mockSetError = vi.fn();
    const mockOnJobDeleted = vi.fn();
    const getApiBaseUrl = vi.fn(() => "http://localhost:8080");

    const result = await deleteJob(
      mockDeleteV1JobsById,
      "job-123",
      mockSetLoading,
      mockSetError,
      mockRefreshJobs,
      getApiBaseUrl,
      confirmDelete,
      "job-123",
      mockOnJobDeleted,
    );

    expect(confirmDelete).toHaveBeenCalled();
    expect(result).toEqual({ status: "canceled" });
    expect(mockDeleteV1JobsById).not.toHaveBeenCalled();
    expect(mockRefreshJobs).not.toHaveBeenCalled();
    expect(mockOnJobDeleted).not.toHaveBeenCalled();
  });

  it("should handle API error", async () => {
    const confirmDelete = vi.fn(() => true);

    mockDeleteV1JobsById.mockResolvedValue({ error: "Delete failed" });
    const mockRefreshJobs = vi.fn().mockResolvedValue(undefined);
    const mockSetLoading = vi.fn();
    const mockSetError = vi.fn();
    const mockOnJobDeleted = vi.fn();
    const getApiBaseUrl = vi.fn(() => "http://localhost:8080");

    const result = await deleteJob(
      mockDeleteV1JobsById,
      "job-123",
      mockSetLoading,
      mockSetError,
      mockRefreshJobs,
      getApiBaseUrl,
      confirmDelete,
      "job-123",
      mockOnJobDeleted,
    );

    expect(result).toEqual({ status: "error", message: "Delete failed" });
    expect(mockSetError).toHaveBeenCalledWith("Delete failed");
    expect(mockRefreshJobs).not.toHaveBeenCalled();
    expect(mockOnJobDeleted).not.toHaveBeenCalled();
  });

  it("should call onJobDeleted when deleting selected job", async () => {
    const confirmDelete = vi.fn(() => true);

    mockDeleteV1JobsById.mockResolvedValue({ error: undefined });
    const mockRefreshJobs = vi.fn().mockResolvedValue(undefined);
    const mockSetLoading = vi.fn();
    const mockSetError = vi.fn();
    const mockOnJobDeleted = vi.fn();
    const getApiBaseUrl = vi.fn(() => "http://localhost:8080");

    await deleteJob(
      mockDeleteV1JobsById,
      "job-123",
      mockSetLoading,
      mockSetError,
      mockRefreshJobs,
      getApiBaseUrl,
      confirmDelete,
      "job-123",
      mockOnJobDeleted,
    );

    expect(mockOnJobDeleted).toHaveBeenCalledWith("job-123");
  });

  it("should not call onJobDeleted when deleting different job", async () => {
    const confirmDelete = vi.fn(() => true);

    mockDeleteV1JobsById.mockResolvedValue({ error: undefined });
    const mockRefreshJobs = vi.fn().mockResolvedValue(undefined);
    const mockSetLoading = vi.fn();
    const mockSetError = vi.fn();
    const mockOnJobDeleted = vi.fn();
    const getApiBaseUrl = vi.fn(() => "http://localhost:8080");

    await deleteJob(
      mockDeleteV1JobsById,
      "job-123",
      mockSetLoading,
      mockSetError,
      mockRefreshJobs,
      getApiBaseUrl,
      confirmDelete,
      "job-456",
      mockOnJobDeleted,
    );

    expect(mockOnJobDeleted).not.toHaveBeenCalled();
  });
});
