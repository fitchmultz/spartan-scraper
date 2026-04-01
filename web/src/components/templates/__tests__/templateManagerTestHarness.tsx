/**
 * Purpose: Provide shared mocks, fixtures, and render helpers for TemplateManager route tests.
 * Responsibilities: Centralize API mocks, wrapper rendering, and common promotion/template fixtures.
 * Scope: Test-only support for TemplateManager Vitest suites.
 * Usage: Import helpers into focused TemplateManager test files and call `setupTemplateManagerTest()` inside each suite module.
 * Invariants/Assumptions: TemplateManager stays route-scoped, API calls remain mocked, and the visual builder continues to be replaced with a deterministic inline stub.
 */

import { render } from "@testing-library/react";
import { beforeEach, vi } from "vitest";

const templateApiMocks = vi.hoisted(() => ({
  getTemplate: vi.fn(),
  createTemplate: vi.fn(),
  updateTemplate: vi.fn(),
  deleteTemplate: vi.fn(),
  testSelector: vi.fn(),
  aiTemplateGenerate: vi.fn(),
  aiTemplateDebug: vi.fn(),
}));

vi.mock("../../../api", () => templateApiMocks);

vi.mock("../../../lib/api-config", () => ({
  getApiBaseUrl: vi.fn(() => "http://127.0.0.1:8741"),
}));

vi.mock("../../VisualSelectorBuilder", () => ({
  VisualSelectorBuilder: ({
    initialTemplate,
    onSave,
    onCancel,
  }: {
    initialTemplate?: { name?: string };
    onSave: (template: { name?: string }) => void;
    onCancel: () => void;
  }) => (
    <div>
      <div>Visual Builder Mock</div>
      <div>{initialTemplate?.name ?? "new template"}</div>
      <button
        type="button"
        onClick={() =>
          onSave({ name: initialTemplate?.name ?? "builder-saved" })
        }
      >
        Save Builder
      </button>
      <button type="button" onClick={onCancel}>
        Cancel Builder
      </button>
    </div>
  ),
}));

import { AIAssistantProvider } from "../../ai-assistant";
import { ToastProvider } from "../../toast";

export type TemplateManagerProps = {
  templateNames?: string[];
  onTemplatesChanged?: (...args: unknown[]) => void;
  promotionSeed?: unknown;
};

export function getTemplateApiMocks() {
  return templateApiMocks;
}

export const onTemplatesChanged = vi.fn();

export function setupTemplateManagerTest() {
  beforeEach(() => {
    vi.clearAllMocks();
    onTemplatesChanged.mockReset();
    if (typeof window.localStorage.clear === "function") {
      window.localStorage.clear();
    }
    if (typeof window.sessionStorage.clear === "function") {
      window.sessionStorage.clear();
    }
    vi.stubGlobal(
      "confirm",
      vi.fn(() => true),
    );
  });
}

export async function renderTemplateManager(
  props: Partial<TemplateManagerProps> = {},
) {
  const { TemplateManager } = await import("../TemplateManager");
  return render(
    <ToastProvider>
      <AIAssistantProvider>
        <TemplateManager
          templateNames={props.templateNames ?? []}
          onTemplatesChanged={props.onTemplatesChanged ?? onTemplatesChanged}
          promotionSeed={props.promotionSeed as never}
        />
      </AIAssistantProvider>
    </ToastProvider>,
  );
}

export function buildTemplateRecord(
  options: {
    name?: string;
    isBuiltIn?: boolean;
    selectorName?: string;
    selector?: string;
  } = {},
) {
  return {
    name: options.name ?? "article",
    is_built_in: options.isBuiltIn ?? false,
    template: {
      name: options.name ?? "article",
      selectors: [
        {
          name: options.selectorName ?? "title",
          selector: options.selector ?? "h1",
          attr: "text",
        },
      ],
    },
  };
}

export function buildGetTemplateResponse(
  options: {
    name?: string;
    isBuiltIn?: boolean;
    selectorName?: string;
    selector?: string;
  } = {},
) {
  const record = buildTemplateRecord(options);
  return {
    data: record,
    error: undefined as never,
    request: new Request(`http://127.0.0.1:8741/v1/templates/${record.name}`),
    response: new Response(),
  };
}

export function buildNamedTemplatePromotionSeed() {
  return {
    kind: "template" as const,
    mode: "named-template" as const,
    source: {
      jobId: "job-123",
      jobKind: "scrape" as const,
      jobStatus: "succeeded" as const,
      label: "Source URL",
      value: "https://example.com/article",
    },
    suggestedName: "article-copy",
    previewUrl: "https://example.com/article",
    templateName: "article",
    carriedForward: ["The saved extraction rules from template “article”."],
    remainingDecisions: ["Review the duplicated template name before saving."],
    unsupportedCarryForward: [
      "Runtime execution settings and auth do not become part of the duplicated template automatically.",
    ],
  };
}

export function buildGuidedBlankPromotionSeed() {
  return {
    kind: "template" as const,
    mode: "guided-blank" as const,
    source: {
      jobId: "job-blank",
      jobKind: "scrape" as const,
      jobStatus: "succeeded" as const,
      label: "Source URL",
      value: "https://example.com",
    },
    suggestedName: "example-com-template",
    previewUrl: "https://example.com",
    carriedForward: [
      "A suggested template name and the verified source page for previewing a new draft.",
    ],
    remainingDecisions: [
      "Add at least one reusable selector rule with both a field name and CSS selector before save unlocks.",
    ],
    unsupportedCarryForward: [
      "This job did not include reusable template rules, so Spartan starts a guided blank draft instead of inventing a fake conversion.",
    ],
  };
}
