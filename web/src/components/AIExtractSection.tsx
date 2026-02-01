import { useState } from "react";

interface AIExtractSectionProps {
  enabled: boolean;
  setEnabled: (value: boolean) => void;
  mode: "natural_language" | "schema_guided";
  setMode: (value: "natural_language" | "schema_guided") => void;
  prompt: string;
  setPrompt: (value: string) => void;
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
          Use AI to intelligently extract data from HTML without writing
          selectors.
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

      {showHelp && (
        <div className="mb-4 p-3 bg-slate-800 rounded-md text-sm text-slate-300">
          <p className="mb-2">
            <strong>AI Extraction</strong> uses LLM technology to extract
            structured data from HTML.
          </p>
          <ul className="list-disc list-inside space-y-1 text-slate-400">
            <li>
              <strong>Natural Language:</strong> Describe what you want to
              extract in plain English
            </li>
            <li>
              <strong>Schema Guided:</strong> Provide an example object to guide
              extraction
            </li>
          </ul>
        </div>
      )}

      <div className="space-y-4">
        {/* Mode selector */}
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

        {/* Fields input */}
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
            onChange={(e) => setFields(e.target.value)}
            placeholder="e.g., title, price, description, rating"
            className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-md text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent"
          />
          <p className="mt-1 text-xs text-slate-500">
            Optional: Specify field names to guide the AI extraction
          </p>
        </div>

        {/* Prompt input */}
        <div>
          <label
            htmlFor="ai-prompt"
            className="block text-sm font-medium text-slate-300 mb-2"
          >
            {mode === "natural_language"
              ? "Extraction Instructions"
              : "Schema Example (JSON)"}
          </label>
          <textarea
            id="ai-prompt"
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            rows={mode === "natural_language" ? 3 : 5}
            placeholder={
              mode === "natural_language"
                ? "e.g., Extract all product prices, names, and ratings from this e-commerce page"
                : '{"product_name": "Example Product", "price": "$19.99", "rating": 4.5}'
            }
            className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-md text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent font-mono text-sm"
          />
          <p className="mt-1 text-xs text-slate-500">
            {mode === "natural_language"
              ? "Describe what data you want to extract in natural language"
              : "Provide a JSON object with example field names and values"}
          </p>
        </div>
      </div>
    </div>
  );
}
