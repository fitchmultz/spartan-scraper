/**
 * Purpose: Share template-route controller types and pure helpers across the split template hooks.
 * Responsibilities: Define draft-session shapes, build persisted draft sessions, and detect dirty workspace state without introducing rendering concerns.
 * Scope: Template-route controller helpers only.
 * Usage: Imported by the split template controller hooks.
 * Invariants/Assumptions: Draft sessions snapshot the last persisted template state, and dirty checks compare normalized template snapshots instead of editor-local IDs.
 */

import { deepEqual } from "../../lib/diff-utils";
import type { Template } from "../../api";
import {
  buildDraftFromTemplate,
  buildTemplateSnapshot,
  type TemplateDraftState,
} from "./templateEditorUtils";

export type DraftSource = "selected" | "create" | "duplicate";

export interface TemplateWorkspaceDraftSession {
  source: DraftSource;
  originalName: string | null;
  selectedName: string | null;
  initialTemplate: Template;
  draft: TemplateDraftState;
  visible: boolean;
}

export function createTemplateWorkspaceDraftSession(
  template: Template | undefined,
  source: DraftSource,
  options?: {
    originalName?: string | null;
    selectedName?: string | null;
    visible?: boolean;
  },
): TemplateWorkspaceDraftSession {
  const draft = buildDraftFromTemplate(template);

  return {
    source,
    originalName:
      options?.originalName !== undefined
        ? options.originalName
        : (template?.name ?? null),
    selectedName:
      options?.selectedName !== undefined
        ? options.selectedName
        : (template?.name ?? null),
    initialTemplate: buildTemplateSnapshot(draft),
    draft,
    visible: options?.visible ?? true,
  };
}

export function isTemplateWorkspaceDraftDirty(
  session: TemplateWorkspaceDraftSession,
): boolean {
  return !deepEqual(
    buildTemplateSnapshot(session.draft),
    session.initialTemplate,
  );
}
