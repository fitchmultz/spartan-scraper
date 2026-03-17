/**
 * Purpose: Provide the supported hook for accessing the global toast controller.
 * Responsibilities: Read the toast context and fail fast when the provider is missing.
 * Scope: Thin React hook wrapper only.
 * Usage: Call `const toast = useToast()` from components rendered within `ToastProvider`.
 * Invariants/Assumptions: The hook is only called from the mounted application tree where `ToastProvider` is present.
 */

import { useContext } from "react";
import { ToastContext, type ToastController } from "./ToastProvider";

export function useToast(): ToastController {
  const value = useContext(ToastContext);
  if (!value) {
    throw new Error("useToast must be used within ToastProvider");
  }
  return value;
}
