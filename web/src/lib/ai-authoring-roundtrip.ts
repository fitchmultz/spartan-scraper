/**
 * Purpose: Build continuation guidance for round-trip AI authoring after native Settings edits.
 * Responsibilities: Preserve operator instructions and serialize the selected manual-edit baseline when generation endpoints lack a first-class baseline field.
 * Scope: Frontend-only AI authoring continuation helpers.
 * Usage: Import from generator modals before retrying from a manually edited candidate.
 * Invariants/Assumptions: Serialized artifact context is only appended when a caller supplies an artifact baseline.
 */

export function buildManualEditContinuationGuidance<TArtifact>(input: {
  operatorInstructions: string;
  artifact: TArtifact | null;
  artifactLabel: string;
}): string | undefined {
  const trimmedInstructions = input.operatorInstructions.trim();

  if (!input.artifact) {
    return trimmedInstructions || undefined;
  }

  const serializedArtifact = JSON.stringify(input.artifact, null, 2);

  return [
    trimmedInstructions ? `Operator guidance:\n${trimmedInstructions}` : null,
    `Use the following ${input.artifactLabel} as the continuation baseline. Preserve unchanged behavior unless page evidence or operator guidance requires a change.`,
    serializedArtifact,
  ]
    .filter(Boolean)
    .join("\n\n");
}
