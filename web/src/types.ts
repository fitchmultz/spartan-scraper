/**
 * Purpose: Define shared type contracts for types.
 * Responsibilities: Export reusable TypeScript types and aliases that keep the surrounding feature consistent.
 * Scope: Type-level contracts only; runtime logic stays in implementation modules.
 * Usage: Import these types from adjacent feature, route, and test modules.
 * Invariants/Assumptions: The exported types should reflect the current source-of-truth contracts without introducing runtime side effects.
 */

import type {
  Job,
  CrawlState,
  ExtractOptions,
  ScrapeRequest,
  CrawlRequest,
  ResearchRequest,
  PipelineOptions,
  AuthOptions,
  FieldValue,
  InterceptedEntry,
} from "./api";

export type {
  Job,
  CrawlState,
  ExtractOptions,
  ScrapeRequest,
  CrawlRequest,
  ResearchRequest,
  PipelineOptions,
  AuthOptions,
};

export type JobEntry = Job;

export type EvidenceItem = {
  url: string;
  title: string;
  snippet: string;
  score: number;
  confidence?: number;
  citationUrl?: string;
  clusterId?: string;
  fields?: Record<string, FieldValue>;
};

export type ClusterItem = {
  id: string;
  label: string;
  confidence: number;
  evidence: EvidenceItem[];
};

export type CitationItem = {
  canonical: string;
  anchor?: string;
  url?: string;
};

export type ExtractedData = Record<string, unknown>;

export type NormalizedData = Record<string, unknown>;

export type CrawlResultItem = {
  url: string;
  status: number;
  title: string;
  text: string;
  links: string[];
  metadata?: Record<string, unknown>;
  extracted?: ExtractedData;
  normalized?: NormalizedData;
};

export type CrawlResultWithTraffic = CrawlResultItem & {
  interceptedData?: InterceptedEntry[];
};

export type AgenticResearchRoundItem = {
  round: number;
  goal?: string;
  focusAreas?: string[];
  selectedUrls?: string[];
  addedEvidenceCount?: number;
  reasoning?: string;
};

export type AgenticResearchItem = {
  status: string;
  instructions?: string;
  summary?: string;
  objective?: string;
  focusAreas?: string[];
  keyFindings?: string[];
  openQuestions?: string[];
  recommendedNextSteps?: string[];
  followUpUrls?: string[];
  rounds?: AgenticResearchRoundItem[];
  confidence?: number;
  route_id?: string;
  provider?: string;
  model?: string;
  cached?: boolean;
  error?: string;
};

export type ResearchResultItem = {
  query?: string;
  summary?: string;
  confidence?: number;
  evidence?: EvidenceItem[];
  clusters?: ClusterItem[];
  citations?: CitationItem[];
  agentic?: AgenticResearchItem;
};

export type ResultItem = CrawlResultItem | ResearchResultItem;
