interface ResearchAgenticSectionProps {
  enabled: boolean;
  setEnabled: (value: boolean) => void;
  instructions: string;
  setInstructions: (value: string) => void;
  maxRounds: number;
  setMaxRounds: (value: number) => void;
  maxFollowUpUrls: number;
  setMaxFollowUpUrls: (value: number) => void;
}

export function ResearchAgenticSection({
  enabled,
  setEnabled,
  instructions,
  setInstructions,
  maxRounds,
  setMaxRounds,
  maxFollowUpUrls,
  setMaxFollowUpUrls,
}: ResearchAgenticSectionProps) {
  return (
    <div className="border border-cyan-700/50 rounded-lg p-4 bg-cyan-900/10">
      <div className="flex items-center justify-between gap-3 mb-4">
        <div>
          <div className="flex items-center gap-2">
            <span className="text-cyan-400">🧭</span>
            <span className="font-medium text-slate-200">Agentic Research</span>
            {enabled ? (
              <span className="px-2 py-0.5 text-xs font-medium text-cyan-300 bg-cyan-900/50 rounded">
                Enabled
              </span>
            ) : null}
          </div>
          <p className="mt-2 text-sm text-slate-400">
            Let pi select bounded follow-up URLs from discovered evidence and
            synthesize a final research brief. Deterministic research remains
            the baseline and is always preserved.
          </p>
        </div>
        <button
          type="button"
          onClick={() => setEnabled(!enabled)}
          className={`px-4 py-2 text-sm font-medium rounded-md transition-colors ${
            enabled
              ? "bg-slate-700 text-slate-200 hover:bg-slate-600"
              : "bg-cyan-600 text-white hover:bg-cyan-700"
          }`}
        >
          {enabled ? "Disable" : "Enable Agentic Research"}
        </button>
      </div>

      {enabled ? (
        <div className="space-y-4">
          <div>
            <label
              htmlFor="agentic-instructions"
              className="block text-sm font-medium text-slate-300 mb-2"
            >
              Additional instructions
            </label>
            <textarea
              id="agentic-instructions"
              value={instructions}
              onChange={(event) => setInstructions(event.target.value)}
              rows={3}
              placeholder="Prioritize pricing, contract terms, and support commitments. Prefer primary source pages over blog summaries."
              className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-md text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-transparent"
            />
            <p className="mt-1 text-xs text-slate-500">
              Optional: guide follow-up selection and final synthesis without
              replacing the original query.
            </p>
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <div>
              <label
                htmlFor="agentic-max-rounds"
                className="block text-sm font-medium text-slate-300 mb-2"
              >
                Max follow-up rounds
              </label>
              <input
                id="agentic-max-rounds"
                type="number"
                min={1}
                max={3}
                value={maxRounds}
                onChange={(event) =>
                  setMaxRounds(Number.parseInt(event.target.value, 10) || 1)
                }
                className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-md text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-transparent"
              />
            </div>
            <div>
              <label
                htmlFor="agentic-max-follow-up-urls"
                className="block text-sm font-medium text-slate-300 mb-2"
              >
                Max follow-up URLs per round
              </label>
              <input
                id="agentic-max-follow-up-urls"
                type="number"
                min={1}
                max={10}
                value={maxFollowUpUrls}
                onChange={(event) =>
                  setMaxFollowUpUrls(
                    Number.parseInt(event.target.value, 10) || 1,
                  )
                }
                className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-md text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-transparent"
              />
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
