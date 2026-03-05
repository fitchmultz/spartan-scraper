/**
 * ChainContainer - Container component for chain management functionality
 *
 * This component encapsulates all chain-related state and operations:
 * - Loading and displaying job chains
 * - Creating and deleting chains
 * - Submitting chains to create jobs
 *
 * It does NOT handle:
 * - Job submission or results viewing
 * - Watch management
 * - Batch operations
 *
 * @module ChainContainer
 */

import { useCallback, useEffect, useState, Suspense, lazy } from "react";
import {
  listChains,
  createChain,
  deleteChain,
  submitChain,
  type JobChain,
  type ChainCreateRequest,
} from "../../api";
import { getApiBaseUrl } from "../../lib/api-config";

const ChainBuilder = lazy(() =>
  import("../../components/ChainBuilder").then((mod) => ({
    default: mod.ChainBuilder,
  })),
);

import { ChainList } from "../../components/ChainList";

interface ChainContainerProps {
  onChainSubmit?: () => void;
}

export function ChainContainer({ onChainSubmit }: ChainContainerProps) {
  const [chains, setChains] = useState<JobChain[]>([]);
  const [chainsLoading, setChainsLoading] = useState(false);
  const [showChainBuilder, setShowChainBuilder] = useState(false);

  const refreshChains = useCallback(async () => {
    setChainsLoading(true);
    try {
      const { data, error } = await listChains({ baseUrl: getApiBaseUrl() });
      if (error) throw error;
      setChains(data?.chains || []);
    } catch (err) {
      console.error("Failed to load chains:", err);
    } finally {
      setChainsLoading(false);
    }
  }, []);

  const handleCreateChain = useCallback(
    async (request: ChainCreateRequest) => {
      const { error } = await createChain({
        baseUrl: getApiBaseUrl(),
        body: request,
      });
      if (error) throw error;
      await refreshChains();
      setShowChainBuilder(false);
    },
    [refreshChains],
  );

  const handleDeleteChain = useCallback(
    async (id: string) => {
      const { error } = await deleteChain({
        baseUrl: getApiBaseUrl(),
        path: { id },
      });
      if (error) throw error;
      await refreshChains();
    },
    [refreshChains],
  );

  const handleSubmitChain = useCallback(
    async (id: string, overrides?: Record<string, unknown>) => {
      const formattedOverrides: { [key: string]: { [key: string]: unknown } } =
        {};
      if (overrides) {
        for (const [key, value] of Object.entries(overrides)) {
          if (typeof value === "object" && value !== null) {
            formattedOverrides[key] = value as { [key: string]: unknown };
          }
        }
      }
      const { error } = await submitChain({
        baseUrl: getApiBaseUrl(),
        path: { id },
        body: { overrides: formattedOverrides },
      });
      if (error) throw error;
      onChainSubmit?.();
    },
    [onChainSubmit],
  );

  useEffect(() => {
    refreshChains();
  }, [refreshChains]);

  return (
    <section id="chains">
      {showChainBuilder ? (
        <Suspense
          fallback={
            <div className="loading-placeholder">Loading chain builder...</div>
          }
        >
          <ChainBuilder
            onCreate={handleCreateChain}
            onCancel={() => setShowChainBuilder(false)}
          />
        </Suspense>
      ) : (
        <ChainList
          chains={chains}
          onRefresh={refreshChains}
          onDelete={handleDeleteChain}
          onSubmit={handleSubmitChain}
          loading={chainsLoading}
          onCreateClick={() => setShowChainBuilder(true)}
        />
      )}
    </section>
  );
}
