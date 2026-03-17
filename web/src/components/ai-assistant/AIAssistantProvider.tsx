/**
 * Purpose: Own the shared integrated AI assistant UI state and route-aware context across the web app.
 * Responsibilities: Persist assistant expansion and width, store the current route context, and expose imperative open/close/toggle helpers to route surfaces and launch points.
 * Scope: Web AI assistant shell state only; route adapters still own route-specific API calls and apply actions.
 * Usage: Wrap the application shell with `AIAssistantProvider` and consume state through `useAIAssistant()`.
 * Invariants/Assumptions: Width stays within a bounded desktop range, persisted state fails open to sensible defaults, and assistant output never mutates route state without explicit route-level apply actions.
 */

import {
  createContext,
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

export type AssistantSurface = "job-submission" | "templates" | "results";

export type AssistantContext =
  | {
      surface: "job-submission";
      jobType: "scrape" | "crawl" | "research";
      url?: string;
      query?: string;
      templateName?: string;
      formSnapshot: Record<string, unknown>;
    }
  | {
      surface: "templates";
      templateName?: string;
      templateSnapshot?: Record<string, unknown>;
      selectedUrl?: string;
    }
  | {
      surface: "results";
      jobId: string;
      resultFormat: string;
      selectedResultIndex: number;
      resultSummary?: string | null;
    };

export interface AIAssistantController {
  isOpen: boolean;
  width: number;
  context: AssistantContext | null;
  open: (context: AssistantContext) => void;
  close: () => void;
  toggle: () => void;
  setContext: (context: AssistantContext) => void;
  setWidth: (width: number) => void;
}

export const AI_ASSISTANT_OPEN_KEY = "spartan.ai-assistant.open";
export const AI_ASSISTANT_WIDTH_KEY = "spartan.ai-assistant.width";

const AI_ASSISTANT_DEFAULT_WIDTH = 380;
const AI_ASSISTANT_MIN_WIDTH = 340;
const AI_ASSISTANT_MAX_WIDTH = 460;

function clampWidth(width: number): number {
  return Math.min(
    AI_ASSISTANT_MAX_WIDTH,
    Math.max(AI_ASSISTANT_MIN_WIDTH, Math.round(width)),
  );
}

function getBrowserStorage(): Pick<Storage, "getItem" | "setItem"> | null {
  if (typeof window === "undefined") {
    return null;
  }

  try {
    const storage = window.localStorage;
    if (
      !storage ||
      typeof storage.getItem !== "function" ||
      typeof storage.setItem !== "function"
    ) {
      return null;
    }

    return storage;
  } catch {
    return null;
  }
}

function readStoredOpen(): boolean {
  const storage = getBrowserStorage();
  if (!storage) {
    return true;
  }

  const rawValue = storage.getItem(AI_ASSISTANT_OPEN_KEY);
  if (rawValue === "false") {
    return false;
  }
  if (rawValue === "true") {
    return true;
  }
  return true;
}

function readStoredWidth(): number {
  const storage = getBrowserStorage();
  if (!storage) {
    return AI_ASSISTANT_DEFAULT_WIDTH;
  }

  const rawValue = Number.parseInt(
    storage.getItem(AI_ASSISTANT_WIDTH_KEY) ?? "",
    10,
  );
  if (Number.isFinite(rawValue)) {
    return clampWidth(rawValue);
  }
  return AI_ASSISTANT_DEFAULT_WIDTH;
}

export const AIAssistantContext = createContext<AIAssistantController | null>(
  null,
);

interface AIAssistantProviderProps {
  children: ReactNode;
}

export function AIAssistantProvider({ children }: AIAssistantProviderProps) {
  const [isOpen, setIsOpen] = useState<boolean>(() => readStoredOpen());
  const [width, setWidthState] = useState<number>(() => readStoredWidth());
  const [context, setContextState] = useState<AssistantContext | null>(null);

  useEffect(() => {
    const storage = getBrowserStorage();
    if (!storage) {
      return;
    }

    storage.setItem(AI_ASSISTANT_OPEN_KEY, isOpen ? "true" : "false");
  }, [isOpen]);

  useEffect(() => {
    const storage = getBrowserStorage();
    if (!storage) {
      return;
    }

    storage.setItem(AI_ASSISTANT_WIDTH_KEY, String(width));
  }, [width]);

  const open = useCallback((nextContext: AssistantContext) => {
    setContextState(nextContext);
    setIsOpen(true);
  }, []);

  const close = useCallback(() => {
    setIsOpen(false);
  }, []);

  const toggle = useCallback(() => {
    setIsOpen((previous) => !previous);
  }, []);

  const setContext = useCallback((nextContext: AssistantContext) => {
    setContextState(nextContext);
  }, []);

  const setWidth = useCallback((nextWidth: number) => {
    setWidthState(clampWidth(nextWidth));
  }, []);

  const value = useMemo<AIAssistantController>(
    () => ({
      isOpen,
      width,
      context,
      open,
      close,
      toggle,
      setContext,
      setWidth,
    }),
    [close, context, isOpen, open, setContext, setWidth, toggle, width],
  );

  return (
    <AIAssistantContext.Provider value={value}>
      {children}
    </AIAssistantContext.Provider>
  );
}
