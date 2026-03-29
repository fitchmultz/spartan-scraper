/**
 * Purpose: Apply template-promotion seeds into the split template workspace controller.
 * Responsibilities: Consume route promotion seeds once, resolve named-template sources when needed, and replace the active workspace draft only after explicit confirmation when unsaved edits exist.
 * Scope: Promotion-seed orchestration only; saved-template detail ownership and general draft-session behavior stay in sibling hooks.
 * Usage: Called from `useTemplateRouteController()` after detail loading and draft-session hooks have been initialized.
 * Invariants/Assumptions: Promotion seeds are one-shot route inputs, promoted drafts never auto-save, and named-template promotions reuse the same detail fetch path as normal library selection.
 */

import { useEffect, useRef } from "react";

import type { Template, TemplateDetail } from "../../api";
import type { TemplatePromotionSeed } from "../../types/promotion";
import type { TemplateAssistantMode } from "../ai-assistant";
import type { DraftSource } from "./templateRouteControllerShared";

interface UseTemplatePromotionStateOptions {
  promotionSeed?: TemplatePromotionSeed | null;
  confirmReplaceCurrentDraft: (options?: {
    title?: string;
    reason?: string;
  }) => Promise<boolean>;
  fetchTemplateDetail: (name: string) => Promise<{
    detail: TemplateDetail | null;
    error: string | null;
  }>;
  loadDraft: (
    template: Template | undefined,
    source: DraftSource,
    options?: {
      originalName?: string | null;
      selectedName?: string | null;
      visible?: boolean;
    },
  ) => void;
  preventAutoSelect: () => void;
  setDetailError: (error: string | null) => void;
  setIsLoadingDetail: (value: boolean) => void;
  setPreviewUrl: (value: string) => void;
  setRailTab: (mode: TemplateAssistantMode) => void;
  setSelectedName: (name: string | null) => void;
  setSelectedTemplate: (template: TemplateDetail | null) => void;
}

export function useTemplatePromotionState({
  promotionSeed = null,
  confirmReplaceCurrentDraft,
  fetchTemplateDetail,
  loadDraft,
  preventAutoSelect,
  setDetailError,
  setIsLoadingDetail,
  setPreviewUrl,
  setRailTab,
  setSelectedName,
  setSelectedTemplate,
}: UseTemplatePromotionStateOptions) {
  const handledPromotionSeedRef = useRef<TemplatePromotionSeed | null>(null);

  useEffect(() => {
    if (!promotionSeed) {
      handledPromotionSeedRef.current = null;
      return;
    }

    if (handledPromotionSeedRef.current === promotionSeed) {
      return;
    }
    handledPromotionSeedRef.current = promotionSeed;

    let cancelled = false;

    const applyPromotionSeed = async () => {
      const confirmed = await confirmReplaceCurrentDraft({
        title: "Replace the current template draft?",
        reason:
          "This verified-job draft will replace the current local template draft. Keep the current draft if you still need those unsaved edits.",
      });
      if (!confirmed) {
        return;
      }

      preventAutoSelect();
      setDetailError(null);
      setPreviewUrl(promotionSeed.previewUrl ?? "");
      setRailTab("preview");

      if (promotionSeed.mode === "inline-template" && promotionSeed.template) {
        setSelectedName(null);
        setSelectedTemplate(null);
        loadDraft(
          {
            ...promotionSeed.template,
            name: promotionSeed.suggestedName,
          },
          "create",
          { selectedName: null },
        );
        return;
      }

      if (
        promotionSeed.mode === "named-template" &&
        promotionSeed.templateName
      ) {
        setIsLoadingDetail(true);
        const { detail, error } = await fetchTemplateDetail(
          promotionSeed.templateName,
        );
        if (cancelled) {
          return;
        }

        if (error) {
          setDetailError(error);
          setIsLoadingDetail(false);
          return;
        }

        setSelectedName(detail?.template?.name ?? promotionSeed.templateName);
        setSelectedTemplate(detail);
        loadDraft(
          {
            ...detail?.template,
            name: promotionSeed.suggestedName,
          },
          "duplicate",
          {
            originalName: detail?.template?.name,
            selectedName: detail?.template?.name ?? promotionSeed.templateName,
          },
        );
        setIsLoadingDetail(false);
        return;
      }

      setSelectedName(null);
      setSelectedTemplate(null);
      loadDraft(
        {
          name: promotionSeed.suggestedName,
        },
        "create",
        { selectedName: null },
      );
    };

    void applyPromotionSeed();

    return () => {
      cancelled = true;
    };
  }, [
    confirmReplaceCurrentDraft,
    fetchTemplateDetail,
    loadDraft,
    preventAutoSelect,
    promotionSeed,
    setDetailError,
    setIsLoadingDetail,
    setPreviewUrl,
    setRailTab,
    setSelectedName,
    setSelectedTemplate,
  ]);
}
