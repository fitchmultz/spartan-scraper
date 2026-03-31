/**
 * Purpose: Own template-library selection and detail loading for the Templates route.
 * Responsibilities: Track the selected template, load detail on demand, and keep the library auto-selection rules stable without mutating workspace draft state.
 * Scope: Template-detail selection/loading only; workspace draft refresh, draft actions, and promotion handling stay in sibling hooks.
 * Usage: Called from `useTemplateRouteController()` before composing draft-session and promotion behavior.
 * Invariants/Assumptions: Saved template detail is fetched from the API on demand, built-in status is derived from template detail when available, and loading template detail never rewrites the local workspace draft by itself.
 */

import { useCallback, useEffect, useMemo, useState } from "react";

import { getTemplate, type TemplateDetail } from "../../api";
import { getApiBaseUrl } from "../../lib/api-config";
import { getApiErrorMessage } from "../../lib/api-errors";
import { isBuiltInTemplateName } from "./templateRouteControllerShared";

interface UseTemplateDetailLoaderOptions {
  templateNames: string[];
  initialSelectedName: string | null;
  hasInitialDraftSession: boolean;
}

interface FetchTemplateDetailResult {
  detail: TemplateDetail | null;
  error: string | null;
}

export function useTemplateDetailLoader({
  templateNames,
  initialSelectedName,
  hasInitialDraftSession,
}: UseTemplateDetailLoaderOptions) {
  const [selectedName, setSelectedName] = useState<string | null>(
    () => initialSelectedName ?? templateNames[0] ?? null,
  );
  const [selectedTemplate, setSelectedTemplate] =
    useState<TemplateDetail | null>(null);
  const [isLoadingDetail, setIsLoadingDetail] = useState(false);
  const [detailError, setDetailError] = useState<string | null>(null);
  const [shouldAutoSelectFirst, setShouldAutoSelectFirst] = useState(
    () => !hasInitialDraftSession,
  );

  const selectedTemplateData = selectedTemplate?.template ?? null;
  const selectedIsBuiltIn =
    selectedTemplate?.is_built_in ?? isBuiltInTemplateName(selectedName);

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

      setSelectedTemplate(detail);
      setIsLoadingDetail(false);
    };

    void loadSelectedTemplate();

    return () => {
      cancelled = true;
    };
  }, [fetchTemplateDetail, selectedName]);

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
    ],
  );
}
