/**
 * Purpose: Own template-library selection and detail loading for the Templates route.
 * Responsibilities: Track the selected template, load detail on demand, keep the library auto-selection rules stable, and refresh selected draft snapshots when the authoritative saved template changes.
 * Scope: Template-detail selection/loading only; workspace draft actions and promotion handling stay in sibling hooks.
 * Usage: Called from `useTemplateRouteController()` before composing draft-session and promotion behavior.
 * Invariants/Assumptions: Saved template detail is fetched from the API on demand, built-in status is derived from template detail when available, and untouched selected drafts refresh from authoritative template detail.
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type Dispatch,
  type SetStateAction,
} from "react";

import { getTemplate, type TemplateDetail } from "../../api";
import { getApiBaseUrl } from "../../lib/api-config";
import { getApiErrorMessage } from "../../lib/api-errors";
import { BUILT_IN_TEMPLATE_NAMES } from "./templateEditorUtils";
import {
  createTemplateWorkspaceDraftSession,
  isTemplateWorkspaceDraftDirty,
  type TemplateWorkspaceDraftSession,
} from "./templateRouteControllerShared";

interface UseTemplateDetailLoaderOptions {
  templateNames: string[];
  workspaceDraftSession: TemplateWorkspaceDraftSession | null;
  setWorkspaceDraftSession: Dispatch<
    SetStateAction<TemplateWorkspaceDraftSession | null>
  >;
}

interface FetchTemplateDetailResult {
  detail: TemplateDetail | null;
  error: string | null;
}

export function useTemplateDetailLoader({
  templateNames,
  workspaceDraftSession,
  setWorkspaceDraftSession,
}: UseTemplateDetailLoaderOptions) {
  const [selectedName, setSelectedName] = useState<string | null>(
    () => workspaceDraftSession?.selectedName ?? templateNames[0] ?? null,
  );
  const [selectedTemplate, setSelectedTemplate] =
    useState<TemplateDetail | null>(null);
  const [isLoadingDetail, setIsLoadingDetail] = useState(false);
  const [detailError, setDetailError] = useState<string | null>(null);
  const [shouldAutoSelectFirst, setShouldAutoSelectFirst] = useState(
    () => workspaceDraftSession === null,
  );

  const selectedTemplateData = selectedTemplate?.template ?? null;
  const selectedIsBuiltIn =
    selectedTemplate?.is_built_in ??
    (selectedName
      ? BUILT_IN_TEMPLATE_NAMES.includes(
          selectedName as (typeof BUILT_IN_TEMPLATE_NAMES)[number],
        )
      : false);

  const syncSelectedDraft = useCallback(
    (detail: TemplateDetail | null, name: string) => {
      setSelectedTemplate(detail);
      setWorkspaceDraftSession((current) => {
        if (
          !current ||
          current.source !== "selected" ||
          current.originalName !== detail?.template?.name ||
          isTemplateWorkspaceDraftDirty(current)
        ) {
          return current;
        }

        return createTemplateWorkspaceDraftSession(
          detail?.template,
          "selected",
          {
            originalName: detail?.template?.name,
            selectedName: name,
            visible: current.visible,
          },
        );
      });
    },
    [setWorkspaceDraftSession],
  );

  const fetchTemplateDetail = useCallback(
    async (name: string): Promise<FetchTemplateDetailResult> => {
      try {
        const response = await getTemplate({
          baseUrl: getApiBaseUrl(),
          path: { name },
        });

        if (response.error) {
          throw new Error(
            getApiErrorMessage(
              response.error,
              "Failed to load template details.",
            ),
          );
        }

        return {
          detail: response.data ?? null,
          error: null,
        };
      } catch (error) {
        return {
          detail: null,
          error:
            error instanceof Error
              ? error.message
              : "Failed to load template details.",
        };
      }
    },
    [],
  );

  const preventAutoSelect = useCallback(() => {
    setShouldAutoSelectFirst(false);
  }, []);

  useEffect(() => {
    if (templateNames.length === 0) {
      setSelectedName(null);
      setSelectedTemplate(null);
      return;
    }

    if (!selectedName && shouldAutoSelectFirst) {
      setSelectedName(templateNames[0] ?? null);
      setShouldAutoSelectFirst(false);
    }
  }, [selectedName, shouldAutoSelectFirst, templateNames]);

  useEffect(() => {
    if (!selectedName) {
      setSelectedTemplate(null);
      setDetailError(null);
      return;
    }

    let cancelled = false;

    const loadSelectedTemplate = async () => {
      setIsLoadingDetail(true);
      setDetailError(null);

      const { detail, error } = await fetchTemplateDetail(selectedName);
      if (cancelled) {
        return;
      }

      if (error) {
        setDetailError(error);
        setSelectedTemplate(null);
        setIsLoadingDetail(false);
        return;
      }

      syncSelectedDraft(detail, selectedName);
      setIsLoadingDetail(false);
    };

    void loadSelectedTemplate();

    return () => {
      cancelled = true;
    };
  }, [fetchTemplateDetail, selectedName, syncSelectedDraft]);

  return useMemo(
    () => ({
      detailError,
      fetchTemplateDetail,
      isLoadingDetail,
      preventAutoSelect,
      selectedIsBuiltIn,
      selectedName,
      selectedTemplate,
      selectedTemplateData,
      setDetailError,
      setIsLoadingDetail,
      setSelectedName,
      setSelectedTemplate,
      syncSelectedDraft,
    }),
    [
      detailError,
      fetchTemplateDetail,
      isLoadingDetail,
      preventAutoSelect,
      selectedIsBuiltIn,
      selectedName,
      selectedTemplate,
      selectedTemplateData,
      syncSelectedDraft,
    ],
  );
}
