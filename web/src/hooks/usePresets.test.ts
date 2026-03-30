/**
 * Purpose: Verify custom preset hydration and persistence for the web preset surface.
 * Responsibilities: Assert stored custom presets load immediately and newly saved presets round-trip through localStorage.
 * Scope: Hook-level preset state and storage behavior only.
 * Usage: Run via `pnpm run test` or `make test-ci`.
 * Invariants/Assumptions: Stored preset payloads may be missing or invalid, and custom preset CRUD should fail open.
 */

import { renderHook, act } from "@testing-library/react";
import {
  afterAll,
  afterEach,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from "vitest";
import { usePresets } from "./usePresets";
import type { JobPreset } from "../types/presets";

const storage = new Map<string, string>();

function makePreset(overrides: Partial<JobPreset> = {}): JobPreset {
  return {
    id: "custom-1",
    name: "Custom Preset",
    description: "Stored preset",
    icon: "⚙️",
    jobType: "scrape",
    config: {
      url: "https://example.com",
      headless: true,
    },
    resources: {
      timeSeconds: 30,
      cpu: "low",
      memory: "low",
    },
    useCases: ["Example"],
    isBuiltIn: false,
    createdAt: 1741178400000,
    ...overrides,
  };
}

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
  if (typeof window !== "undefined") {
    Object.defineProperty(window, "localStorage", {
      value: localStorageMock,
      configurable: true,
    });
  }
});

afterAll(() => {
  vi.unstubAllGlobals();
});

describe("usePresets", () => {
  beforeEach(() => {
    storage.clear();
  });

  afterEach(() => {
    storage.clear();
  });

  it("loads stored custom presets during initial render", () => {
    const preset = makePreset();
    localStorage.setItem("spartan-job-presets", JSON.stringify([preset]));

    const { result } = renderHook(() => usePresets());

    expect(result.current.customPresets).toEqual([preset]);
    expect(result.current.getPresetById("custom-1")).toEqual(preset);
  });

  it("persists newly saved custom presets", () => {
    const { result } = renderHook(() => usePresets());

    act(() => {
      result.current.savePreset(
        "  Saved Preset  ",
        "  Saved description  ",
        "scrape",
        {
          url: "https://example.org",
          headless: true,
        },
      );
    });

    const stored = JSON.parse(
      localStorage.getItem("spartan-job-presets") ?? "[]",
    ) as JobPreset[];

    expect(stored).toHaveLength(1);
    expect(stored[0]).toMatchObject({
      name: "Saved Preset",
      description: "Saved description",
      jobType: "scrape",
      icon: "⚙️",
      isBuiltIn: false,
      useCases: ["Custom preset"],
    });
    expect(stored[0].resources.timeSeconds).toBeGreaterThan(0);
  });
});
