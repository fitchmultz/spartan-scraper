/**
 * Vite configuration for the Spartan web frontend.
 *
 * Configures React plugin, Vitest test environment with jsdom,
 * and development server proxy for API endpoints.
 */
import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  const apiTarget = env.VITE_API_BASE_URL || "http://localhost:8741";

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
        "/v1": apiTarget,
        "/healthz": apiTarget,
      },
    },
  };
});
