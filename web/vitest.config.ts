/**
 * Purpose: Provide the repo-owned Vitest entrypoint for web tests and pin the Node localStorage backing file.
 * Responsibilities: Merge the shared Vite config, preserve the jsdom test environment, and inject the localStorage file path used by workers.
 * Scope: Vitest execution for the web package only; build and dev-server behavior stay in vite.config.ts.
 * Usage: Vitest auto-loads this file, and package scripts/Makefile invoke it through `vitest --config ./vitest.config.ts`.
 * Invariants/Assumptions: Node 25 honors `--localstorage-file`; the file lives under the web package root and may be deleted safely between runs.
 */
import { fileURLToPath } from "node:url";

import { defineConfig, mergeConfig } from "vitest/config";

import viteConfig from "./vite.config";

const localStorageFilePath = fileURLToPath(
  new URL("./.vitest-localstorage", import.meta.url),
);

export default defineConfig((env) =>
  mergeConfig(viteConfig(env), {
    test: {
      execArgv: [`--localstorage-file=${localStorageFilePath}`],
    },
  }),
);
