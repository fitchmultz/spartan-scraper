/**
 * Purpose: Render a consistent resume-or-discard notice for hidden Settings drafts that remain available in the current browser tab.
 * Responsibilities: Explain that Close is non-destructive, surface the current draft status, and provide explicit resume/discard actions with consistent copy.
 * Scope: Shared Settings draft notice chrome only; draft persistence and confirmation behavior stay in the owning editor.
 * Usage: Mount from Settings editors whenever a hidden draft should be resumable instead of silently discarded.
 * Invariants/Assumptions: Resume always reopens the existing local draft, and discard always routes through an explicit confirmation flow owned by the caller.
 */

interface ResumableSettingsDraftNoticeProps {
  title: string;
  description: string;
  onResume: () => void;
  onDiscard: () => void;
  resumeLabel?: string;
  discardLabel?: string;
}

export function ResumableSettingsDraftNotice({
  title,
  description,
  onResume,
  onDiscard,
  resumeLabel = "Resume draft",
  discardLabel = "Discard draft",
}: ResumableSettingsDraftNoticeProps) {
  return (
    <div className="rounded-md border border-amber-300 bg-amber-50 p-4 text-sm text-amber-950">
      <p>{title}</p>
      <p className="mt-2">{description}</p>
      <div className="mt-3 flex flex-wrap gap-2">
        <button type="button" className="button-secondary" onClick={onResume}>
          {resumeLabel}
        </button>
        <button type="button" className="button-secondary" onClick={onDiscard}>
          {discardLabel}
        </button>
      </div>
    </div>
  );
}
