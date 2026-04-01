/**
 * Purpose: Provide the vite.config module for this repository.
 * Responsibilities: Define the file-local logic, exports, and helpers that belong to this module.
 * Scope: This file only; broader orchestration stays in adjacent modules.
 * Usage: Import from the owning package or feature surface.
 * Invariants/Assumptions: The file should stay aligned with surrounding source-of-truth contracts and avoid hidden side effects.
 */

import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import { chunkNameForModuleId } from "./src/lib/build-chunks";
import {
  resolveApiProxyTarget,
  resolveWebSocketProxyTarget,
} from "./src/lib/dev-proxy-config";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  const apiTarget = resolveApiProxyTarget(env);
  const wsTarget = resolveWebSocketProxyTarget(apiTarget);

  return {
    plugins: [react()],
    test: {
      environment: "jsdom",
      globals: true,
      setupFiles: ["./src/test/setup.ts"],
      // Run tests once in CI mode (non-interactive)
      watch: process.env.CI !== "1",
    },
    server: {
      proxy: {
        "/v1/ws": {
          target: wsTarget,
          ws: true,
        },
        "/v1": {
          target: apiTarget,
        },
        "/healthz": {
          target: apiTarget,
        },
      },
    },
    build: {
      target: "baseline-widely-available",
      rollupOptions: {
        output: {
          manualChunks: chunkNameForModuleId,
        },
      },
    },
  };
});
