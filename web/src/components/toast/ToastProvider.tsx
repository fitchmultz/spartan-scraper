/**
 * Purpose: Own the global toast and confirmation state for the Spartan Scraper web UI.
 * Responsibilities: Expose imperative toast lifecycle helpers, manage a shared confirmation dialog, and render the global notification chrome alongside application content.
 * Scope: Web-only transient operator feedback and destructive-action confirmation flows.
 * Usage: Wrap the root application with `ToastProvider` and consume the controller with `useToast()`.
 * Invariants/Assumptions: Toast IDs stay stable for updates, loading toasts persist until explicitly updated or dismissed, and at most one confirmation dialog is active at a time.
 */

import {
  createContext,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { ConfirmDialog, type ConfirmDialogOptions } from "./ConfirmDialog";
import { ToastContainer } from "./ToastContainer";

export type ToastTone = "success" | "info" | "warning" | "error" | "loading";

export interface ToastAction {
  label: string;
  onSelect: () => void | Promise<void>;
}

export interface ToastInput {
  id?: string;
  tone: ToastTone;
  title: string;
  description?: string;
  durationMs?: number;
  action?: ToastAction;
  dismissible?: boolean;
}

export interface ToastRecord extends ToastInput {
  id: string;
  dismissible: boolean;
  updatedAt: number;
}

export interface ToastController {
  show: (toast: ToastInput) => string;
  update: (id: string, patch: Partial<ToastInput>) => void;
  dismiss: (id: string) => void;
  confirm: (options: ConfirmDialogOptions) => Promise<boolean>;
}

interface ConfirmRequest extends ConfirmDialogOptions {
  id: string;
}

const DEFAULT_SUCCESS_DURATION_MS = 4_500;
const DEFAULT_INFO_DURATION_MS = 5_000;
const DEFAULT_WARNING_DURATION_MS = 7_000;
const DEFAULT_ERROR_DURATION_MS = 8_000;
const MAX_STORED_TOASTS = 8;

let toastSequence = 0;
let confirmSequence = 0;

function createToastId(): string {
  toastSequence += 1;
  return `toast-${Date.now()}-${toastSequence}`;
}

function createConfirmId(): string {
  confirmSequence += 1;
  return `confirm-${Date.now()}-${confirmSequence}`;
}

function getDefaultDuration(tone: ToastTone): number | undefined {
  switch (tone) {
    case "success":
      return DEFAULT_SUCCESS_DURATION_MS;
    case "info":
      return DEFAULT_INFO_DURATION_MS;
    case "warning":
      return DEFAULT_WARNING_DURATION_MS;
    case "error":
      return DEFAULT_ERROR_DURATION_MS;
    case "loading":
      return undefined;
  }
}

function normalizeToastInput(input: ToastInput): ToastRecord {
  const id = input.id?.trim() ? input.id : createToastId();
  return {
    ...input,
    id,
    durationMs: input.durationMs ?? getDefaultDuration(input.tone),
    dismissible: input.dismissible ?? input.tone !== "loading",
    updatedAt: Date.now(),
  };
}

function applyToastPatch(
  current: ToastRecord,
  patch: Partial<ToastInput>,
): ToastRecord {
  const nextTone = patch.tone ?? current.tone;
  const nextDuration =
    patch.durationMs !== undefined
      ? patch.durationMs
      : patch.tone && patch.tone !== current.tone
        ? getDefaultDuration(nextTone)
        : current.durationMs;
  const nextDismissible =
    patch.dismissible !== undefined
      ? patch.dismissible
      : patch.tone && patch.tone !== current.tone
        ? nextTone !== "loading"
        : current.dismissible;

  return {
    ...current,
    ...patch,
    id: current.id,
    tone: nextTone,
    durationMs: nextDuration,
    dismissible: nextDismissible,
    updatedAt: Date.now(),
  };
}

export const ToastContext = createContext<ToastController | null>(null);

interface ToastProviderProps {
  children: ReactNode;
}

export function ToastProvider({ children }: ToastProviderProps) {
  const [toasts, setToasts] = useState<ToastRecord[]>([]);
  const [confirmRequest, setConfirmRequest] = useState<ConfirmRequest | null>(
    null,
  );
  const confirmResolverRef = useRef<((value: boolean) => void) | null>(null);

  useEffect(() => {
    return () => {
      confirmResolverRef.current?.(false);
      confirmResolverRef.current = null;
    };
  }, []);

  const dismiss = useCallback((id: string) => {
    setToasts((current) => current.filter((toast) => toast.id !== id));
  }, []);

  const show = useCallback((input: ToastInput) => {
    const toast = normalizeToastInput(input);
    setToasts((current) => {
      const withoutExisting = current.filter((item) => item.id !== toast.id);
      return [...withoutExisting, toast].slice(-MAX_STORED_TOASTS);
    });
    return toast.id;
  }, []);

  const update = useCallback((id: string, patch: Partial<ToastInput>) => {
    setToasts((current) =>
      current.map((toast) =>
        toast.id === id ? applyToastPatch(toast, patch) : toast,
      ),
    );
  }, []);

  const resolveConfirm = useCallback((value: boolean) => {
    confirmResolverRef.current?.(value);
    confirmResolverRef.current = null;
    setConfirmRequest(null);
  }, []);

  const confirm = useCallback((options: ConfirmDialogOptions) => {
    confirmResolverRef.current?.(false);

    return new Promise<boolean>((resolve) => {
      confirmResolverRef.current = resolve;
      setConfirmRequest({
        id: createConfirmId(),
        tone: options.tone ?? "warning",
        confirmLabel: options.confirmLabel ?? "Confirm",
        cancelLabel: options.cancelLabel ?? "Cancel",
        ...options,
      });
    });
  }, []);

  const value = useMemo<ToastController>(
    () => ({
      show,
      update,
      dismiss,
      confirm,
    }),
    [confirm, dismiss, show, update],
  );

  return (
    <ToastContext.Provider value={value}>
      {children}
      <ToastContainer toasts={toasts} onDismiss={dismiss} />
      <ConfirmDialog
        request={confirmRequest}
        onCancel={() => resolveConfirm(false)}
        onConfirm={() => resolveConfirm(true)}
      />
    </ToastContext.Provider>
  );
}
