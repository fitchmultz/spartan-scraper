/**
 * Presets Hook
 *
 * Custom React hook for managing job presets with localStorage persistence.
 * Merges built-in presets with user-saved custom presets, providing CRUD
 * operations for custom preset management.
 *
 * @module hooks/usePresets
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { BUILTIN_PRESETS } from "../data/builtin-presets";
import type { JobPreset, JobType, PresetConfig } from "../types/presets";

const STORAGE_KEY = "spartan-job-presets";

/**
 * Validates that a loaded object conforms to the JobPreset interface.
 *
 * @param obj - The object to validate
 * @returns True if the object is a valid JobPreset
 */
function isValidPreset(obj: unknown): obj is JobPreset {
  if (typeof obj !== "object" || obj === null) {
    return false;
  }

  const preset = obj as Record<string, unknown>;

  // Required fields
  if (typeof preset.id !== "string") return false;
  if (typeof preset.name !== "string") return false;
  if (typeof preset.description !== "string") return false;
  if (typeof preset.icon !== "string") return false;
  if (!["scrape", "crawl", "research"].includes(preset.jobType as string)) {
    return false;
  }
  if (typeof preset.config !== "object" || preset.config === null) {
    return false;
  }
  if (typeof preset.resources !== "object" || preset.resources === null) {
    return false;
  }

  // Validate resources
  const resources = preset.resources as Record<string, unknown>;
  if (typeof resources.timeSeconds !== "number") return false;
  if (!["low", "medium", "high"].includes(resources.cpu as string)) {
    return false;
  }
  if (!["low", "medium", "high"].includes(resources.memory as string)) {
    return false;
  }

  // Validate useCases array
  if (!Array.isArray(preset.useCases)) return false;
  if (!preset.useCases.every((uc) => typeof uc === "string")) return false;

  return true;
}

/**
 * Hook return type definition
 */
export interface UsePresetsReturn {
  /** All presets (built-in + custom) */
  presets: JobPreset[];
  /** Only custom presets */
  customPresets: JobPreset[];
  /** Save current form state as a new preset */
  savePreset: (
    name: string,
    description: string,
    jobType: JobType,
    config: PresetConfig,
  ) => void;
  /** Delete a custom preset */
  deletePreset: (id: string) => void;
  /** Find preset by ID */
  getPresetById: (id: string) => JobPreset | undefined;
  /** Find presets matching a URL pattern */
  findPresetsForUrl: (url: string) => JobPreset[];
}

/**
 * Hook for managing custom job presets with localStorage persistence.
 *
 * Provides CRUD operations for user-saved presets, merged with built-ins.
 *
 * @returns Object containing presets and preset management functions
 */
export function usePresets(): UsePresetsReturn {
  const [customPresets, setCustomPresets] = useState<JobPreset[]>([]);
  const [isLoaded, setIsLoaded] = useState(false);

  // Load custom presets from localStorage on mount
  useEffect(() => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored) {
        const parsed = JSON.parse(stored) as unknown[];
        const valid = parsed.filter(isValidPreset) as JobPreset[];
        setCustomPresets(valid);
      }
    } catch {
      // Ignore localStorage errors (e.g., quota exceeded, disabled)
    }
    setIsLoaded(true);
  }, []);

  // Save custom presets to localStorage when they change
  useEffect(() => {
    if (!isLoaded) return;

    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(customPresets));
    } catch {
      // Ignore localStorage errors
    }
  }, [customPresets, isLoaded]);

  // All presets (built-in + custom)
  const presets = useMemo<JobPreset[]>(
    () => [...BUILTIN_PRESETS, ...customPresets],
    [customPresets],
  );

  /**
   * Save current form state as a new custom preset.
   */
  const savePreset = useCallback(
    (
      name: string,
      description: string,
      jobType: JobType,
      config: PresetConfig,
    ) => {
      const newPreset: JobPreset = {
        id: `custom-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
        name: name.trim(),
        description: description.trim(),
        icon: "⚙️",
        jobType,
        config,
        resources: {
          timeSeconds: estimateTime(config),
          cpu: estimateCpu(config),
          memory: estimateMemory(config),
        },
        useCases: ["Custom preset"],
        isBuiltIn: false,
        createdAt: Date.now(),
      };

      setCustomPresets((prev) => [...prev, newPreset]);
    },
    [],
  );

  /**
   * Delete a custom preset by ID.
   */
  const deletePreset = useCallback((id: string) => {
    setCustomPresets((prev) => prev.filter((p) => p.id !== id));
  }, []);

  /**
   * Find a preset by ID (searches both built-in and custom).
   */
  const getPresetById = useCallback(
    (id: string): JobPreset | undefined => {
      return presets.find((p) => p.id === id);
    },
    [presets],
  );

  /**
   * Find presets matching a URL pattern.
   */
  const findPresetsForUrl = useCallback(
    (url: string): JobPreset[] => {
      if (!url) return [];

      return presets.filter((preset) => {
        if (!preset.urlPatterns || preset.urlPatterns.length === 0) {
          return false;
        }
        return preset.urlPatterns.some((pattern) => {
          try {
            const regex = new RegExp(pattern, "i");
            return regex.test(url);
          } catch {
            return false;
          }
        });
      });
    },
    [presets],
  );

  return {
    presets,
    customPresets,
    savePreset,
    deletePreset,
    getPresetById,
    findPresetsForUrl,
  };
}

/**
 * Estimate time based on configuration.
 */
function estimateTime(config: PresetConfig): number {
  let time = 10; // Base time

  if (config.headless) {
    time += 20;
  }
  if (config.usePlaywright) {
    time += 15;
  }
  if (config.maxDepth) {
    time += config.maxDepth * 30;
  }
  if (config.maxPages) {
    time += Math.min(config.maxPages * 0.5, 300);
  }
  if (config.timeoutSeconds) {
    time = Math.max(time, config.timeoutSeconds);
  }

  return Math.round(time);
}

/**
 * Estimate CPU usage based on configuration.
 */
function estimateCpu(config: PresetConfig): "low" | "medium" | "high" {
  let score = 0;

  if (config.headless) score += 2;
  if (config.usePlaywright) score += 2;
  if (config.maxDepth && config.maxDepth > 2) score += 1;
  if (config.maxPages && config.maxPages > 100) score += 1;

  if (score >= 4) return "high";
  if (score >= 2) return "medium";
  return "low";
}

/**
 * Estimate memory usage based on configuration.
 */
function estimateMemory(config: PresetConfig): "low" | "medium" | "high" {
  let score = 0;

  if (config.headless) score += 2;
  if (config.usePlaywright) score += 2;
  if (config.maxDepth && config.maxDepth > 2) score += 1;
  if (config.maxPages && config.maxPages > 200) score += 1;

  if (score >= 4) return "high";
  if (score >= 2) return "medium";
  return "low";
}
