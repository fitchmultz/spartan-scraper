/**
 * Purpose: Provide safe sessionStorage-backed React state for serializable UI sessions.
 * Responsibilities: Hydrate JSON state from browser sessionStorage, persist updates, and expose an explicit reset helper that falls back to in-memory state when storage is unavailable.
 * Scope: Browser-only transient UI state persistence for the current tab.
 * Usage: Call `const [state, setState, clearState] = useSessionStorageState(key, initialValue)` inside client components that need tab-scoped draft recovery.
 * Invariants/Assumptions: Stored values must be JSON-serializable, storage failures must fail open without breaking interaction flows, and resetting always returns to the provided initial value.
 */

import {
  useCallback,
  useEffect,
  useState,
  type Dispatch,
  type SetStateAction,
} from "react";

function getSessionStorage(): Storage | null {
  if (typeof window === "undefined") {
    return null;
  }

  try {
    return window.sessionStorage;
  } catch {
    return null;
  }
}

function resolveInitialValue<T>(initialValue: T | (() => T)): T {
  return initialValue instanceof Function ? initialValue() : initialValue;
}

function readStoredValue<T>(
  storageKey: string,
  initialValue: T | (() => T),
): T {
  const fallback = resolveInitialValue(initialValue);
  const storage = getSessionStorage();
  if (!storage) {
    return fallback;
  }

  try {
    const raw = storage.getItem(storageKey);
    if (!raw) {
      return fallback;
    }
    return JSON.parse(raw) as T;
  } catch {
    storage.removeItem(storageKey);
    return fallback;
  }
}

export function useSessionStorageState<T>(
  storageKey: string | null,
  initialValue: T | (() => T),
): readonly [T, Dispatch<SetStateAction<T>>, () => void] {
  const [state, setState] = useState<T>(() =>
    storageKey
      ? readStoredValue(storageKey, initialValue)
      : resolveInitialValue(initialValue),
  );

  useEffect(() => {
    if (!storageKey) {
      return;
    }

    const storage = getSessionStorage();
    if (!storage) {
      return;
    }

    try {
      storage.setItem(storageKey, JSON.stringify(state));
    } catch {
      // Ignore storage failures; drafts should still work in memory.
    }
  }, [storageKey, state]);

  const clearState = useCallback(() => {
    if (storageKey) {
      const storage = getSessionStorage();
      try {
        storage?.removeItem(storageKey);
      } catch {
        // Ignore storage failures; state still resets in memory.
      }
    }

    setState(resolveInitialValue(initialValue));
  }, [storageKey, initialValue]);

  return [state, setState, clearState] as const;
}
