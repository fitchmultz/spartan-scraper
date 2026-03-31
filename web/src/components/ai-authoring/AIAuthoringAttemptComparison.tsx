/**
 * Purpose: Render the shared AI attempt history and comparison panels for authoring modals.
 * Responsibilities: Show attempt selection controls, restore guidance, and display baseline/selected artifact comparisons with consistent empty states.
 * Scope: Shared Web UI results presentation for AI generators and debuggers.
 * Usage: Mount inside an AI authoring modal after the request form and pass the active history controller plus artifact labels.
 * Invariants/Assumptions: The selected attempt drives save/retry behavior, baseline comparisons only reference older attempts, and artifact diffs use the existing shared diff viewer.
 */

import { AIAttemptHistoryList } from "../AIAttemptHistoryList";
import { AIAuthoringAttemptPanel } from "../AIAuthoringAttemptPanel";
import { AICandidateDiffView } from "../AICandidateDiffView";
import type {
  AIAttempt,
  AIAttemptHistoryController,
} from "../../hooks/useAIAttemptHistory";
import type { JsTargetScript, RenderProfile } from "../../api";

type ArtifactKind = "render-profile" | "pipeline-js";
type AIAuthoringArtifact = RenderProfile | JsTargetScript;

interface AIAuthoringAttemptComparisonProps<
  TArtifact extends AIAuthoringArtifact,
> {
  history: AIAttemptHistoryController<TArtifact>;
  artifactKind: ArtifactKind;
  emptyBaselineMessage: string;
  emptySelectedMessage: string;
  onRestoreGuidance: (attempt: AIAttempt<TArtifact>) => void;
  onEditInSettings?: (attempt: AIAttempt<TArtifact>) => void;
}

export function AIAuthoringAttemptComparison<
  TArtifact extends AIAuthoringArtifact,
>({
  history,
  artifactKind,
  emptyBaselineMessage,
  emptySelectedMessage,
  onRestoreGuidance,
  onEditInSettings,
}: AIAuthoringAttemptComparisonProps<TArtifact>) {
  const { activeAttempt, baselineAttempt, latestAttempt } = history;

  return (
    <>
      <AIAttemptHistoryList
        attempts={history.attempts}
        activeAttemptId={history.activeAttemptId}
        baselineAttemptId={history.baselineAttemptId}
        onSelectAttempt={history.selectAttempt}
        onSelectBaseline={history.selectBaseline}
        onRestoreGuidance={onRestoreGuidance}
        onEditInSettings={onEditInSettings}
      />

      {baselineAttempt ? (
        <AIAuthoringAttemptPanel
          key={baselineAttempt.id}
          label={`Comparison baseline · Attempt ${baselineAttempt.ordinal}`}
          routeId={baselineAttempt.routeId}
          provider={baselineAttempt.provider}
          model={baselineAttempt.model}
          visualContextUsed={baselineAttempt.visualContextUsed}
          recheckStatus={baselineAttempt.recheckStatus}
          recheckEngine={baselineAttempt.recheckEngine}
          recheckError={baselineAttempt.recheckError}
          issues={baselineAttempt.issues}
          resolvedGoal={baselineAttempt.resolvedGoal}
          explanation={baselineAttempt.explanation}
          rawResponse={baselineAttempt.rawResponse}
          manualEdit={baselineAttempt.manualEdit}
          muted
        >
          {baselineAttempt.artifact ? (
            <AICandidateDiffView
              artifactKind={artifactKind}
              selectedArtifact={baselineAttempt.artifact}
              selectedLabel={`Attempt ${baselineAttempt.ordinal}`}
            />
          ) : (
            <div className="text-sm text-slate-400">{emptyBaselineMessage}</div>
          )}
        </AIAuthoringAttemptPanel>
      ) : null}

      {activeAttempt ? (
        <AIAuthoringAttemptPanel
          key={activeAttempt.id}
          label={`${activeAttempt.id === latestAttempt?.id ? "Latest" : "Selected"} candidate · Attempt ${activeAttempt.ordinal}`}
          routeId={activeAttempt.routeId}
          provider={activeAttempt.provider}
          model={activeAttempt.model}
          visualContextUsed={activeAttempt.visualContextUsed}
          recheckStatus={activeAttempt.recheckStatus}
          recheckEngine={activeAttempt.recheckEngine}
          recheckError={activeAttempt.recheckError}
          issues={activeAttempt.issues}
          resolvedGoal={activeAttempt.resolvedGoal}
          explanation={activeAttempt.explanation}
          rawResponse={activeAttempt.rawResponse}
          manualEdit={activeAttempt.manualEdit}
        >
          {activeAttempt.artifact ? (
            <AICandidateDiffView
              artifactKind={artifactKind}
              baselineArtifact={baselineAttempt?.artifact ?? null}
              selectedArtifact={activeAttempt.artifact}
              baselineLabel={
                baselineAttempt
                  ? `Attempt ${baselineAttempt.ordinal}`
                  : "Comparison baseline"
              }
              selectedLabel={`Attempt ${activeAttempt.ordinal}`}
            />
          ) : (
            <div className="text-sm text-slate-400">{emptySelectedMessage}</div>
          )}
        </AIAuthoringAttemptPanel>
      ) : null}
    </>
  );
}
