/**
 * Purpose: Define shared promotion-flow types for routing, seeded drafts, and visible source-job context.
 * Responsibilities: Describe source-job context, destination-specific promotion seeds, and chooser options reused across results and destination workspaces.
 * Scope: Type contracts only; promotion mapping logic lives in `../lib/promotion` and rendering lives in promotion-aware components.
 * Usage: Import from web route containers, managers, and helpers that participate in job-to-automation promotion.
 * Invariants/Assumptions: Promotion seeds only contain browser-safe sanitized job data, destination routes remain canonical ownership surfaces, and unsupported carry-forward stays explicit in the seed metadata.
 */

import type { Job, Template } from "../api";
import type { ExportScheduleFormData } from "./export-schedule";
import type { WatchFormData } from "./watch";

export type PromotionDestination = "template" | "watch" | "export-schedule";

export interface PromotionSourceContext {
  jobId: string;
  jobKind: Job["kind"];
  jobStatus: Job["status"];
  label: string;
  value: string;
  finishedAt?: string;
}

export interface BasePromotionSeed {
  kind: PromotionDestination;
  source: PromotionSourceContext;
  carriedForward: string[];
  remainingDecisions: string[];
  unsupportedCarryForward: string[];
}

export interface TemplatePromotionSeed extends BasePromotionSeed {
  kind: "template";
  mode: "named-template" | "inline-template" | "guided-blank";
  suggestedName: string;
  previewUrl?: string;
  templateName?: string;
  template?: Template;
}

export interface WatchPromotionSeed extends BasePromotionSeed {
  kind: "watch";
  eligible: boolean;
  eligibilityMessage?: string;
  formData: WatchFormData;
}

export interface ExportSchedulePromotionSeed extends BasePromotionSeed {
  kind: "export-schedule";
  formData: ExportScheduleFormData;
  seededFormat?: ExportScheduleFormData["format"];
}

export type PromotionSeed =
  | TemplatePromotionSeed
  | WatchPromotionSeed
  | ExportSchedulePromotionSeed;

export interface PromotionOption {
  destination: PromotionDestination;
  title: string;
  description: string;
  actionLabel: string;
  eligible: boolean;
  eligibilityMessage?: string;
  seed: PromotionSeed;
}
