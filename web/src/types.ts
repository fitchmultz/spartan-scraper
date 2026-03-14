/**
 * Shared Type Definitions
 *
 * Centralized type definitions for the web UI. Re-exports common types from the
 * generated API client and defines UI-specific types for result items, evidence,
 * clusters, and citations. Provides type aliases for consistency across components.
 *
 * @module types
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

export type ResearchResultItem = {
  summary?: string;
  confidence?: number;
  evidence?: EvidenceItem[];
  clusters?: ClusterItem[];
  citations?: CitationItem[];
};

export type ResultItem = CrawlResultItem | ResearchResultItem;
