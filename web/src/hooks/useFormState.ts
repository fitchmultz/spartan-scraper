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
}

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
};

export function useFormState(): FormState & FormActions {
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

  useEffect(() => {
    if (!state.headless && state.usePlaywright) {
      setState((prev) => ({ ...prev, usePlaywright: false }));
    }
  }, [state.headless, state.usePlaywright]);

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
  };
}
