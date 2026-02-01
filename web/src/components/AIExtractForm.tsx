import { useState } from "react";
import { aiExtractPreview } from "../api";
import type { AiExtractPreviewResponse } from "../api";

interface AIExtractFormProps {
  url: string;
  html: string;
  onExtract?: (result: AiExtractPreviewResponse) => void;
}

interface ExtractResult {
  fields: Record<string, { values: string[]; source: string }>;
  confidence: number;
  explanation: string;
  tokens_used: number;
  cached: boolean;
}

export function AIExtractForm({ url, html, onExtract }: AIExtractFormProps) {
  const [mode, setMode] = useState<"natural_language" | "schema_guided">(
    "natural_language",
  );
  const [prompt, setPrompt] = useState("");
  const [fields, setFields] = useState("");
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<ExtractResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleExtract = async () => {
    if (!prompt.trim()) {
      setError("Please enter extraction instructions");
      return;
    }

    setLoading(true);
    setError(null);
    setResult(null);

    try {
      const response = await aiExtractPreview({
        body: {
          url,
          html,
          mode,
          prompt,
          fields: fields
            .split(",")
            .map((f) => f.trim())
            .filter(Boolean),
        },
      });

      if (response.error) {
        setError(`Extraction failed: ${JSON.stringify(response.error)}`);
        return;
      }

      const data = response.data as AiExtractPreviewResponse;
      const extractResult: ExtractResult = {
        fields: data.fields as Record<
          string,
          { values: string[]; source: string }
        >,
        confidence: data.confidence ?? 0,
        explanation: data.explanation || "",
        tokens_used: data.tokens_used ?? 0,
        cached: data.cached ?? false,
      };

      setResult(extractResult);
      if (onExtract) {
        onExtract(data);
      }
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "An unexpected error occurred",
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Input Section */}
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
            Fields to Extract
          </label>
          <input
            id="ai-fields"
            type="text"
            value={fields}
            onChange={(e) => setFields(e.target.value)}
            placeholder="e.g., title, price, description, rating (comma-separated)"
            className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-md text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent"
          />
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
            rows={mode === "natural_language" ? 4 : 6}
            placeholder={
              mode === "natural_language"
                ? "Describe what you want to extract. For example:\n\nExtract all product information including name, price, description, and rating. Look for items in the product grid."
                : '{\n  "product_name": "Example Product",\n  "price": "$19.99",\n  "description": "Product description here",\n  "rating": 4.5,\n  "in_stock": true\n}'
            }
            className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-md text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent font-mono text-sm"
          />
        </div>

        {/* Extract button */}
        <button
          type="button"
          onClick={handleExtract}
          disabled={loading}
          className="w-full flex items-center justify-center gap-2 px-4 py-3 text-sm font-medium text-white bg-purple-600 hover:bg-purple-700 disabled:bg-purple-800 disabled:cursor-not-allowed rounded-md transition-colors"
        >
          {loading ? (
            <>
              <span className="animate-spin">⏳</span>
              Extracting with AI...
            </>
          ) : (
            <>
              <span>✨</span>
              Extract Data with AI
            </>
          )}
        </button>
      </div>

      {/* Error display */}
      {error && (
        <div className="flex items-start gap-3 p-4 bg-red-900/20 border border-red-700/50 rounded-lg">
          <span className="text-red-400 flex-shrink-0">⚠️</span>
          <div className="text-sm text-red-200">{error}</div>
        </div>
      )}

      {/* Results display */}
      {result && (
        <div className="space-y-4">
          {/* Stats bar */}
          <div className="flex flex-wrap items-center gap-4 p-3 bg-slate-800 rounded-lg">
            <div className="flex items-center gap-2">
              <span className="text-green-400">✓</span>
              <span className="text-sm text-slate-300">
                Confidence:{" "}
                <span className="font-medium text-green-400">
                  {Math.round(result.confidence * 100)}%
                </span>
              </span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-purple-400">⚡</span>
              <span className="text-sm text-slate-300">
                Tokens:{" "}
                <span className="font-medium text-purple-400">
                  {result.tokens_used.toLocaleString()}
                </span>
              </span>
            </div>
            {result.cached && (
              <div className="flex items-center gap-2">
                <span className="text-blue-400">💾</span>
                <span className="text-sm text-blue-400">Cached</span>
              </div>
            )}
          </div>

          {/* Explanation */}
          {result.explanation && (
            <div className="p-3 bg-purple-900/20 border border-purple-700/50 rounded-lg">
              <p className="text-sm text-purple-200">{result.explanation}</p>
            </div>
          )}

          {/* Fields table */}
          <div className="border border-slate-700 rounded-lg overflow-hidden">
            <div className="bg-slate-800 px-4 py-2 border-b border-slate-700">
              <h4 className="text-sm font-medium text-slate-200">
                Extracted Fields
              </h4>
            </div>
            <div className="divide-y divide-slate-700">
              {Object.entries(result.fields).map(([name, field]) => (
                <div key={name} className="px-4 py-3 bg-slate-800/50">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-sm font-medium text-slate-200">
                      {name}
                    </span>
                    <span className="px-1.5 py-0.5 text-xs text-slate-400 bg-slate-700 rounded">
                      {field.source}
                    </span>
                  </div>
                  <div className="space-y-1">
                    {field.values.map((value) => (
                      <div
                        key={value}
                        className="text-sm text-slate-400 font-mono truncate"
                      >
                        {value}
                      </div>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
