/**
 * Tests for authentication parsing utilities.
 *
 * Tests cookie parsing, query parameter parsing, and auth payload building
 * for scrape, crawl, and research request forms.
 */
import { describe, it, expect } from "vitest";
import { parseCookies, parseQueryParams, buildAuth } from "./lib/form-utils";

// Test parseCookies function
describe("parseCookies", () => {
  it("should return undefined for empty input", () => {
    expect(parseCookies("")).toBeUndefined();
    expect(parseCookies("   \n  ")).toBeUndefined();
  });

  it("should parse single cookie", () => {
    expect(parseCookies("session_id=abc123")).toEqual(["session_id=abc123"]);
  });

  it("should parse multiple cookies", () => {
    const input = "session_id=abc123\nauth_token=xyz789";
    expect(parseCookies(input)).toEqual([
      "session_id=abc123",
      "auth_token=xyz789",
    ]);
  });

  it("should trim whitespace", () => {
    const input = "  session_id=abc123  \n  auth_token=xyz789  ";
    expect(parseCookies(input)).toEqual([
      "session_id=abc123",
      "auth_token=xyz789",
    ]);
  });

  it("should filter empty lines", () => {
    const input = "session_id=abc123\n\n\nauth_token=xyz789\n";
    expect(parseCookies(input)).toEqual([
      "session_id=abc123",
      "auth_token=xyz789",
    ]);
  });

  it("should return undefined if all lines are empty after filtering", () => {
    expect(parseCookies("\n\n\n")).toBeUndefined();
  });
});

// Test parseQueryParams function
describe("parseQueryParams", () => {
  it("should return undefined for empty input", () => {
    expect(parseQueryParams("")).toBeUndefined();
    expect(parseQueryParams("   \n  ")).toBeUndefined();
  });

  it("should parse single query param", () => {
    expect(parseQueryParams("api_key=abc123")).toEqual({ api_key: "abc123" });
  });

  it("should parse multiple query params", () => {
    const input = "api_key=abc123\nversion=v1";
    expect(parseQueryParams(input)).toEqual({
      api_key: "abc123",
      version: "v1",
    });
  });

  it("should trim whitespace from keys and values", () => {
    const input = "  api_key  =  abc123  \n  version  =  v1  ";
    expect(parseQueryParams(input)).toEqual({
      api_key: "abc123",
      version: "v1",
    });
  });

  it("should filter empty lines", () => {
    const input = "api_key=abc123\n\n\nversion=v1\n";
    expect(parseQueryParams(input)).toEqual({
      api_key: "abc123",
      version: "v1",
    });
  });

  it("should skip lines without equals sign", () => {
    const input = "api_key=abc123\ninvalid_line\nversion=v1";
    expect(parseQueryParams(input)).toEqual({
      api_key: "abc123",
      version: "v1",
    });
  });

  it("should skip lines with empty key or value", () => {
    const input = "api_key=abc123\n=value\nversion=v1\nkey=";
    expect(parseQueryParams(input)).toEqual({
      api_key: "abc123",
      version: "v1",
    });
  });

  it("should handle values with equals sign", () => {
    expect(parseQueryParams("token=abc=123")).toEqual({ token: "abc=123" });
  });
});

// Test buildAuth serialization
describe("buildAuth serialization", () => {
  it("should return undefined when all fields are empty", () => {
    expect(buildAuth("", undefined, undefined, undefined)).toBeUndefined();
  });

  it("should include all non-empty fields", () => {
    const result = buildAuth(
      "user:pass",
      { "X-Custom": "header" },
      ["cookie1=val1"],
      { key: "value" },
    );
    expect(result).toEqual({
      basic: "user:pass",
      headers: { "X-Custom": "header" },
      cookies: ["cookie1=val1"],
      query: { key: "value" },
    });
  });

  it("should serialize to JSON correctly", () => {
    const result = buildAuth(
      "user:pass",
      { "X-Token": "xyz" },
      ["session=abc", "auth=xyz"],
      { api_key: "123" },
    );
    const json = JSON.stringify(result);
    const parsed = JSON.parse(json);

    expect(parsed).toEqual({
      basic: "user:pass",
      headers: { "X-Token": "xyz" },
      cookies: ["session=abc", "auth=xyz"],
      query: { api_key: "123" },
    });
  });
});
