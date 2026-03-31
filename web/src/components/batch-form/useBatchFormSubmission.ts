/**
 * Purpose: Encapsulate batch-form validation, file parsing, and submit orchestration.
 * Responsibilities: Parse URL input, validate batch limits, read CSV/JSON uploads, build batch requests from shared form state, and expose small handlers for the presentational form sections.
 * Scope: Batch authoring behavior only; visual rendering stays in `BatchFormSections` and server-side execution stays in the parent container.
 * Usage: Call from `BatchForm` with the active tab, shared form controller, and batch-local state before wiring the returned handlers into the UI.
 * Invariants/Assumptions: URL lists must be non-empty and valid before submit, research batches require a non-empty query, and shared request config construction may throw validation errors that should surface through toasts.
 */

import {
  useCallback,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type FormEvent,
} from "react";

import type { DeviceEmulation } from "../../api";
import {
  buildAIExtractOptions,
  buildResearchAgenticOptions,
  buildSharedRequestConfig,
} from "../../lib/form-utils";
import type { FormController } from "../../hooks/useFormState";
import { useToast } from "../toast";
import {
  buildBatchCrawlRequest,
  buildBatchResearchRequest,
  buildBatchScrapeRequest,
  validateUrls as findInvalidBatchUrls,
} from "../../lib/batch-utils";
import { MAX_BATCH_SIZE, parseBatchUrls } from "../../lib/batch-urls";
import type { BatchFormTab } from "./types";

interface UseBatchFormSubmissionArgs {
  activeTab: BatchFormTab;
  form: FormController;
  urlsInput: string;
  setUrlsInput: (value: string) => void;
  maxDepth: number;
  maxPages: number;
  query: string;
  setQuery: (value: string) => void;
  device: DeviceEmulation | null;
  onSubmitScrape: (
    request: import("../../api").BatchScrapeRequest,
  ) => Promise<void>;
  onSubmitCrawl: (
    request: import("../../api").BatchCrawlRequest,
  ) => Promise<void>;
  onSubmitResearch: (
    request: import("../../api").BatchResearchRequest,
  ) => Promise<void>;
}

export function useBatchFormSubmission({
  activeTab,
  form,
  urlsInput,
  setUrlsInput,
  maxDepth,
  maxPages,
  query,
  setQuery,
  device,
  onSubmitScrape,
  onSubmitCrawl,
  onSubmitResearch,
}: UseBatchFormSubmissionArgs) {
  const toast = useToast();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [fileError, setFileError] = useState<string | null>(null);
  const [urlError, setUrlError] = useState<string | null>(null);

  const parsedUrls = useMemo(() => parseBatchUrls(urlsInput), [urlsInput]);
  const isValidBatchSize = parsedUrls.length <= MAX_BATCH_SIZE;

  const resolveAIExtractOptions = useCallback(() => {
    return buildAIExtractOptions(
      form.aiExtractEnabled,
      form.aiExtractMode,
      form.aiExtractPrompt,
      form.aiExtractSchema,
      form.aiExtractFields,
    );
  }, [
    form.aiExtractEnabled,
    form.aiExtractFields,
    form.aiExtractMode,
    form.aiExtractPrompt,
    form.aiExtractSchema,
  ]);

  const resolveAgenticOptions = useCallback(() => {
    return buildResearchAgenticOptions(
      form.agenticResearchEnabled,
      form.agenticResearchInstructions,
      form.agenticResearchMaxRounds,
      form.agenticResearchMaxFollowUpUrls,
    );
  }, [
    form.agenticResearchEnabled,
    form.agenticResearchInstructions,
    form.agenticResearchMaxFollowUpUrls,
    form.agenticResearchMaxRounds,
  ]);

  const handleFileUpload = useCallback(
    async (event: ChangeEvent<HTMLInputElement>) => {
      const file = event.target.files?.[0];
      if (!file) {
        return;
      }

      setFileError(null);

      try {
        const text = await file.text();
        let urls: string[] = [];

        if (file.name.endsWith(".json")) {
          const jobs = JSON.parse(text) as Array<{ url: string }>;
          if (!Array.isArray(jobs)) {
            throw new Error("JSON file must contain an array");
          }
          urls = jobs.map((job) => job.url).filter(Boolean);
        } else if (file.name.endsWith(".csv")) {
          const lines = text.split("\n").filter(Boolean);
          const hasHeader =
            lines.length > 0 &&
            (lines[0].toLowerCase().includes("url") ||
              !lines[0].startsWith("http"));
          const startIndex = hasHeader ? 1 : 0;
          urls = lines
            .slice(startIndex)
            .map((line) => line.split(",")[0]?.trim())
            .filter(
              (url): url is string => Boolean(url) && url.startsWith("http"),
            );
        } else {
          throw new Error("File must be .json or .csv");
        }

        if (urls.length === 0) {
          throw new Error("No valid URLs found in file");
        }

        if (urls.length > MAX_BATCH_SIZE) {
          throw new Error(
            `File contains ${urls.length} URLs, but maximum is ${MAX_BATCH_SIZE}`,
          );
        }

        setUrlsInput(urls.join("\n"));
      } catch (error) {
        setFileError(
          error instanceof Error ? error.message : "Failed to parse file",
        );
      }

      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    },
    [setUrlsInput],
  );

  const validateUrls = useCallback(() => {
    if (parsedUrls.length === 0) {
      setUrlError("At least one URL is required");
      return false;
    }
    if (parsedUrls.length > MAX_BATCH_SIZE) {
      setUrlError(`Maximum ${MAX_BATCH_SIZE} URLs allowed`);
      return false;
    }

    const invalid = findInvalidBatchUrls(parsedUrls);
    if (invalid.length > 0) {
      setUrlError(`Invalid URLs: ${invalid.slice(0, 3).join(", ")}`);
      return false;
    }

    setUrlError(null);
    return true;
  }, [parsedUrls]);

  const handleSubmitScrape = useCallback(async () => {
    if (!validateUrls()) {
      return;
    }

    try {
      const shared = buildSharedRequestConfig(form);
      const aiExtractOptions = resolveAIExtractOptions();
      const request = buildBatchScrapeRequest(
        parsedUrls,
        form.headless,
        form.usePlaywright,
        form.timeoutSeconds,
        shared.authProfile,
        shared.auth,
        shared.extract,
        shared.pipeline,
        form.incremental,
        shared.webhook,
        shared.screenshot,
        device || undefined,
        shared.networkIntercept,
        aiExtractOptions,
      );

      await onSubmitScrape(request);
    } catch (error) {
      toast.show({
        tone: "error",
        title: "Batch scrape configuration is invalid",
        description: error instanceof Error ? error.message : String(error),
      });
    }
  }, [
    device,
    form,
    onSubmitScrape,
    parsedUrls,
    resolveAIExtractOptions,
    toast,
    validateUrls,
  ]);

  const handleSubmitCrawl = useCallback(async () => {
    if (!validateUrls()) {
      return;
    }

    try {
      const shared = buildSharedRequestConfig(form);
      const aiExtractOptions = resolveAIExtractOptions();
      const request = buildBatchCrawlRequest(
        parsedUrls,
        maxDepth,
        maxPages,
        form.headless,
        form.usePlaywright,
        form.timeoutSeconds,
        shared.authProfile,
        shared.auth,
        shared.extract,
        shared.pipeline,
        form.incremental,
        shared.webhook,
        shared.screenshot,
        device || undefined,
        shared.networkIntercept,
        aiExtractOptions,
      );

      await onSubmitCrawl(request);
    } catch (error) {
      toast.show({
        tone: "error",
        title: "Batch crawl configuration is invalid",
        description: error instanceof Error ? error.message : String(error),
      });
    }
  }, [
    device,
    form,
    maxDepth,
    maxPages,
    onSubmitCrawl,
    parsedUrls,
    resolveAIExtractOptions,
    toast,
    validateUrls,
  ]);

  const handleSubmitResearch = useCallback(async () => {
    if (!validateUrls()) {
      return;
    }
    if (!query.trim()) {
      toast.show({
        tone: "warning",
        title: "Research query required",
        description: "Add the question you want this batch of URLs to answer.",
      });
      return;
    }

    try {
      const shared = buildSharedRequestConfig(form);
      const aiExtractOptions = resolveAIExtractOptions();
      const request = buildBatchResearchRequest(
        parsedUrls,
        query,
        maxDepth,
        maxPages,
        form.headless,
        form.usePlaywright,
        form.timeoutSeconds,
        shared.authProfile,
        shared.auth,
        shared.extract,
        shared.pipeline,
        shared.webhook,
        shared.screenshot,
        device || undefined,
        shared.networkIntercept,
        aiExtractOptions,
        resolveAgenticOptions(),
      );

      await onSubmitResearch(request);
    } catch (error) {
      toast.show({
        tone: "error",
        title: "Batch research configuration is invalid",
        description: error instanceof Error ? error.message : String(error),
      });
    }
  }, [
    device,
    form,
    maxDepth,
    maxPages,
    onSubmitResearch,
    parsedUrls,
    query,
    resolveAIExtractOptions,
    resolveAgenticOptions,
    toast,
    validateUrls,
  ]);

  const handleSubmit = useCallback(() => {
    switch (activeTab) {
      case "scrape":
        return handleSubmitScrape();
      case "crawl":
        return handleSubmitCrawl();
      case "research":
        return handleSubmitResearch();
    }
  }, [activeTab, handleSubmitCrawl, handleSubmitResearch, handleSubmitScrape]);

  const handleFormSubmit = useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      void handleSubmit();
    },
    [handleSubmit],
  );

  const handleUrlsInputChange = useCallback(
    (value: string) => {
      setUrlsInput(value);
      setUrlError(null);
    },
    [setUrlsInput],
  );

  const handleClear = useCallback(() => {
    setUrlsInput("");
    setQuery("");
    setUrlError(null);
    setFileError(null);
  }, [setQuery, setUrlsInput]);

  return {
    fileError,
    fileInputRef,
    handleClear,
    handleFileUpload,
    handleFormSubmit,
    handleUrlsInputChange,
    isValidBatchSize,
    parsedUrls,
    urlError,
  };
}
