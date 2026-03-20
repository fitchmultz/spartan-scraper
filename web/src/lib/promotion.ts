/**
 * Purpose: Build browser-safe promotion drafts from sanitized completed-job data.
 * Responsibilities: Derive destination-specific seeded drafts, summarize carried-forward vs. missing decisions, and produce chooser options for template, watch, and export-schedule promotion.
 * Scope: Pure promotion mapping only; routing, API fetching, and UI rendering stay in the application shell and route components.
 * Usage: Call from `/jobs/:id` route surfaces before navigating into `/templates`, `/automation/watches`, or `/automation/exports`.
 * Invariants/Assumptions: Inputs are sanitized `Job` objects, watch promotion is scrape-first in v1, export schedules automate future matching completed jobs rather than rerunning a source job, and template promotion only treats real reusable extraction structure as high-confidence carry-forward.
 */

import type { Job, Template } from "../api";
import { defaultFormData as defaultExportScheduleFormData } from "./export-schedule-utils";
import { defaultFormData as defaultWatchFormData } from "./watch-utils";
import type {
  ExportSchedulePromotionSeed,
  PromotionOption,
  PromotionSourceContext,
  TemplatePromotionSeed,
  WatchPromotionSeed,
} from "../types/promotion";

function asRecord(value: unknown): Record<string, unknown> | null {
  return value !== null && typeof value === "object" && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : null;
}

function asString(value: unknown): string | undefined {
  return typeof value === "string" && value.trim().length > 0
    ? value.trim()
    : undefined;
}

function asBoolean(value: unknown): boolean | undefined {
  return typeof value === "boolean" ? value : undefined;
}

function asNumber(value: unknown): number | undefined {
  return typeof value === "number" && Number.isFinite(value)
    ? value
    : undefined;
}

function asStringArray(value: unknown): string[] {
  return Array.isArray(value)
    ? value.filter((item): item is string => typeof item === "string")
    : [];
}

function slugify(value: string): string {
  const normalized = value
    .toLowerCase()
    .replace(/^https?:\/\//, "")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 48);

  return normalized || "verified-job";
}

function executionSpec(job: Job): Record<string, unknown> | null {
  return asRecord(asRecord(job.spec)?.execution);
}

function primaryTarget(job: Job): { label: string; value: string } {
  const spec = asRecord(job.spec);
  const url = asString(spec?.url);
  if (url) {
    return { label: "Source URL", value: url };
  }

  const query = asString(spec?.query);
  if (query) {
    return { label: "Research query", value: query };
  }

  const firstUrl = asStringArray(spec?.urls)[0];
  if (firstUrl) {
    return { label: "Primary URL", value: firstUrl };
  }

  return { label: "Source job", value: job.id };
}

function suggestedBase(job: Job): string {
  const target = primaryTarget(job).value;
  return slugify(target);
}

function previewUrl(job: Job): string | undefined {
  const spec = asRecord(job.spec);
  return asString(spec?.url) ?? asStringArray(spec?.urls)[0];
}

function templateFromUnknown(value: unknown): Template | undefined {
  const template = asRecord(value);
  if (!template) {
    return undefined;
  }

  const selectors = Array.isArray(template.selectors)
    ? template.selectors
        .map((item) => {
          const rule = asRecord(item);
          if (!rule) {
            return null;
          }
          return {
            name: asString(rule.name),
            selector: asString(rule.selector),
            attr: asString(rule.attr),
            all: asBoolean(rule.all),
            join: asString(rule.join),
            trim: asBoolean(rule.trim),
            required: asBoolean(rule.required),
          };
        })
        .filter((item): item is NonNullable<typeof item> => item !== null)
    : undefined;

  const jsonld = Array.isArray(template.jsonld)
    ? template.jsonld
        .map((item) => {
          const rule = asRecord(item);
          if (!rule) {
            return null;
          }
          return {
            name: asString(rule.name),
            type: asString(rule.type),
            path: asString(rule.path),
            all: asBoolean(rule.all),
            required: asBoolean(rule.required),
          };
        })
        .filter((item): item is NonNullable<typeof item> => item !== null)
    : undefined;

  const regex = Array.isArray(template.regex)
    ? template.regex
        .map((item) => {
          const rule = asRecord(item);
          if (!rule) {
            return null;
          }
          return {
            name: asString(rule.name),
            pattern: asString(rule.pattern),
            group: asNumber(rule.group),
            all: asBoolean(rule.all),
            source: asString(rule.source) as
              | "text"
              | "html"
              | "url"
              | undefined,
            required: asBoolean(rule.required),
          };
        })
        .filter((item): item is NonNullable<typeof item> => item !== null)
    : undefined;

  const normalizeRecord = asRecord(template.normalize);
  const normalize = normalizeRecord
    ? {
        titleField: asString(normalizeRecord.titleField),
        descriptionField: asString(normalizeRecord.descriptionField),
        textField: asString(normalizeRecord.textField),
        metaFields: asRecord(normalizeRecord.metaFields) as
          | Record<string, string>
          | undefined,
      }
    : undefined;

  if (!selectors?.length && !jsonld?.length && !regex?.length && !normalize) {
    return undefined;
  }

  return {
    name: asString(template.name),
    selectors,
    jsonld,
    regex,
    normalize,
  };
}

export function buildPromotionSourceContext(job: Job): PromotionSourceContext {
  const target = primaryTarget(job);
  return {
    jobId: job.id,
    jobKind: job.kind,
    jobStatus: job.status,
    label: target.label,
    value: target.value,
    finishedAt: job.finishedAt,
  };
}

export function buildTemplatePromotionSeed(job: Job): TemplatePromotionSeed {
  const exec = executionSpec(job);
  const extract = asRecord(exec?.extract);
  const namedTemplate = asString(extract?.template);
  const inlineTemplate = templateFromUnknown(extract?.inline);
  const source = buildPromotionSourceContext(job);
  const previewTarget = previewUrl(job);
  const fallbackName = `${suggestedBase(job)}-template`;

  if (inlineTemplate) {
    return {
      kind: "template",
      mode: "inline-template",
      source,
      previewUrl: previewTarget,
      suggestedName: asString(inlineTemplate.name) ?? fallbackName,
      template: {
        ...inlineTemplate,
        name: asString(inlineTemplate.name) ?? fallbackName,
      },
      carriedForward: [
        "Inline selector, JSON-LD, regex, and normalization rules from the successful job.",
        "Source page context so preview and debugging stay anchored to the verified run.",
      ],
      remainingDecisions: [
        "Review the suggested template name and trim any one-off extraction rules.",
        "Save only the reusable extraction behavior you want to keep in the library.",
      ],
      unsupportedCarryForward: [
        "Runtime-only execution settings stay outside the saved template unless you re-enter them intentionally.",
      ],
    };
  }

  if (namedTemplate) {
    return {
      kind: "template",
      mode: "named-template",
      source,
      previewUrl: previewTarget,
      suggestedName: `${namedTemplate}-copy`,
      templateName: namedTemplate,
      carriedForward: [
        `The saved extraction rules from template “${namedTemplate}”.`,
        "Source job context so the duplicated draft stays tied to the successful run.",
      ],
      remainingDecisions: [
        "Review the duplicated template name before saving.",
        "Decide whether to keep or simplify any rules that were only useful for the original run.",
      ],
      unsupportedCarryForward: [
        "Runtime execution settings and auth do not become part of the duplicated template automatically.",
      ],
    };
  }

  return {
    kind: "template",
    mode: "guided-blank",
    source,
    previewUrl: previewTarget,
    suggestedName: fallbackName,
    carriedForward: [
      "A suggested template name and the verified source page for previewing a new draft.",
      "Source-job lineage so the workspace still shows where this draft came from.",
    ],
    remainingDecisions: [
      "Define the reusable selector or structured extraction rules you actually want to save.",
      "Review the blank draft before saving so runtime-only extraction logic is not mistaken for a reusable template.",
    ],
    unsupportedCarryForward: [
      "This job did not include reusable template rules, so Spartan starts a guided blank draft instead of inventing a fake conversion.",
    ],
  };
}

export function buildWatchPromotionSeed(job: Job): WatchPromotionSeed {
  const spec = asRecord(job.spec);
  const exec = executionSpec(job);
  const screenshot = asRecord(exec?.screenshot);
  const url = asString(spec?.url) ?? "";
  const source = buildPromotionSourceContext(job);
  const unsupportedCarryForward: string[] = [];

  if (asString(exec?.authProfile) || exec?.auth !== undefined) {
    unsupportedCarryForward.push(
      "Authentication settings are not carried into watches in this cut.",
    );
  }
  if (asRecord(exec?.pipeline)) {
    unsupportedCarryForward.push(
      "Pipeline JavaScript is not carried into watches in this cut.",
    );
  }
  if (asRecord(exec?.extract)) {
    unsupportedCarryForward.push(
      "Extraction templates and extract-specific rules are not part of the watch contract.",
    );
  }

  const eligible = job.kind === "scrape" && Boolean(url);
  const eligibilityMessage = eligible
    ? undefined
    : job.kind !== "scrape"
      ? "Watches are scrape-first in this cut because they model single-target monitoring, not full crawl or research replay."
      : "This job does not expose a promotable single target URL for watch creation.";

  return {
    kind: "watch",
    source,
    eligible,
    eligibilityMessage,
    formData: {
      ...defaultWatchFormData,
      url,
      headless: asBoolean(exec?.headless) ?? defaultWatchFormData.headless,
      usePlaywright:
        asBoolean(exec?.playwright) ?? defaultWatchFormData.usePlaywright,
      screenshotEnabled:
        asBoolean(screenshot?.enabled) ??
        defaultWatchFormData.screenshotEnabled,
      screenshotFullPage:
        asBoolean(screenshot?.fullPage) ??
        defaultWatchFormData.screenshotFullPage,
      screenshotFormat:
        (asString(screenshot?.format) as "png" | "jpeg" | undefined) ??
        defaultWatchFormData.screenshotFormat,
      visualDiffThreshold:
        typeof screenshot?.enabled === "boolean" && screenshot.enabled
          ? defaultWatchFormData.visualDiffThreshold
          : defaultWatchFormData.visualDiffThreshold,
    },
    carriedForward: [
      "The verified target URL from the successful job.",
      "Browser runtime choices such as headless mode, Playwright, and screenshot capture when they map cleanly to watches.",
    ],
    remainingDecisions: [
      "Set the monitoring interval, diff style, notifications, and any selector targeting before saving.",
      "Review whether visual change detection and sensitivity match the page you actually want to monitor over time.",
    ],
    unsupportedCarryForward,
  };
}

export function buildExportSchedulePromotionSeed(
  job: Job,
  preferredFormat?: "json" | "jsonl" | "md" | "csv" | "xlsx",
): ExportSchedulePromotionSeed {
  const source = buildPromotionSourceContext(job);
  const base = suggestedBase(job).replace(/-template$/, "");

  return {
    kind: "export-schedule",
    source,
    seededFormat: preferredFormat,
    formData: {
      ...defaultExportScheduleFormData,
      name: `${base}-verified-export`,
      filterJobKinds: [job.kind],
      filterJobStatus: ["succeeded"],
      filterHasResults: true,
      format: preferredFormat ?? defaultExportScheduleFormData.format,
    },
    carriedForward: [
      `A ${job.kind} filter scoped to future successful jobs like this one.`,
      preferredFormat
        ? `The ${preferredFormat.toUpperCase()} export format from the latest verified export.`
        : "A starting export policy draft anchored to the successful source job.",
    ],
    remainingDecisions: [
      preferredFormat
        ? "Confirm the destination, retry policy, and any extra filter constraints before saving."
        : "Choose the export format, destination, and any extra filter constraints before saving.",
      "Review whether you want only this job kind or a broader matching policy for future jobs.",
    ],
    unsupportedCarryForward: [
      "Export schedules automate future matching completed jobs; they do not rerun this source job on a cadence.",
    ],
  };
}

export function buildPromotionOptions(
  job: Job,
  preferredExportFormat?: "json" | "jsonl" | "md" | "csv" | "xlsx",
): PromotionOption[] {
  const templateSeed = buildTemplatePromotionSeed(job);
  const watchSeed = buildWatchPromotionSeed(job);
  const exportSeed = buildExportSchedulePromotionSeed(
    job,
    preferredExportFormat,
  );

  return [
    {
      destination: "template",
      title: "Save as Template",
      description:
        templateSeed.mode === "guided-blank"
          ? "Start a source-aware template draft when this job proved the page is worth extracting again."
          : "Preserve reusable extraction rules from this successful job in the template workspace.",
      actionLabel: "Open template draft",
      eligible: true,
      seed: templateSeed,
    },
    {
      destination: "watch",
      title: "Create Watch",
      description:
        "Monitor the same verified target over time with an explicit single-page watch draft.",
      actionLabel: "Open watch draft",
      eligible: watchSeed.eligible,
      eligibilityMessage: watchSeed.eligibilityMessage,
      seed: watchSeed,
    },
    {
      destination: "export-schedule",
      title: "Create Export Schedule",
      description:
        "Automatically export future matching completed jobs without rebuilding the export policy from scratch.",
      actionLabel: "Open export draft",
      eligible: true,
      seed: exportSeed,
    },
  ];
}
