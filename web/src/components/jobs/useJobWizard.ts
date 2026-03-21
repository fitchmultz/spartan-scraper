/**
 * Purpose: Coordinate guided job-wizard mode, per-job draft persistence, and shared form hydration for the new-job route.
 * Responsibilities: Persist wizard mode and per-job drafts, sync tab switches into the shared form controller, validate step progression, and expose preset/draft orchestration helpers.
 * Scope: Web single-job creation wizard behavior only.
 * Usage: Call from `JobSubmissionContainer` and bind the returned state/actions to wizard steps, expert mode, and preset submission flows.
 * Invariants/Assumptions: `useFormState` remains the canonical owner of shared runtime/extraction fields, job-specific URL/query/device values are stored separately, and local-storage corruption must fail open to empty drafts.
 */

import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import {
  buildPresetConfig,
  createInitialJobDraftLocalState,
  extractLocalDraftFields,
  type CrawlDraftFields,
  type JobDraftLocalState,
  type ResearchDraftFields,
  type ScrapeDraftFields,
} from "../../lib/job-drafts";
import type { FormController } from "../../hooks/useFormState";
import type { JobType, PresetConfig } from "../../types/presets";

export type WizardStepId = "basics" | "runtime" | "extraction" | "review";

const STEP_ORDER: WizardStepId[] = [
  "basics",
  "runtime",
  "extraction",
  "review",
];

const JOB_CREATION_MODE_KEY = "spartan.job-creation.mode";
const JOB_DRAFT_KEY_PREFIX = "spartan.job-draft";

interface UseJobWizardOptions {
  activeTab: JobType;
  formState: FormController;
}

function getDraftStorageKey(jobType: JobType): string {
  return `${JOB_DRAFT_KEY_PREFIX}.${jobType}`;
}

function readStoredMode(): boolean {
  if (typeof window === "undefined") {
    return false;
  }

  return window.localStorage.getItem(JOB_CREATION_MODE_KEY) === "expert";
}

function readStoredDraft(jobType: JobType): PresetConfig | null {
  if (typeof window === "undefined") {
    return null;
  }

  const rawValue = window.localStorage.getItem(getDraftStorageKey(jobType));
  if (!rawValue) {
    return null;
  }

  try {
    const parsed = JSON.parse(rawValue) as PresetConfig;
    return parsed && typeof parsed === "object" ? parsed : null;
  } catch {
    return null;
  }
}

function writeStoredDraft(jobType: JobType, config: PresetConfig): void {
  if (typeof window === "undefined") {
    return;
  }

  window.localStorage.setItem(
    getDraftStorageKey(jobType),
    JSON.stringify(config),
  );
}

function clearStoredDraft(jobType: JobType): void {
  if (typeof window === "undefined") {
    return;
  }

  window.localStorage.removeItem(getDraftStorageKey(jobType));
}

function updateJobDraftState(
  previousState: JobDraftLocalState,
  jobType: JobType,
  config: PresetConfig | null | undefined,
): JobDraftLocalState {
  if (jobType === "scrape") {
    return {
      ...previousState,
      scrape: extractLocalDraftFields("scrape", config) as ScrapeDraftFields,
    };
  }

  if (jobType === "crawl") {
    return {
      ...previousState,
      crawl: extractLocalDraftFields("crawl", config) as CrawlDraftFields,
    };
  }

  return {
    ...previousState,
    research: extractLocalDraftFields(
      "research",
      config,
    ) as ResearchDraftFields,
  };
}

function buildReviewWarnings(jobType: JobType, config: PresetConfig): string[] {
  const warnings: string[] = [];

  if (!config.extractTemplate && !config.aiExtractEnabled) {
    warnings.push(
      "No extraction template or AI extraction is configured yet, so this run may return raw content only.",
    );
  }

  if (!config.headless && (config.screenshotEnabled || config.device)) {
    warnings.push(
      "Browser-only capture settings are configured while headless execution is off, so those settings will not take effect.",
    );
  }

  if (jobType === "research" && !config.urls?.trim()) {
    warnings.push(
      "Research runs need source URLs to produce useful synthesis.",
    );
  }

  if (jobType === "crawl" && config.sitemapOnly && !config.sitemapURL?.trim()) {
    warnings.push("Sitemap-only mode is enabled without a sitemap URL.");
  }

  return warnings;
}

export function useJobWizard({ activeTab, formState }: UseJobWizardOptions) {
  const [activeStep, setActiveStepState] = useState<WizardStepId>("basics");
  const [completedSteps, setCompletedSteps] = useState<WizardStepId[]>([]);
  const [expertMode, setExpertMode] = useState<boolean>(() => readStoredMode());
  const [validationErrors, setValidationErrors] = useState<
    Partial<Record<WizardStepId, string[]>>
  >({});
  const [draftSavedAt, setDraftSavedAt] = useState<number | null>(null);
  const [localState, setLocalState] = useState<JobDraftLocalState>(() =>
    createInitialJobDraftLocalState(),
  );
  const [isInitialized, setIsInitialized] = useState(false);

  const draftsRef = useRef<Record<JobType, PresetConfig | null>>({
    scrape: null,
    crawl: null,
    research: null,
  });
  const initialActiveTabRef = useRef<JobType>(activeTab);
  const previousTabRef = useRef<JobType>(activeTab);

  const hydrateSharedForm = useCallback(
    (config: PresetConfig | null | undefined) => {
      formState.reset();
      if (config && Object.keys(config).length > 0) {
        formState.applyPreset(config);
      }
    },
    [formState.applyPreset, formState.reset],
  );

  const getCurrentConfig = useCallback(
    (jobType: JobType = activeTab): PresetConfig => {
      if (jobType !== activeTab) {
        return draftsRef.current[jobType] ?? {};
      }

      return buildPresetConfig(activeTab, formState, localState);
    },
    [activeTab, formState, localState],
  );

  const activeConfig = getCurrentConfig(activeTab);
  const activeConfigKey = JSON.stringify(activeConfig);

  const validateStep = useCallback(
    (step: WizardStepId, jobType: JobType = activeTab): string[] => {
      const config = getCurrentConfig(jobType);

      if (step !== "basics") {
        return [];
      }

      if (jobType === "scrape" && !config.url?.trim()) {
        return ["A target URL is required before continuing."];
      }

      if (jobType === "crawl") {
        return config.url?.trim()
          ? []
          : ["A crawl start URL is required before continuing."];
      }

      if (jobType === "research") {
        const errors: string[] = [];
        if (!config.query?.trim()) {
          errors.push("A research query is required before continuing.");
        }
        if (!config.urls?.trim()) {
          errors.push("At least one source URL is required before continuing.");
        }
        return errors;
      }

      return [];
    },
    [activeTab, getCurrentConfig],
  );

  useEffect(() => {
    if (typeof window !== "undefined") {
      window.localStorage.setItem(
        JOB_CREATION_MODE_KEY,
        expertMode ? "expert" : "guided",
      );
    }
  }, [expertMode]);

  useLayoutEffect(() => {
    const storedDrafts: Record<JobType, PresetConfig | null> = {
      scrape: readStoredDraft("scrape"),
      crawl: readStoredDraft("crawl"),
      research: readStoredDraft("research"),
    };

    draftsRef.current = storedDrafts;

    setLocalState((previousState) => {
      let nextState = previousState;
      nextState = updateJobDraftState(nextState, "scrape", storedDrafts.scrape);
      nextState = updateJobDraftState(nextState, "crawl", storedDrafts.crawl);
      nextState = updateJobDraftState(
        nextState,
        "research",
        storedDrafts.research,
      );
      return nextState;
    });

    hydrateSharedForm(storedDrafts[initialActiveTabRef.current]);
    setIsInitialized(true);
  }, [hydrateSharedForm]);

  useEffect(() => {
    if (!isInitialized) {
      return;
    }

    const previousTab = previousTabRef.current;
    if (previousTab === activeTab) {
      return;
    }

    const previousConfig = buildPresetConfig(
      previousTab,
      formState,
      localState,
    );
    draftsRef.current[previousTab] = previousConfig;
    writeStoredDraft(previousTab, previousConfig);

    const nextConfig = draftsRef.current[activeTab];
    hydrateSharedForm(nextConfig);
    setValidationErrors({});
    setCompletedSteps([]);
    setActiveStepState("basics");
    setDraftSavedAt(nextConfig ? Date.now() : null);
    previousTabRef.current = activeTab;
  }, [activeTab, formState, hydrateSharedForm, isInitialized, localState]);

  useEffect(() => {
    if (!isInitialized) {
      return;
    }

    const timeoutId = window.setTimeout(() => {
      const parsedConfig = JSON.parse(activeConfigKey) as PresetConfig;
      draftsRef.current[activeTab] = parsedConfig;
      writeStoredDraft(activeTab, parsedConfig);
      setDraftSavedAt(Date.now());
    }, 300);

    return () => window.clearTimeout(timeoutId);
  }, [activeTab, activeConfigKey, isInitialized]);

  useEffect(() => {
    if (!isInitialized) {
      return;
    }

    if (validateStep("basics").length > 0) {
      setCompletedSteps((previousSteps) => {
        if (!previousSteps.includes("basics")) {
          return previousSteps;
        }

        return previousSteps.filter((step) => step !== "basics");
      });
    }
  }, [isInitialized, validateStep]);

  const updateLocalState = useCallback(
    (
      jobType: JobType,
      patch:
        | Partial<ScrapeDraftFields>
        | Partial<CrawlDraftFields>
        | Partial<ResearchDraftFields>,
    ) => {
      setLocalState((previousState) => {
        if (jobType === "scrape") {
          return {
            ...previousState,
            scrape: {
              ...previousState.scrape,
              ...patch,
            },
          };
        }

        if (jobType === "crawl") {
          return {
            ...previousState,
            crawl: {
              ...previousState.crawl,
              ...patch,
            },
          };
        }

        return {
          ...previousState,
          research: {
            ...previousState.research,
            ...patch,
          },
        };
      });
    },
    [],
  );

  const validateCurrentStep = useCallback((): string[] => {
    const errors = validateStep(activeStep, activeTab);
    setValidationErrors((previousErrors) => ({
      ...previousErrors,
      [activeStep]: errors,
    }));
    return errors;
  }, [activeStep, activeTab, validateStep]);

  const goBack = useCallback(() => {
    const currentIndex = STEP_ORDER.indexOf(activeStep);
    if (currentIndex <= 0) {
      return;
    }
    setActiveStepState(STEP_ORDER[currentIndex - 1]);
  }, [activeStep]);

  const goNext = useCallback(() => {
    const errors = validateStep(activeStep, activeTab);
    setValidationErrors((previousErrors) => ({
      ...previousErrors,
      [activeStep]: errors,
    }));

    if (errors.length > 0) {
      return;
    }

    const currentIndex = STEP_ORDER.indexOf(activeStep);
    if (currentIndex >= STEP_ORDER.length - 1) {
      return;
    }

    setCompletedSteps((previousSteps) => {
      if (previousSteps.includes(activeStep)) {
        return previousSteps;
      }
      return [...previousSteps, activeStep];
    });
    setActiveStepState(STEP_ORDER[currentIndex + 1]);
  }, [activeStep, activeTab, validateStep]);

  const setActiveStep = useCallback(
    (step: WizardStepId) => {
      const targetIndex = STEP_ORDER.indexOf(step);
      const canNavigate = STEP_ORDER.slice(0, targetIndex).every(
        (candidateStep) => completedSteps.includes(candidateStep),
      );

      if (!canNavigate && step !== activeStep) {
        return;
      }

      setActiveStepState(step);
    },
    [activeStep, completedSteps],
  );

  const applyPreset = useCallback(
    (config: PresetConfig, jobType: JobType = activeTab) => {
      const baseConfig =
        jobType === activeTab
          ? getCurrentConfig(activeTab)
          : (draftsRef.current[jobType] ?? {});
      const mergedConfig = {
        ...baseConfig,
        ...config,
      } satisfies PresetConfig;

      draftsRef.current[jobType] = mergedConfig;
      writeStoredDraft(jobType, mergedConfig);
      setLocalState((previousState) =>
        updateJobDraftState(previousState, jobType, mergedConfig),
      );
      setDraftSavedAt(Date.now());

      if (jobType === activeTab) {
        formState.applyPreset(config);
        setValidationErrors({});
      }
    },
    [activeTab, formState, getCurrentConfig],
  );

  const clearDraft = useCallback(
    (jobType: JobType = activeTab) => {
      draftsRef.current[jobType] = null;
      clearStoredDraft(jobType);
      setLocalState((previousState) =>
        updateJobDraftState(previousState, jobType, null),
      );

      if (jobType === activeTab) {
        hydrateSharedForm(null);
        setValidationErrors({});
        setCompletedSteps([]);
        setActiveStepState("basics");
        setDraftSavedAt(null);
      }
    },
    [activeTab, hydrateSharedForm],
  );

  const reviewWarnings = useMemo(
    () => buildReviewWarnings(activeTab, activeConfig),
    [activeConfig, activeTab],
  );

  return {
    activeStep,
    completedSteps,
    expertMode,
    setExpertMode,
    validationErrors,
    draftSavedAt,
    localState,
    updateLocalState,
    validateCurrentStep,
    goBack,
    goNext,
    setActiveStep,
    getCurrentConfig,
    applyPreset,
    clearDraft,
    reviewWarnings,
  };
}
