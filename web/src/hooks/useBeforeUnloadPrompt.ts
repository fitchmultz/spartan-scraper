/**
 * Purpose: Register a browser beforeunload warning when in-tab draft persistence is not enough to prevent data loss.
 * Responsibilities: Attach and remove a `beforeunload` listener based on a reactive enabled flag, and fail open when the browser does not support custom messaging.
 * Scope: Browser-tab exit warnings only; in-app route changes and draft persistence stay in higher-level workflow hooks.
 * Usage: Call `useBeforeUnloadPrompt(enabled, message)` from components that own unsaved local drafts that would be lost if the tab closes.
 * Invariants/Assumptions: Modern browsers ignore custom beforeunload text, but setting `event.returnValue` is still required to trigger the native warning dialog.
 */

import { useEffect } from "react";

export function useBeforeUnloadPrompt(
  enabled: boolean,
  message = "You have unsaved changes that will be lost if you leave this tab.",
) {
  useEffect(() => {
    if (!enabled || typeof window === "undefined") {
      return;
    }

    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = message;
      return message;
    };

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
    };
  }, [enabled, message]);
}
