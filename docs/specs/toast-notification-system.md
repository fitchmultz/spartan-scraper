# Toast Notification System

**Status:** Completed
**Primary surface:** Web UI global feedback layer

## Summary

Introduce a global toast notification system for success, error, loading, and progress feedback across the Web UI.

The current UI uses a fragmented feedback model: inline errors, `alert()`, `confirm()`, `console.error`, and generic loading text. That produces inconsistent operator feedback and breaks product polish. This spec defines one notification layer that all transient operations should use.

## Problems This Solves

- `alert()` interrupts flow and feels low quality.
- `confirm()` is a jarring browser-native fallback.
- `console.error` provides no operator-facing feedback.
- Many actions have no visible completion confirmation.
- Async operations such as submit, refresh, check, cancel, delete, and export need consistent success/error handling.

## Goals

- Provide one consistent feedback system for transient operations.
- Support success, info, warning, error, and loading states.
- Allow optional actions such as Retry, Undo, View, or Open.
- Work across job submission, automation, templates, exports, and AI flows.
- Be accessible and theme-aware.

## API Design

```ts
type ToastTone = "success" | "info" | "warning" | "error" | "loading";

interface ToastAction {
  label: string;
  onSelect: () => void | Promise<void>;
}

interface ToastInput {
  id?: string;
  tone: ToastTone;
  title: string;
  description?: string;
  durationMs?: number;
  action?: ToastAction;
  dismissible?: boolean;
}

interface ToastController {
  show: (toast: ToastInput) => string;
  update: (id: string, patch: Partial<ToastInput>) => void;
  dismiss: (id: string) => void;
}
```

## Default Behavior Rules

- success/info: auto-dismiss after roughly 4–6 seconds
- warning/error: stay longer
- loading: persist until updated or dismissed
- max visible: 4 on desktop, 2–3 on mobile
- placement: top-right desktop, bottom-oriented on mobile

## Migration Priority

1. job submission success/failure
2. batch submission and cancel flows
3. watch create/update/delete/check flows
4. export schedule operations
5. template save/delete flows
6. destructive confirmation flows currently using `confirm()`

## Acceptance Criteria

- The Web UI no longer depends on `alert()` for normal transient feedback.
- Operator-triggered async actions surface visible success/failure states.
- Loading states can upgrade in place to success or error.
- Toasts are keyboard accessible and screen-reader friendly.
- The system is reusable across routes without new third-party lock-in.
