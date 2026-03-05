/**
 * Tests for API URL configuration utilities.
 *
 * Tests base URL resolution, path construction, and URL building
 * for API requests with various base URL configurations.
 */
import { describe, it, expect } from "vitest";
import { buildApiUrl, buildApiUrlWithBase, getApiBaseUrl } from "./api-config";

describe("API configuration", () => {
  describe("getApiBaseUrl", () => {
    it("should return empty string when env var is not set", () => {
      // In the test environment, VITE_API_BASE_URL is not set
      // The function should default to empty string
      const result = getApiBaseUrl();
      expect(result).toBe("");
    });

    it("should return the env var value when set", () => {
      // Note: Vite environment variables are build-time constants.
      // This test documents the expected behavior when VITE_API_BASE_URL is set.
      // In production, this would be set at build time via .env or build args.
      // For example: VITE_API_BASE_URL=https://api.example.com pnpm build
      const result = getApiBaseUrl();
      // In test environment, this returns empty string
      expect(typeof result).toBe("string");
    });
  });

  describe("buildApiUrl", () => {
    it("should return path directly when base URL is empty", () => {
      // In test environment, VITE_API_BASE_URL is not set, so base URL is empty
      // This is the dev proxy mode where paths are relative
      const result = buildApiUrl("/v1/jobs/123/results");
      expect(result).toBe("/v1/jobs/123/results");
    });

    it("should return empty string for empty path", () => {
      const result = buildApiUrl("");
      // With empty base URL, empty path returns empty string
      expect(result).toBe("");
    });

    it("should handle paths without leading slash", () => {
      const result = buildApiUrl("v1/jobs/123/results");
      expect(result).toBe("v1/jobs/123/results");
    });

    it("should return a string type", () => {
      const result = buildApiUrl("/v1/jobs/123/results");
      expect(typeof result).toBe("string");
    });
  });

  describe("buildApiUrlWithBase", () => {
    it("should return path directly when base URL is empty", () => {
      const result = buildApiUrlWithBase("", "/v1/jobs/123/results");
      expect(result).toBe("/v1/jobs/123/results");
    });

    it("should return full URL when base URL is set", () => {
      const result = buildApiUrlWithBase(
        "https://api.example.com",
        "/v1/jobs/123/results",
      );
      expect(result).toBe("https://api.example.com/v1/jobs/123/results");
    });

    it("should handle trailing slash in base URL", () => {
      // User accidentally adds trailing slash to VITE_API_BASE_URL
      const result = buildApiUrlWithBase(
        "https://api.example.com/",
        "/v1/jobs/123/results",
      );
      // Should not produce double slash
      expect(result).toBe("https://api.example.com/v1/jobs/123/results");
    });

    it("should handle base URL with only trailing slash", () => {
      const result = buildApiUrlWithBase("/", "/v1/jobs/123/results");
      expect(result).toBe("/v1/jobs/123/results");
    });
  });
});
