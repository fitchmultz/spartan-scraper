/**
 * Purpose: Compute reusable operator-facing diffs and summaries for crawl, research, and AI authoring surfaces.
 * Responsibilities: Compare supported result shapes, expose field-level changes, summarize AI candidates, and signal when raw JSON fallback is safer than a partial summary.
 * Scope: Shared web diff logic only.
 * Usage: Import from result diff views and AI authoring components instead of rebuilding ad hoc comparison logic.
 * Invariants/Assumptions: Equality checks must stay deterministic, unchanged fields should be omitted from comparison summaries, and unsupported AI candidate changes should fall back to raw JSON by default.
 */
import type { JsTargetScript, RenderProfile } from "../api";
import type {
  CrawlResultItem,
  ResearchResultItem,
  ResultItem,
  EvidenceItem,
  ClusterItem,
  CitationItem,
} from "../types";

/**
 * Represents a change to a specific field.
 */
export interface FieldChange {
  /** Field name that changed */
  field: string;
  /** Previous value */
  oldValue: unknown;
  /** New value */
  newValue: unknown;
  /** Optional canonical field path */
  path?: string;
}

/**
 * Summary of a supported latest-only candidate field.
 */
export interface CandidateFieldSummary {
  /** Operator-facing label */
  label: string;
  /** Canonical field path */
  path: string;
  /** Latest field value */
  value: unknown;
}

/**
 * Summary of an AI candidate comparison.
 */
export interface CandidateDiffSummary {
  /** Changed supported fields between the previous and latest candidate */
  changes: FieldChange[];
  /** Latest candidate highlights for single-candidate views */
  latestFields: CandidateFieldSummary[];
  /** Whether raw JSON should be shown by default */
  shouldShowRawJsonByDefault: boolean;
  /** Human-readable explanation for raw JSON fallback */
  rawJsonReason?: string;
}

/**
 * Result of comparing two sets of crawl results.
 */
export interface CrawlDiffResult {
  /** Items present in 'after' but not in 'before' */
  added: CrawlResultItem[];
  /** Items present in 'before' but not in 'after' */
  removed: CrawlResultItem[];
  /** Items present in both with field-level changes */
  modified: {
    url: string;
    before: CrawlResultItem;
    after: CrawlResultItem;
    changes: FieldChange[];
  }[];
  /** Items that are identical in both sets */
  unchanged: CrawlResultItem[];
}

/**
 * Result of comparing two research results.
 */
export interface ResearchDiffResult {
  /** Summary changes */
  summaryChanges?: {
    oldValue: string | undefined;
    newValue: string | undefined;
  };
  /** Confidence score changes */
  confidenceChanges?: {
    oldValue: number | undefined;
    newValue: number | undefined;
  };
  /** Evidence items added */
  evidenceAdded: EvidenceItem[];
  /** Evidence items removed */
  evidenceRemoved: EvidenceItem[];
  /** Evidence items modified (by URL match) */
  evidenceModified: {
    url: string;
    before: EvidenceItem;
    after: EvidenceItem;
    changes: FieldChange[];
  }[];
  /** Evidence unchanged */
  evidenceUnchanged: EvidenceItem[];
  /** Clusters added */
  clustersAdded: ClusterItem[];
  /** Clusters removed */
  clustersRemoved: ClusterItem[];
  /** Clusters modified (by ID match) */
  clustersModified: {
    id: string;
    before: ClusterItem;
    after: ClusterItem;
    changes: FieldChange[];
  }[];
  /** Citations added */
  citationsAdded: CitationItem[];
  /** Citations removed */
  citationsRemoved: CitationItem[];
}

/**
 * Compute a unique key for a crawl result item.
 * Uses the URL as the primary identifier.
 *
 * @param result - Crawl result item
 * @returns Unique key string
 */
export function computeCrawlResultKey(result: CrawlResultItem): string {
  return result.url;
}

/**
 * Compute a unique key for an evidence item.
 *
 * @param evidence - Evidence item
 * @returns Unique key string
 */
export function computeEvidenceKey(evidence: EvidenceItem): string {
  return evidence.url;
}

/**
 * Compute a unique key for a cluster item.
 *
 * @param cluster - Cluster item
 * @returns Unique key string
 */
export function computeClusterKey(cluster: ClusterItem): string {
  return cluster.id;
}

/**
 * Compute a unique key for a citation item.
 *
 * @param citation - Citation item
 * @returns Unique key string
 */
export function computeCitationKey(citation: CitationItem): string {
  return citation.canonical || citation.url || "";
}

/**
 * Compare two values for equality.
 * Handles primitives, arrays, and objects.
 *
 * @param a - First value
 * @param b - Second value
 * @returns True if values are equal
 */
export function deepEqual(a: unknown, b: unknown): boolean {
  if (a === b) return true;
  if (typeof a !== typeof b) return false;
  if (a == null || b == null) return a === b;

  if (Array.isArray(a) && Array.isArray(b)) {
    if (a.length !== b.length) return false;
    return a.every((item, index) => deepEqual(item, b[index]));
  }

  if (typeof a === "object" && typeof b === "object") {
    const aObj = a as Record<string, unknown>;
    const bObj = b as Record<string, unknown>;
    const aKeys = Object.keys(aObj);
    const bKeys = Object.keys(bObj);
    if (aKeys.length !== bKeys.length) return false;
    return aKeys.every((key) => deepEqual(aObj[key], bObj[key]));
  }

  return false;
}

/**
 * Compute field-level changes between two crawl result items.
 *
 * @param before - Previous crawl result
 * @param after - Current crawl result
 * @returns Array of field changes
 */
export function computeCrawlFieldChanges(
  before: CrawlResultItem,
  after: CrawlResultItem,
): FieldChange[] {
  const changes: FieldChange[] = [];
  const fields: (keyof CrawlResultItem)[] = [
    "status",
    "title",
    "text",
    "links",
    "metadata",
    "extracted",
    "normalized",
  ];

  for (const field of fields) {
    const oldValue = before[field];
    const newValue = after[field];

    if (!deepEqual(oldValue, newValue)) {
      changes.push({ field, oldValue, newValue });
    }
  }

  return changes;
}

/**
 * Compute field-level changes between two evidence items.
 *
 * @param before - Previous evidence
 * @param after - Current evidence
 * @returns Array of field changes
 */
export function computeEvidenceFieldChanges(
  before: EvidenceItem,
  after: EvidenceItem,
): FieldChange[] {
  const changes: FieldChange[] = [];
  const fields: (keyof EvidenceItem)[] = [
    "title",
    "snippet",
    "score",
    "confidence",
    "citationUrl",
    "clusterId",
  ];

  for (const field of fields) {
    const oldValue = before[field];
    const newValue = after[field];

    if (!deepEqual(oldValue, newValue)) {
      changes.push({ field, oldValue, newValue });
    }
  }

  return changes;
}

/**
 * Compute field-level changes between two cluster items.
 *
 * @param before - Previous cluster
 * @param after - Current cluster
 * @returns Array of field changes
 */
export function computeClusterFieldChanges(
  before: ClusterItem,
  after: ClusterItem,
): FieldChange[] {
  const changes: FieldChange[] = [];
  const fields: (keyof ClusterItem)[] = ["label", "confidence", "evidence"];

  for (const field of fields) {
    const oldValue = before[field];
    const newValue = after[field];

    if (!deepEqual(oldValue, newValue)) {
      changes.push({ field, oldValue, newValue });
    }
  }

  return changes;
}

/**
 * Compare two sets of crawl results.
 *
 * @param before - Previous crawl results
 * @param after - Current crawl results
 * @returns Detailed diff result
 */
export function diffCrawlResults(
  before: CrawlResultItem[],
  after: CrawlResultItem[],
): CrawlDiffResult {
  const beforeMap = new Map<string, CrawlResultItem>();
  const afterMap = new Map<string, CrawlResultItem>();

  for (const item of before) {
    beforeMap.set(computeCrawlResultKey(item), item);
  }

  for (const item of after) {
    afterMap.set(computeCrawlResultKey(item), item);
  }

  const added: CrawlResultItem[] = [];
  const removed: CrawlResultItem[] = [];
  const modified: CrawlDiffResult["modified"] = [];
  const unchanged: CrawlResultItem[] = [];

  // Find added and modified items
  for (const [key, afterItem] of afterMap) {
    const beforeItem = beforeMap.get(key);
    if (!beforeItem) {
      added.push(afterItem);
    } else {
      const changes = computeCrawlFieldChanges(beforeItem, afterItem);
      if (changes.length > 0) {
        modified.push({
          url: key,
          before: beforeItem,
          after: afterItem,
          changes,
        });
      } else {
        unchanged.push(afterItem);
      }
    }
  }

  // Find removed items
  for (const [key, beforeItem] of beforeMap) {
    if (!afterMap.has(key)) {
      removed.push(beforeItem);
    }
  }

  return { added, removed, modified, unchanged };
}

/**
 * Compare two research results.
 *
 * @param before - Previous research result
 * @param after - Current research result
 * @returns Detailed diff result
 */
export function diffResearchResults(
  before: ResearchResultItem,
  after: ResearchResultItem,
): ResearchDiffResult {
  const result: ResearchDiffResult = {
    evidenceAdded: [],
    evidenceRemoved: [],
    evidenceModified: [],
    evidenceUnchanged: [],
    clustersAdded: [],
    clustersRemoved: [],
    clustersModified: [],
    citationsAdded: [],
    citationsRemoved: [],
  };

  // Compare summary
  if (before.summary !== after.summary) {
    result.summaryChanges = {
      oldValue: before.summary,
      newValue: after.summary,
    };
  }

  // Compare confidence
  if (before.confidence !== after.confidence) {
    result.confidenceChanges = {
      oldValue: before.confidence,
      newValue: after.confidence,
    };
  }

  // Compare evidence
  const beforeEvidence = before.evidence || [];
  const afterEvidence = after.evidence || [];
  const beforeEvidenceMap = new Map(
    beforeEvidence.map((e) => [computeEvidenceKey(e), e]),
  );
  const afterEvidenceMap = new Map(
    afterEvidence.map((e) => [computeEvidenceKey(e), e]),
  );

  for (const [key, afterItem] of afterEvidenceMap) {
    const beforeItem = beforeEvidenceMap.get(key);
    if (!beforeItem) {
      result.evidenceAdded.push(afterItem);
    } else {
      const changes = computeEvidenceFieldChanges(beforeItem, afterItem);
      if (changes.length > 0) {
        result.evidenceModified.push({
          url: key,
          before: beforeItem,
          after: afterItem,
          changes,
        });
      } else {
        result.evidenceUnchanged.push(afterItem);
      }
    }
  }

  for (const [key, beforeItem] of beforeEvidenceMap) {
    if (!afterEvidenceMap.has(key)) {
      result.evidenceRemoved.push(beforeItem);
    }
  }

  // Compare clusters
  const beforeClusters = before.clusters || [];
  const afterClusters = after.clusters || [];
  const beforeClusterMap = new Map(
    beforeClusters.map((c) => [computeClusterKey(c), c]),
  );
  const afterClusterMap = new Map(
    afterClusters.map((c) => [computeClusterKey(c), c]),
  );

  for (const [key, afterItem] of afterClusterMap) {
    const beforeItem = beforeClusterMap.get(key);
    if (!beforeItem) {
      result.clustersAdded.push(afterItem);
    } else {
      const changes = computeClusterFieldChanges(beforeItem, afterItem);
      if (changes.length > 0) {
        result.clustersModified.push({
          id: key,
          before: beforeItem,
          after: afterItem,
          changes,
        });
      }
    }
  }

  for (const [key, beforeItem] of beforeClusterMap) {
    if (!afterClusterMap.has(key)) {
      result.clustersRemoved.push(beforeItem);
    }
  }

  // Compare citations
  const beforeCitations = before.citations || [];
  const afterCitations = after.citations || [];
  const beforeCitationSet = new Set(beforeCitations.map(computeCitationKey));
  const afterCitationSet = new Set(afterCitations.map(computeCitationKey));

  for (const citation of afterCitations) {
    const key = computeCitationKey(citation);
    if (!beforeCitationSet.has(key)) {
      result.citationsAdded.push(citation);
    }
  }

  for (const citation of beforeCitations) {
    const key = computeCitationKey(citation);
    if (!afterCitationSet.has(key)) {
      result.citationsRemoved.push(citation);
    }
  }

  return result;
}

/**
 * Type guard for crawl result items.
 *
 * @param item - Result item to check
 * @returns True if item is a crawl result
 */
function isCrawlResultItem(item: ResultItem): item is CrawlResultItem {
  return "url" in item && "status" in item;
}

/**
 * Type guard for research result items.
 *
 * @param item - Result item to check
 * @returns True if item is a research result
 */
function isResearchResultItem(item: ResultItem): item is ResearchResultItem {
  return "summary" in item || "evidence" in item;
}

/**
 * Compare two sets of results (crawl or research).
 * Automatically detects result type and delegates to appropriate diff function.
 *
 * @param before - Previous results
 * @param after - Current results
 * @returns Diff result (crawl or research type)
 */
export function diffResults(
  before: ResultItem[],
  after: ResultItem[],
): CrawlDiffResult | ResearchDiffResult | null {
  if (before.length === 0 && after.length === 0) {
    return null;
  }

  // Detect result type from first non-null item
  const sampleBefore = before.find((item) => item != null);
  const sampleAfter = after.find((item) => item != null);
  const sample = sampleBefore || sampleAfter;

  if (!sample) return null;

  if (isCrawlResultItem(sample)) {
    return diffCrawlResults(
      before.filter(isCrawlResultItem),
      after.filter(isCrawlResultItem),
    );
  }

  if (isResearchResultItem(sample)) {
    // Research results are typically single items
    const beforeItem = (before.find(isResearchResultItem) ||
      {}) as ResearchResultItem;
    const afterItem = (after.find(isResearchResultItem) ||
      {}) as ResearchResultItem;
    return diffResearchResults(beforeItem, afterItem);
  }

  return null;
}

/**
 * Get summary statistics for a crawl diff.
 *
 * @param diff - Crawl diff result
 * @returns Summary counts
 */
export function getCrawlDiffStats(diff: CrawlDiffResult): {
  added: number;
  removed: number;
  modified: number;
  unchanged: number;
  total: number;
} {
  return {
    added: diff.added.length,
    removed: diff.removed.length,
    modified: diff.modified.length,
    unchanged: diff.unchanged.length,
    total:
      diff.added.length +
      diff.removed.length +
      diff.modified.length +
      diff.unchanged.length,
  };
}

/**
 * Get summary statistics for a research diff.
 *
 * @param diff - Research diff result
 * @returns Summary counts
 */
export function getResearchDiffStats(diff: ResearchDiffResult): {
  evidenceAdded: number;
  evidenceRemoved: number;
  evidenceModified: number;
  clustersAdded: number;
  clustersRemoved: number;
  clustersModified: number;
  citationsAdded: number;
  citationsRemoved: number;
  summaryChanged: boolean;
  confidenceChanged: boolean;
} {
  return {
    evidenceAdded: diff.evidenceAdded.length,
    evidenceRemoved: diff.evidenceRemoved.length,
    evidenceModified: diff.evidenceModified.length,
    clustersAdded: diff.clustersAdded.length,
    clustersRemoved: diff.clustersRemoved.length,
    clustersModified: diff.clustersModified.length,
    citationsAdded: diff.citationsAdded.length,
    citationsRemoved: diff.citationsRemoved.length,
    summaryChanged: !!diff.summaryChanges,
    confidenceChanged: !!diff.confidenceChanges,
  };
}

type CandidateFieldDescriptor<T> = {
  path: string;
  label: string;
  getValue: (artifact: T) => unknown;
};

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === "object" && !Array.isArray(value);
}

function isDisplayEmpty(value: unknown): boolean {
  if (value === undefined || value === null || value === "") {
    return true;
  }
  if (Array.isArray(value)) {
    return value.length === 0;
  }
  return false;
}

function flattenComparable(
  value: unknown,
  prefix = "",
  output: Record<string, unknown> = {},
): Record<string, unknown> {
  if (!isPlainObject(value)) {
    if (prefix) {
      output[prefix] = value;
    }
    return output;
  }

  for (const [key, nestedValue] of Object.entries(value)) {
    const nextPrefix = prefix ? `${prefix}.${key}` : key;
    if (Array.isArray(nestedValue) || !isPlainObject(nestedValue)) {
      output[nextPrefix] = nestedValue;
      continue;
    }
    flattenComparable(nestedValue, nextPrefix, output);
  }

  return output;
}

function computeCandidateFieldChanges<T>(
  before: T,
  after: T,
  descriptors: readonly CandidateFieldDescriptor<T>[],
): FieldChange[] {
  return descriptors.flatMap((descriptor) => {
    const oldValue = descriptor.getValue(before);
    const newValue = descriptor.getValue(after);
    if (deepEqual(oldValue, newValue)) {
      return [];
    }

    return [
      {
        field: descriptor.label,
        path: descriptor.path,
        oldValue,
        newValue,
      },
    ];
  });
}

function buildLatestFieldSummary<T>(
  artifact: T,
  descriptors: readonly CandidateFieldDescriptor<T>[],
): CandidateFieldSummary[] {
  return descriptors
    .map((descriptor) => ({
      label: descriptor.label,
      path: descriptor.path,
      value: descriptor.getValue(artifact),
    }))
    .filter((entry) => !isDisplayEmpty(entry.value));
}

function findUnsupportedChangedPaths<T>(
  before: T,
  after: T,
  descriptors: readonly CandidateFieldDescriptor<T>[],
): string[] {
  const supportedPaths = new Set(
    descriptors.map((descriptor) => descriptor.path),
  );
  const beforeFlat = flattenComparable(before);
  const afterFlat = flattenComparable(after);
  const allPaths = new Set([
    ...Object.keys(beforeFlat),
    ...Object.keys(afterFlat),
  ]);

  return [...allPaths]
    .filter((path) => {
      if (supportedPaths.has(path)) {
        return false;
      }
      return !deepEqual(beforeFlat[path], afterFlat[path]);
    })
    .sort();
}

const renderProfileFieldDescriptors: readonly CandidateFieldDescriptor<RenderProfile>[] =
  [
    {
      path: "name",
      label: "Profile name",
      getValue: (profile) => profile.name,
    },
    {
      path: "hostPatterns",
      label: "Host patterns",
      getValue: (profile) => profile.hostPatterns,
    },
    {
      path: "wait.mode",
      label: "Wait mode",
      getValue: (profile) => profile.wait?.mode,
    },
    {
      path: "wait.selector",
      label: "Wait selector",
      getValue: (profile) => profile.wait?.selector,
    },
    {
      path: "forceEngine",
      label: "Engine override",
      getValue: (profile) => profile.forceEngine,
    },
    {
      path: "preferHeadless",
      label: "Prefer headless",
      getValue: (profile) => profile.preferHeadless,
    },
    {
      path: "neverHeadless",
      label: "Never headless",
      getValue: (profile) => profile.neverHeadless,
    },
    {
      path: "block.resourceTypes",
      label: "Blocked resource types",
      getValue: (profile) => profile.block?.resourceTypes,
    },
    {
      path: "block.urlPatterns",
      label: "Blocked URL patterns",
      getValue: (profile) => profile.block?.urlPatterns,
    },
    {
      path: "timeouts.maxRenderMs",
      label: "Max render timeout",
      getValue: (profile) => profile.timeouts?.maxRenderMs,
    },
    {
      path: "timeouts.scriptEvalMs",
      label: "Script evaluation timeout",
      getValue: (profile) => profile.timeouts?.scriptEvalMs,
    },
    {
      path: "timeouts.navigationMs",
      label: "Navigation timeout",
      getValue: (profile) => profile.timeouts?.navigationMs,
    },
    {
      path: "screenshot.enabled",
      label: "Screenshot capture",
      getValue: (profile) => profile.screenshot?.enabled,
    },
    {
      path: "screenshot.fullPage",
      label: "Full-page screenshot",
      getValue: (profile) => profile.screenshot?.fullPage,
    },
    {
      path: "screenshot.format",
      label: "Screenshot format",
      getValue: (profile) => profile.screenshot?.format,
    },
    {
      path: "screenshot.quality",
      label: "Screenshot quality",
      getValue: (profile) => profile.screenshot?.quality,
    },
    {
      path: "screenshot.width",
      label: "Screenshot width",
      getValue: (profile) => profile.screenshot?.width,
    },
    {
      path: "screenshot.height",
      label: "Screenshot height",
      getValue: (profile) => profile.screenshot?.height,
    },
  ];

const pipelineScriptFieldDescriptors: readonly CandidateFieldDescriptor<JsTargetScript>[] =
  [
    {
      path: "name",
      label: "Script name",
      getValue: (script) => script.name,
    },
    {
      path: "hostPatterns",
      label: "Host patterns",
      getValue: (script) => script.hostPatterns,
    },
    {
      path: "engine",
      label: "Browser engine",
      getValue: (script) => script.engine,
    },
    {
      path: "selectors",
      label: "Wait selectors",
      getValue: (script) => script.selectors,
    },
    {
      path: "preNav",
      label: "Pre-navigation logic",
      getValue: (script) => script.preNav,
    },
    {
      path: "postNav",
      label: "Post-navigation logic",
      getValue: (script) => script.postNav,
    },
  ];

function buildCandidateDiffSummary<T>(
  previousArtifact: T | null | undefined,
  latestArtifact: T | null | undefined,
  descriptors: readonly CandidateFieldDescriptor<T>[],
  artifactLabel: string,
): CandidateDiffSummary {
  if (!latestArtifact || !isPlainObject(latestArtifact)) {
    return {
      changes: [],
      latestFields: [],
      shouldShowRawJsonByDefault: true,
      rawJsonReason: `Showing raw JSON because the latest ${artifactLabel} could not be summarized safely.`,
    };
  }

  const latestFields = buildLatestFieldSummary(latestArtifact, descriptors);

  if (!previousArtifact) {
    return {
      changes: [],
      latestFields,
      shouldShowRawJsonByDefault: false,
    };
  }

  if (!isPlainObject(previousArtifact)) {
    return {
      changes: [],
      latestFields,
      shouldShowRawJsonByDefault: true,
      rawJsonReason: `Showing raw JSON because the previous ${artifactLabel} could not be summarized safely.`,
    };
  }

  const changes = computeCandidateFieldChanges(
    previousArtifact,
    latestArtifact,
    descriptors,
  );
  const unsupportedChangedPaths = findUnsupportedChangedPaths(
    previousArtifact,
    latestArtifact,
    descriptors,
  );

  return {
    changes,
    latestFields,
    shouldShowRawJsonByDefault: unsupportedChangedPaths.length > 0,
    rawJsonReason:
      unsupportedChangedPaths.length > 0
        ? `Showing raw JSON because ${artifactLabel} changed in unsupported fields: ${unsupportedChangedPaths.join(", ")}.`
        : undefined,
  };
}

export function summarizeRenderProfileCandidateDiff(
  previousArtifact: RenderProfile | null | undefined,
  latestArtifact: RenderProfile | null | undefined,
): CandidateDiffSummary {
  return buildCandidateDiffSummary(
    previousArtifact,
    latestArtifact,
    renderProfileFieldDescriptors,
    "the render profile",
  );
}

export function summarizePipelineScriptCandidateDiff(
  previousArtifact: JsTargetScript | null | undefined,
  latestArtifact: JsTargetScript | null | undefined,
): CandidateDiffSummary {
  return buildCandidateDiffSummary(
    previousArtifact,
    latestArtifact,
    pipelineScriptFieldDescriptors,
    "the pipeline JS script",
  );
}
