/**
 * Purpose: Render the a i extract section UI surface for the web operator experience.
 * Responsibilities: Define the component, its local view helpers, and the presentation logic owned by this file.
 * Scope: File-local UI behavior only; routing, persistence, and broader feature orchestration stay outside this file.
 * Usage: Import from the surrounding feature or route components that own this surface.
 * Invariants/Assumptions: Props and callbacks come from the surrounding feature contracts and should remain the single source of truth.
 */

import { useState } from "react";

interface AIExtractSectionProps {
  enabled: boolean;
  setEnabled: (value: boolean) => void;
  mode: "natural_language" | "schema_guided";
  setMode: (value: "natural_language" | "schema_guided") => void;
  prompt: string;
  setPrompt: (value: string) => void;
  schemaText: string;
  setSchemaText: (value: string) => void;
  fields: string;
  setFields: (value: string) => void;
}

export function AIExtractSection({
  enabled,
  setEnabled,
  mode,
  setMode,
  prompt,
  setPrompt,
  schemaText,
  setSchemaText,
  fields,
  setFields,
}: AIExtractSectionProps) {
  const [showHelp, setShowHelp] = useState(false);

  if (!enabled) {
    return (
      <div className="border border-slate-700 rounded-lg p-4 bg-slate-800/50">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-purple-400">✨</span>
            <span className="font-medium text-slate-200">
              AI-Powered Extraction
            </span>
          </div>
          <button
            type="button"
            onClick={() => setEnabled(true)}
            className="px-4 py-2 text-sm font-medium text-white bg-purple-600 hover:bg-purple-700 rounded-md transition-colors"
          >
            Enable AI Extraction
          </button>
        </div>
        <p className="mt-2 text-sm text-slate-400">
          Use AI to intelligently extract structured fields without hand-writing
          selectors first.
        </p>
      </div>
    );
  }

  return (
    <div className="border border-purple-700/50 rounded-lg p-4 bg-purple-900/10">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <span className="text-purple-400">✨</span>
          <span className="font-medium text-slate-200">
            AI-Powered Extraction
          </span>
          <span className="px-2 py-0.5 text-xs font-medium text-purple-300 bg-purple-900/50 rounded">
            Enabled
          </span>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => setShowHelp(!showHelp)}
            className="p-1.5 text-slate-400 hover:text-slate-200 hover:bg-slate-700 rounded-md transition-colors"
            title="Show help"
          >
            <span>❓</span>
          </button>
          <button
            type="button"
            onClick={() => setEnabled(false)}
            className="px-3 py-1.5 text-sm font-medium text-slate-300 hover:text-white hover:bg-slate-700 rounded-md transition-colors"
          >
            Disable
          </button>
        </div>
      </div>

      {showHelp ? (
        <div className="mb-4 p-3 bg-slate-800 rounded-md text-sm text-slate-300">
          <p className="mb-2">
            <strong>AI Extraction</strong> augments template extraction with
            route-backed LLM extraction.
          </p>
          <ul className="list-disc list-inside space-y-1 text-slate-400">
            <li>
              <strong>Natural Language:</strong> describe what to pull from the
              page in plain English.
            </li>
            <li>
              <strong>Schema Guided:</strong> provide an example JSON object so
              field names and shapes are explicit.
            </li>
            <li>
              <strong>Fields:</strong> optionally constrain extraction to a
              known set of output keys.
            </li>
          </ul>
        </div>
      ) : null}

      <div className="space-y-4">
        <div>
          <span className="block text-sm font-medium text-slate-300 mb-2">
            Extraction Mode
          </span>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => setMode("natural_language")}
              className={`flex-1 px-4 py-2 text-sm font-medium rounded-md transition-colors ${
                mode === "natural_language"
                  ? "bg-purple-600 text-white"
                  : "bg-slate-700 text-slate-300 hover:bg-slate-600"
              }`}
            >
              Natural Language
            </button>
            <button
              type="button"
              onClick={() => setMode("schema_guided")}
              className={`flex-1 px-4 py-2 text-sm font-medium rounded-md transition-colors ${
                mode === "schema_guided"
                  ? "bg-purple-600 text-white"
                  : "bg-slate-700 text-slate-300 hover:bg-slate-600"
              }`}
            >
              Schema Guided
            </button>
          </div>
        </div>

        <div>
          <label
            htmlFor="ai-fields"
            className="block text-sm font-medium text-slate-300 mb-2"
          >
            Fields to Extract (comma-separated)
          </label>
          <input
            id="ai-fields"
            type="text"
            value={fields}
            onChange={(event) => setFields(event.target.value)}
            placeholder="e.g., title, price, description, rating"
            className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-md text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent"
          />
          <p className="mt-1 text-xs text-slate-500">
            Optional: guide the extractor toward known output fields.
          </p>
        </div>

        {mode === "natural_language" ? (
          <div>
            <label
              htmlFor="ai-prompt"
              className="block text-sm font-medium text-slate-300 mb-2"
            >
              Extraction Instructions
            </label>
            <textarea
              id="ai-prompt"
              value={prompt}
              onChange={(event) => setPrompt(event.target.value)}
              rows={3}
              placeholder="Extract the product title, price, availability, and shipping details from this page."
              className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-md text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent"
            />
            <p className="mt-1 text-xs text-slate-500">
              Describe what data should be extracted and any normalization
              hints.
            </p>
          </div>
        ) : (
          <div>
            <label
              htmlFor="ai-schema"
              className="block text-sm font-medium text-slate-300 mb-2"
            >
              Schema Example (JSON object)
            </label>
            <textarea
              id="ai-schema"
              value={schemaText}
              onChange={(event) => setSchemaText(event.target.value)}
              rows={6}
              placeholder={
                '{\n  "title": "Example product",\n  "price": "$19.99",\n  "in_stock": true\n}'
              }
              className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-md text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent font-mono text-sm"
            />
            <p className="mt-1 text-xs text-slate-500">
              Provide an example JSON object. Spartan will send it as structured
              schema guidance, not raw prompt text.
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
