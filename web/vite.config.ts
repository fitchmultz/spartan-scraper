/**
 * Vite configuration for the Spartan web frontend.
 *
 * Configures React plugin, Vitest test environment with jsdom,
 * development server proxy for API endpoints, and optimized build
 * chunking to minimize initial bundle size.
 *
 * Chunking strategy:
 * - vendor-react: React core libraries (always needed)
 * - vendor-ui: UI primitives (cmdk, radix-ui) - needed for interactions
 * - vendor-onboarding: react-joyride and deps - only needed for first-time users
 * - feature-*: Major feature areas split by directory
 * - api: Generated API client code (large types)
 *
 * @module vite.config
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
      rollupOptions: {
        output: {
          manualChunks: chunkNameForModuleId,
        },
      },
    },
  };
});
