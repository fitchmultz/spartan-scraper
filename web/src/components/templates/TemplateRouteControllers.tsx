/**
 * Purpose: Render the split list, workspace, and assistant controllers for the Templates route.
 * Responsibilities: Present the template library rail, workspace editor/builder surface, and assistant rail without embedding the entire route flow in one component.
 * Scope: Templates-route presentation only; route state and network orchestration live in `useTemplateRouteController.ts`.
 * Usage: Imported by `TemplateManager.tsx` after route-level template state has been resolved.
 * Invariants/Assumptions: The library always reflects authoritative template names, the workspace remains the single save surface, and assistant actions never auto-save into the library.
 */

import type {
  ComponentStatus,
  SelectorRule,
  Template,
  TemplateDetail,
} from "../../api";
import type { TemplatePromotionSeed } from "../../types/promotion";
import {
  TemplateAssistantSection,
  type TemplateAssistantMode,
} from "../ai-assistant";
import { PromotionDraftNotice } from "../promotion/PromotionDraftNotice";
import { ResumableSettingsDraftNotice } from "../settings/ResumableSettingsDraftNotice";
import { VisualSelectorBuilder } from "../VisualSelectorBuilder";
import { TemplateEditorInline } from "./TemplateEditorInline";
import {
  createSelectorDraft,
  isBuiltInTemplateName,
  type TemplateDraftState,
} from "./templateRouteControllerShared";

interface TemplateManagerToolbarProps {
  onStartCreate: () => void;
  onOpenAssistant: () => void;
  onOpenVisualBuilder: () => void;
}

interface TemplateLibraryControllerProps {
  templateNames: string[];
  selectedName: string | null;
  onSelectTemplate: (name: string) => void;
}

interface TemplateWorkspaceControllerProps {
  isBuilderOpen: boolean;
  draftTemplate: Template;
  activeDraft: TemplateDraftState;
  activeDraftSource: "selected" | "create" | "duplicate";
  selectedTemplate: TemplateDetail | null;
  readOnly: boolean;
  canDeleteSelectedTemplate: boolean;
  editorTitle: string;
  promotionSeed?: TemplatePromotionSeed | null;
  hiddenDraft: {
    draft: TemplateDraftState;
    originalName: string | null;
  } | null;
  isHiddenDraftDirty: boolean;
  isLoadingDetail: boolean;
  hasWorkspaceDraft: boolean;
  isDirty: boolean;
  isSaving: boolean;
  saveError: string | null;
  saveNotice: string | null;
  onOpenPreview: () => void;
  onOpenAssistant: () => void;
  onStartDuplicate: () => void;
  onDelete: () => void;
  onOpenSourceJob?: (jobId: string) => void;
  onClearPromotionSeed?: () => void;
  onResumeDraft: () => void;
  onDiscardDraft: () => void;
  onUpdateDraft: (
    updater: (current: TemplateDraftState) => TemplateDraftState,
  ) => void;
  onUpdateSelector: (
    selectorId: string,
    key: keyof SelectorRule,
    value: string | boolean,
  ) => void;
  onSave: () => void;
  onReset: () => void;
  onClose: () => void;
  onBuilderSave: (template: Template) => void;
  onBuilderCancel: () => void;
}

interface TemplateAssistantControllerProps {
  railTab: TemplateAssistantMode;
  draftTemplate: Template;
  previewUrl: string;
  aiStatus?: ComponentStatus | null;
  onModeChange: (mode: TemplateAssistantMode) => void;
  onPreviewUrlChange: (value: string) => void;
  onApplyTemplate: (template: Template) => void;
}

function describeTemplate(detail: TemplateDetail | null) {
  const selectors = detail?.template?.selectors?.length ?? 0;
  const jsonld = detail?.template?.jsonld?.length ?? 0;
  const regex = detail?.template?.regex?.length ?? 0;
  return `${selectors} selector${selectors === 1 ? "" : "s"} · ${jsonld} JSON-LD · ${regex} regex`;
}

export function TemplateManagerToolbar({
  onStartCreate,
  onOpenAssistant,
  onOpenVisualBuilder,
}: TemplateManagerToolbarProps) {
  return (
    <section className="panel template-manager__toolbar">
      <div className="template-manager__toolbar-copy">
        <h3>Template workspace</h3>
        <p>
          Edit rules inline, preview them against a real page, and use AI
          without losing your place.
        </p>
      </div>

      <div className="template-manager__toolbar-actions">
        <button
          type="button"
          className="btn btn--secondary"
          onClick={onStartCreate}
        >
          New Template
        </button>
        <button
          type="button"
          className="btn btn--secondary"
          onClick={onOpenAssistant}
        >
          Open AI assistant
        </button>
        <button
          type="button"
          className="btn btn--secondary"
          onClick={onOpenVisualBuilder}
        >
          Open Visual Builder
        </button>
      </div>
    </section>
  );
}

export function TemplateLibraryController({
  templateNames,
  selectedName,
  onSelectTemplate,
}: TemplateLibraryControllerProps) {
  return (
    <aside className="template-manager__library">
      <div className="template-manager__library-header">
        <h4>Templates</h4>
        <span>{templateNames.length}</span>
      </div>

      {templateNames.length > 0 ? (
        <ul
          className="template-manager__list"
          aria-label="Extraction template list"
        >
          {templateNames.map((name) => {
            const isSelected = name === selectedName;
            const templateKind = isBuiltInTemplateName(name)
              ? "Built-in"
              : "Custom";

            return (
              <li key={name}>
                <button
                  type="button"
                  className={`template-manager__list-item ${
                    isSelected ? "is-selected" : ""
                  }`}
                  onClick={() => onSelectTemplate(name)}
                >
                  <div className="template-manager__list-item-top">
                    <strong>{name}</strong>
                    <span
                      className={`badge ${
                        templateKind === "Built-in" ? "running" : "success"
                      }`}
                    >
                      {templateKind}
                    </span>
                  </div>
                  <span>Open in workspace</span>
                </button>
              </li>
            );
          })}
        </ul>
      ) : null}
    </aside>
  );
}

export function TemplateWorkspaceController({
  isBuilderOpen,
  draftTemplate,
  activeDraft,
  activeDraftSource,
  selectedTemplate,
  readOnly,
  canDeleteSelectedTemplate,
  editorTitle,
  promotionSeed = null,
  hiddenDraft,
  isHiddenDraftDirty,
  isLoadingDetail,
  hasWorkspaceDraft,
  isDirty,
  isSaving,
  saveError,
  saveNotice,
  onOpenPreview,
  onOpenAssistant,
  onStartDuplicate,
  onDelete,
  onOpenSourceJob,
  onClearPromotionSeed,
  onResumeDraft,
  onDiscardDraft,
  onUpdateDraft,
  onUpdateSelector,
  onSave,
  onReset,
  onClose,
  onBuilderSave,
  onBuilderCancel,
}: TemplateWorkspaceControllerProps) {
  if (isBuilderOpen) {
    return (
      <section className="template-manager__builder-surface">
        <VisualSelectorBuilder
          key={`${activeDraft.name || "new"}-${activeDraftSource}`}
          initialTemplate={draftTemplate}
          onSave={onBuilderSave}
          onCancel={onBuilderCancel}
        />
      </section>
    );
  }

  return (
    <section className="template-manager__editor-surface">
      <div className="template-manager__detail-header">
        <div>
          <div className="template-manager__detail-eyebrow">
            <span className={`badge ${readOnly ? "running" : "success"}`}>
              {readOnly ? "Built-in template" : "Editable workspace"}
            </span>
          </div>
          <h3>{editorTitle}</h3>
          <p>
            {selectedTemplate && activeDraftSource === "selected"
              ? describeTemplate(selectedTemplate)
              : "Changes stay in the workspace until you explicitly save them."}
          </p>
        </div>

        <div className="template-manager__detail-actions">
          <button
            type="button"
            className="btn btn--secondary btn--small"
            onClick={onOpenPreview}
          >
            Preview
          </button>
          <button
            type="button"
            className="btn btn--secondary btn--small"
            onClick={onOpenAssistant}
          >
            Open AI assistant
          </button>
          {readOnly ? (
            <button
              type="button"
              className="btn btn--secondary btn--small"
              onClick={onStartDuplicate}
            >
              Duplicate to Edit
            </button>
          ) : canDeleteSelectedTemplate ? (
            <button
              type="button"
              className="btn btn--danger btn--small"
              onClick={onDelete}
            >
              Delete
            </button>
          ) : null}
        </div>
      </div>

      {promotionSeed ? (
        <PromotionDraftNotice
          title="Template draft seeded from a verified job"
          description="This workspace starts from the reusable extraction structure Spartan could safely recover from the successful source job."
          seed={promotionSeed}
          onOpenSourceJob={onOpenSourceJob}
          onClear={onClearPromotionSeed}
        />
      ) : null}

      {hiddenDraft ? (
        <ResumableSettingsDraftNotice
          title={`Template draft for ${
            hiddenDraft.draft.name ||
            hiddenDraft.originalName ||
            "the current workspace"
          }${isHiddenDraftDirty ? " has unsaved edits." : " is still available in this tab."}`}
          description="Close keeps this draft available in the current tab. Resume it when you want to continue editing, or discard it explicitly once you no longer need it."
          resumeLabel="Resume template draft"
          discardLabel="Discard template draft"
          onResume={onResumeDraft}
          onDiscard={onDiscardDraft}
        />
      ) : null}

      {isLoadingDetail &&
      activeDraftSource === "selected" &&
      !hasWorkspaceDraft ? (
        <div className="template-manager__empty">Loading template details…</div>
      ) : null}

      {!hiddenDraft ? (
        <TemplateEditorInline
          draft={activeDraft}
          readOnly={readOnly}
          isDirty={isDirty}
          isSaving={isSaving}
          error={saveError}
          notice={saveNotice}
          onNameChange={(value) =>
            onUpdateDraft((current) => ({ ...current, name: value }))
          }
          onUpdateSelector={onUpdateSelector}
          onAddSelector={() =>
            onUpdateDraft((current) => ({
              ...current,
              selectors: [...current.selectors, createSelectorDraft()],
            }))
          }
          onRemoveSelector={(selectorId) =>
            onUpdateDraft((current) => ({
              ...current,
              selectors: current.selectors.filter(
                (selector) => selector.id !== selectorId,
              ),
            }))
          }
          onJsonldTextChange={(value) =>
            onUpdateDraft((current) => ({ ...current, jsonldText: value }))
          }
          onRegexTextChange={(value) =>
            onUpdateDraft((current) => ({ ...current, regexText: value }))
          }
          onNormalizeTextChange={(value) =>
            onUpdateDraft((current) => ({ ...current, normalizeText: value }))
          }
          onSave={onSave}
          onReset={onReset}
          onClose={hasWorkspaceDraft ? onClose : undefined}
          onDiscard={hasWorkspaceDraft ? onDiscardDraft : undefined}
        />
      ) : null}
    </section>
  );
}

export function TemplateAssistantController({
  railTab,
  draftTemplate,
  previewUrl,
  aiStatus = null,
  onModeChange,
  onPreviewUrlChange,
  onApplyTemplate,
}: TemplateAssistantControllerProps) {
  return (
    <TemplateAssistantSection
      mode={railTab}
      onModeChange={onModeChange}
      draftTemplate={draftTemplate}
      previewUrl={previewUrl}
      aiStatus={aiStatus}
      onPreviewUrlChange={onPreviewUrlChange}
      onApplyTemplate={onApplyTemplate}
    />
  );
}
