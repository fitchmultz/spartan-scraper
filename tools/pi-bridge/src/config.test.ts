/**
 * Purpose: Verify config behavior with automated regression coverage.
 * Responsibilities: Define focused test cases, fixtures, and assertions for the module under test.
 * Scope: Automated test coverage only; production logic stays in the adjacent source modules.
 * Usage: Run through the repo test entrypoints or the feature-local test command.
 * Invariants/Assumptions: Tests should describe the current contract clearly and remain deterministic under local CI settings.
 */

/*
Purpose:
- Verify bridge config loading and route normalization behavior.
Responsibilities:
- Assert defaults, config-file overrides, and route parsing semantics.
- Preserve explicit empty capability overrides as intentional disables.
Scope:
- Bridge config tests only.
Usage:
- Run with pnpm --dir tools/pi-bridge test.
Invariants/Assumptions:
- Missing config keys inherit defaults.
- Explicit empty route arrays remain empty instead of restoring defaults.
*/
import test from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { AuthStorage, ModelRegistry } from "@earendil-works/pi-coding-agent";
import { loadBridgeConfig, parseRouteId, type BridgeConfigFile } from "./config.js";
import {
  CAPABILITY_EXPORT_SHAPE,
  CAPABILITY_EXTRACT_NATURAL,
  CAPABILITY_RESEARCH_REFINE,
  CAPABILITY_TEMPLATE_GENERATE,
} from "./protocol.js";

test("loadBridgeConfig uses defaults without config file", () => {
  const config = loadBridgeConfig({});
  assert.equal(config.mode, "sdk");
  assert.deepEqual(config.routes[CAPABILITY_EXTRACT_NATURAL], [
    "kimi-coding/kimi-for-coding",
    "zai/glm-5.2",
    "openai/gpt-5.5",
  ]);
  assert.deepEqual(config.routes[CAPABILITY_RESEARCH_REFINE], [
    "kimi-coding/kimi-for-coding",
    "zai/glm-5.2",
    "openai/gpt-5.5",
  ]);
  assert.deepEqual(config.routes[CAPABILITY_EXPORT_SHAPE], [
    "kimi-coding/kimi-for-coding",
    "zai/glm-5.2",
    "openai/gpt-5.5",
  ]);
});

test("loadBridgeConfig loads route overrides from PI_CONFIG_PATH", () => {
  const dir = mkdtempSync(join(tmpdir(), "pi-bridge-config-"));
  try {
    const path = join(dir, "bridge.json");
    writeFileSync(
      path,
      JSON.stringify({
        mode: "fixture",
        routes: {
          [CAPABILITY_TEMPLATE_GENERATE]: ["kimi-coding/kimi-for-coding", "zai/glm-5.2"],
        },
      }),
    );

    const config = loadBridgeConfig({ PI_CONFIG_PATH: path });
    assert.equal(config.mode, "fixture");
    assert.deepEqual(config.routes[CAPABILITY_TEMPLATE_GENERATE], [
      "kimi-coding/kimi-for-coding",
      "zai/glm-5.2",
    ]);
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});

test("loadBridgeConfig preserves explicit empty capability overrides", () => {
  const dir = mkdtempSync(join(tmpdir(), "pi-bridge-config-"));
  try {
    const path = join(dir, "bridge.json");
    writeFileSync(
      path,
      JSON.stringify({
        mode: "fixture",
        routes: {
          [CAPABILITY_TEMPLATE_GENERATE]: [],
        },
      }),
    );

    const config = loadBridgeConfig({ PI_CONFIG_PATH: path });
    assert.equal(config.mode, "fixture");
    assert.deepEqual(config.routes[CAPABILITY_TEMPLATE_GENERATE], []);
    assert.deepEqual(config.routes[CAPABILITY_EXTRACT_NATURAL], [
      "kimi-coding/kimi-for-coding",
      "zai/glm-5.2",
      "openai/gpt-5.5",
    ]);
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});

test("parseRouteId validates provider/model IDs", () => {
  assert.deepEqual(parseRouteId("openai/gpt-5.5"), {
    provider: "openai",
    model: "gpt-5.5",
  });
  assert.throws(() => parseRouteId("broken-route"));
});

test("repo pi route defaults resolve against current pi models", () => {
  const registry = ModelRegistry.create(AuthStorage.inMemory());
  const config = JSON.parse(
    readFileSync("../../config/pi-routes.json", "utf8"),
  ) as BridgeConfigFile;

  const missingRoutes = Object.values(config.routes ?? {})
    .flat()
    .filter((routeId, index, routes) => routes.indexOf(routeId) === index)
    .filter((routeId) => {
      const { provider, model } = parseRouteId(routeId);
      return !registry.find(provider, model);
    });

  assert.deepEqual(missingRoutes, []);
});
