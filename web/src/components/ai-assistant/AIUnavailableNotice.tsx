/**
 * Purpose: Render the shared optional-AI unavailable notice used across assistant and settings surfaces.
 * Responsibilities: Present consistent eyebrow, title, and explanatory copy whenever AI helpers are disabled or degraded.
 * Scope: Browser-side presentation only; capability state is derived by callers.
 * Usage: Mount anywhere `describeAICapability()` returns a non-null message for operator-facing guidance.
 * Invariants/Assumptions: Manual workflows remain available outside the AI helper, and disabled optional AI should read as informational rather than alarming.
 */

interface AIUnavailableNoticeProps {
  message: string;
  eyebrow?: string;
  title?: string;
}

export function AIUnavailableNotice({
  message,
  eyebrow = "Optional subsystem",
  title = "AI assistance is not active right now",
}: AIUnavailableNoticeProps) {
  return (
    <div className="ai-unavailable-notice" role="status">
      <div className="ai-unavailable-notice__eyebrow">{eyebrow}</div>
      <div className="ai-unavailable-notice__title">{title}</div>
      <p className="ai-unavailable-notice__message">{message}</p>
    </div>
  );
}
