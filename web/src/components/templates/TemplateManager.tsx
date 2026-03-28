/**
 * Purpose: Coordinate the Templates route by composing the library, workspace, and assistant controllers.
 * Responsibilities: Bridge route inputs into the template controller hook, render the split route surfaces, and keep route-level errors visible without embedding the full template workflow inline.
 * Scope: Templates route coordination only; draft/session state and workspace behavior live in `TemplateRouteControllers.tsx`.
 * Usage: Render from the Templates route with the authoritative template name list and refresh callbacks.
 * Invariants/Assumptions: The controller hook remains the single source of truth for template-route state, and the route coordinator stays thin.
 */

import type { ComponentStatus } from "../../api";
import type { TemplatePromotionSeed } from "../../types/promotion";
import {
  TemplateAssistantController,
  TemplateLibraryController,
  TemplateManagerToolbar,
  TemplateWorkspaceController,
} from "./TemplateRouteControllers";
import { useTemplateRouteController } from "./useTemplateRouteController";

interface TemplateManagerProps {
  templateNames: string[];
  onTemplatesChanged: () => void;
  aiStatus?: ComponentStatus | null;
  promotionSeed?: TemplatePromotionSeed | null;
  onClearPromotionSeed?: () => void;
  onOpenSourceJob?: (jobId: string) => void;
}

export function TemplateManager({
  templateNames,
  onTemplatesChanged,
  aiStatus = null,
  promotionSeed = null,
  onClearPromotionSeed,
  onOpenSourceJob,
}: TemplateManagerProps) {
  const controller = useTemplateRouteController({
    templateNames,
    onTemplatesChanged,
    aiStatus,
    promotionSeed,
    onClearPromotionSeed,
  });

  return (
    <div className="template-manager-shell">
      <TemplateManagerToolbar {...controller.toolbarProps} />

      <section className="template-manager__workspace">
        <TemplateLibraryController {...controller.libraryProps} />

        <TemplateWorkspaceController
          {...controller.workspaceProps}
          onOpenSourceJob={onOpenSourceJob}
        />

        {!controller.isBuilderOpen ? (
          <TemplateAssistantController {...controller.assistantProps} />
        ) : null}
      </section>

      {controller.detailError ? (
        <div className="form-error">{controller.detailError}</div>
      ) : null}
    </div>
  );
}

export default TemplateManager;
