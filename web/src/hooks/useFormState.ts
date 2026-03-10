/**
 * Form State Hook
 *
 * Custom React hook for managing all form-related state for scrape, crawl,
 * and research forms. Includes all form fields and handlers, with automatic
 * headless/playwright constraint enforcement.
 *
 * @module useFormState
 */

import { useCallback, useEffect, useState } from "react";
import type { PresetConfig } from "../types/presets";

const DEFAULT_HEADERS = "";

export interface FormState {
  headless: boolean;
  usePlaywright: boolean;
  timeoutSeconds: number;
  authProfile: string;
  authBasic: string;
  headersRaw: string;
  cookiesRaw: string;
  queryRaw: string;
  loginUrl: string;
  loginUserSelector: string;
  loginPassSelector: string;
  loginSubmitSelector: string;
  loginUser: string;
  loginPass: string;
  extractTemplate: string;
  extractValidate: boolean;
  preProcessors: string;
  postProcessors: string;
  transformers: string;
  incremental: boolean;
  maxDepth: number;
  maxPages: number;
  webhookUrl: string;
  webhookEvents: string[];
  webhookSecret: string;
  // Network interception state
  interceptEnabled: boolean;
  interceptURLPatterns: string;
  interceptResourceTypes: string[];
  interceptCaptureRequestBody: boolean;
  interceptCaptureResponseBody: boolean;
  interceptMaxBodySize: number;
}

export interface FormActions {
  setHeadless: (value: boolean) => void;
  setUsePlaywright: (value: boolean) => void;
  setTimeoutSeconds: (value: number) => void;
  setAuthProfile: (value: string) => void;
  setAuthBasic: (value: string) => void;
  setHeadersRaw: (value: string) => void;
  setCookiesRaw: (value: string) => void;
  setQueryRaw: (value: string) => void;
  setLoginUrl: (value: string) => void;
  setLoginUserSelector: (value: string) => void;
  setLoginPassSelector: (value: string) => void;
  setLoginSubmitSelector: (value: string) => void;
  setLoginUser: (value: string) => void;
  setLoginPass: (value: string) => void;
  setExtractTemplate: (value: string) => void;
  setExtractValidate: (value: boolean) => void;
  setPreProcessors: (value: string) => void;
  setPostProcessors: (value: string) => void;
  setTransformers: (value: string) => void;
  setIncremental: (value: boolean) => void;
  setMaxDepth: (value: number) => void;
  setMaxPages: (value: number) => void;
  setWebhookUrl: (value: string) => void;
  setWebhookEvents: (value: string[]) => void;
  setWebhookSecret: (value: string) => void;
  // Network interception actions
  setInterceptEnabled: (value: boolean) => void;
  setInterceptURLPatterns: (value: string) => void;
  setInterceptResourceTypes: (value: string[]) => void;
  setInterceptCaptureRequestBody: (value: boolean) => void;
  setInterceptCaptureResponseBody: (value: boolean) => void;
  setInterceptMaxBodySize: (value: number) => void;
  /** Apply a preset configuration to the form state */
  applyPreset: (config: PresetConfig) => void;
}

export type FormController = FormState & FormActions;
export type ProfileOption = { name: string; parents: string[] };

const INITIAL_STATE: FormState = {
  headless: false,
  usePlaywright: false,
  timeoutSeconds: 30,
  authProfile: "",
  authBasic: "",
  headersRaw: DEFAULT_HEADERS,
  cookiesRaw: "",
  queryRaw: "",
  loginUrl: "",
  loginUserSelector: "",
  loginPassSelector: "",
  loginSubmitSelector: "",
  loginUser: "",
  loginPass: "",
  extractTemplate: "",
  extractValidate: false,
  preProcessors: "",
  postProcessors: "",
  transformers: "",
  incremental: false,
  maxDepth: 2,
  maxPages: 200,
  webhookUrl: "",
  webhookEvents: ["completed"],
  webhookSecret: "",
  // Network interception defaults
  interceptEnabled: false,
  interceptURLPatterns: "",
  interceptResourceTypes: ["xhr", "fetch"],
  interceptCaptureRequestBody: true,
  interceptCaptureResponseBody: true,
  interceptMaxBodySize: 1048576,
};

export function useFormState(): FormController {
  const [state, setState] = useState<FormState>(INITIAL_STATE);

  const setHeadless = useCallback((value: boolean) => {
    setState((prev) => ({ ...prev, headless: value }));
  }, []);

  const setUsePlaywright = useCallback((value: boolean) => {
    setState((prev) => ({ ...prev, usePlaywright: value }));
  }, []);

  const setTimeoutSeconds = useCallback((value: number) => {
    setState((prev) => ({ ...prev, timeoutSeconds: value }));
  }, []);

  const setAuthProfile = useCallback((value: string) => {
    setState((prev) => ({ ...prev, authProfile: value }));
  }, []);

  const setAuthBasic = useCallback((value: string) => {
    setState((prev) => ({ ...prev, authBasic: value }));
  }, []);

  const setHeadersRaw = useCallback((value: string) => {
    setState((prev) => ({ ...prev, headersRaw: value }));
  }, []);

  const setCookiesRaw = useCallback((value: string) => {
    setState((prev) => ({ ...prev, cookiesRaw: value }));
  }, []);

  const setQueryRaw = useCallback((value: string) => {
    setState((prev) => ({ ...prev, queryRaw: value }));
  }, []);

  const setLoginUrl = useCallback((value: string) => {
    setState((prev) => ({ ...prev, loginUrl: value }));
  }, []);

  const setLoginUserSelector = useCallback((value: string) => {
    setState((prev) => ({ ...prev, loginUserSelector: value }));
  }, []);

  const setLoginPassSelector = useCallback((value: string) => {
    setState((prev) => ({ ...prev, loginPassSelector: value }));
  }, []);

  const setLoginSubmitSelector = useCallback((value: string) => {
    setState((prev) => ({ ...prev, loginSubmitSelector: value }));
  }, []);

  const setLoginUser = useCallback((value: string) => {
    setState((prev) => ({ ...prev, loginUser: value }));
  }, []);

  const setLoginPass = useCallback((value: string) => {
    setState((prev) => ({ ...prev, loginPass: value }));
  }, []);

  const setExtractTemplate = useCallback((value: string) => {
    setState((prev) => ({ ...prev, extractTemplate: value }));
  }, []);

  const setExtractValidate = useCallback((value: boolean) => {
    setState((prev) => ({ ...prev, extractValidate: value }));
  }, []);

  const setPreProcessors = useCallback((value: string) => {
    setState((prev) => ({ ...prev, preProcessors: value }));
  }, []);

  const setPostProcessors = useCallback((value: string) => {
    setState((prev) => ({ ...prev, postProcessors: value }));
  }, []);

  const setTransformers = useCallback((value: string) => {
    setState((prev) => ({ ...prev, transformers: value }));
  }, []);

  const setIncremental = useCallback((value: boolean) => {
    setState((prev) => ({ ...prev, incremental: value }));
  }, []);

  const setMaxDepth = useCallback((value: number) => {
    setState((prev) => ({ ...prev, maxDepth: value }));
  }, []);

  const setMaxPages = useCallback((value: number) => {
    setState((prev) => ({ ...prev, maxPages: value }));
  }, []);

  const setWebhookUrl = useCallback((value: string) => {
    setState((prev) => ({ ...prev, webhookUrl: value }));
  }, []);

  const setWebhookEvents = useCallback((value: string[]) => {
    setState((prev) => ({ ...prev, webhookEvents: value }));
  }, []);

  const setWebhookSecret = useCallback((value: string) => {
    setState((prev) => ({ ...prev, webhookSecret: value }));
  }, []);

  const setInterceptEnabled = useCallback((value: boolean) => {
    setState((prev) => ({ ...prev, interceptEnabled: value }));
  }, []);

  const setInterceptURLPatterns = useCallback((value: string) => {
    setState((prev) => ({ ...prev, interceptURLPatterns: value }));
  }, []);

  const setInterceptResourceTypes = useCallback((value: string[]) => {
    setState((prev) => ({ ...prev, interceptResourceTypes: value }));
  }, []);

  const setInterceptCaptureRequestBody = useCallback((value: boolean) => {
    setState((prev) => ({ ...prev, interceptCaptureRequestBody: value }));
  }, []);

  const setInterceptCaptureResponseBody = useCallback((value: boolean) => {
    setState((prev) => ({ ...prev, interceptCaptureResponseBody: value }));
  }, []);

  const setInterceptMaxBodySize = useCallback((value: number) => {
    setState((prev) => ({ ...prev, interceptMaxBodySize: value }));
  }, []);

  useEffect(() => {
    if (!state.headless && state.usePlaywright) {
      setState((prev) => ({ ...prev, usePlaywright: false }));
    }
  }, [state.headless, state.usePlaywright]);

  const applyPreset = useCallback((config: PresetConfig) => {
    setState((prev) => ({
      ...prev,
      ...(config.headless !== undefined && { headless: config.headless }),
      ...(config.usePlaywright !== undefined && {
        usePlaywright: config.usePlaywright,
      }),
      ...(config.timeoutSeconds !== undefined && {
        timeoutSeconds: config.timeoutSeconds,
      }),
      ...(config.authProfile !== undefined && {
        authProfile: config.authProfile,
      }),
      ...(config.authBasic !== undefined && { authBasic: config.authBasic }),
      ...(config.headersRaw !== undefined && { headersRaw: config.headersRaw }),
      ...(config.cookiesRaw !== undefined && { cookiesRaw: config.cookiesRaw }),
      ...(config.queryRaw !== undefined && { queryRaw: config.queryRaw }),
      ...(config.loginUrl !== undefined && { loginUrl: config.loginUrl }),
      ...(config.loginUserSelector !== undefined && {
        loginUserSelector: config.loginUserSelector,
      }),
      ...(config.loginPassSelector !== undefined && {
        loginPassSelector: config.loginPassSelector,
      }),
      ...(config.loginSubmitSelector !== undefined && {
        loginSubmitSelector: config.loginSubmitSelector,
      }),
      ...(config.loginUser !== undefined && { loginUser: config.loginUser }),
      ...(config.loginPass !== undefined && { loginPass: config.loginPass }),
      ...(config.extractTemplate !== undefined && {
        extractTemplate: config.extractTemplate,
      }),
      ...(config.extractValidate !== undefined && {
        extractValidate: config.extractValidate,
      }),
      ...(config.preProcessors !== undefined && {
        preProcessors: config.preProcessors,
      }),
      ...(config.postProcessors !== undefined && {
        postProcessors: config.postProcessors,
      }),
      ...(config.transformers !== undefined && {
        transformers: config.transformers,
      }),
      ...(config.incremental !== undefined && {
        incremental: config.incremental,
      }),
      ...(config.maxDepth !== undefined && { maxDepth: config.maxDepth }),
      ...(config.maxPages !== undefined && { maxPages: config.maxPages }),
      ...(config.webhookUrl !== undefined && { webhookUrl: config.webhookUrl }),
      ...(config.webhookEvents !== undefined && {
        webhookEvents: config.webhookEvents,
      }),
      ...(config.webhookSecret !== undefined && {
        webhookSecret: config.webhookSecret,
      }),
      ...(config.interceptEnabled !== undefined && {
        interceptEnabled: config.interceptEnabled,
      }),
      ...(config.interceptURLPatterns !== undefined && {
        interceptURLPatterns: config.interceptURLPatterns,
      }),
      ...(config.interceptResourceTypes !== undefined && {
        interceptResourceTypes: config.interceptResourceTypes,
      }),
      ...(config.interceptCaptureRequestBody !== undefined && {
        interceptCaptureRequestBody: config.interceptCaptureRequestBody,
      }),
      ...(config.interceptCaptureResponseBody !== undefined && {
        interceptCaptureResponseBody: config.interceptCaptureResponseBody,
      }),
      ...(config.interceptMaxBodySize !== undefined && {
        interceptMaxBodySize: config.interceptMaxBodySize,
      }),
    }));
  }, []);

  return {
    ...state,
    setHeadless,
    setUsePlaywright,
    setTimeoutSeconds,
    setAuthProfile,
    setAuthBasic,
    setHeadersRaw,
    setCookiesRaw,
    setQueryRaw,
    setLoginUrl,
    setLoginUserSelector,
    setLoginPassSelector,
    setLoginSubmitSelector,
    setLoginUser,
    setLoginPass,
    setExtractTemplate,
    setExtractValidate,
    setPreProcessors,
    setPostProcessors,
    setTransformers,
    setIncremental,
    setMaxDepth,
    setMaxPages,
    setWebhookUrl,
    setWebhookEvents,
    setWebhookSecret,
    setInterceptEnabled,
    setInterceptURLPatterns,
    setInterceptResourceTypes,
    setInterceptCaptureRequestBody,
    setInterceptCaptureResponseBody,
    setInterceptMaxBodySize,
    applyPreset,
  };
}
