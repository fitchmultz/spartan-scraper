import test from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { loadBridgeConfig, parseRouteId } from "./config.js";
import {
  CAPABILITY_EXTRACT_NATURAL,
  CAPABILITY_RESEARCH_REFINE,
  CAPABILITY_TEMPLATE_GENERATE,
} from "./protocol.js";

test("loadBridgeConfig uses defaults without config file", () => {
  const config = loadBridgeConfig({});
  assert.equal(config.mode, "sdk");
  assert.deepEqual(config.routes[CAPABILITY_EXTRACT_NATURAL], [
    "openai/gpt-5.4",
    "kimi-coding/k2p5",
    "zai/glm-5",
  ]);
  assert.deepEqual(config.routes[CAPABILITY_RESEARCH_REFINE], [
    "openai/gpt-5.4",
    "kimi-coding/k2p5",
    "zai/glm-5",
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
          [CAPABILITY_TEMPLATE_GENERATE]: ["kimi-coding/k2p5", "zai/glm-5"],
        },
      }),
    );

    const config = loadBridgeConfig({ PI_CONFIG_PATH: path });
    assert.equal(config.mode, "fixture");
    assert.deepEqual(config.routes[CAPABILITY_TEMPLATE_GENERATE], [
      "kimi-coding/k2p5",
      "zai/glm-5",
    ]);
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});

test("parseRouteId validates provider/model IDs", () => {
  assert.deepEqual(parseRouteId("openai/gpt-5.4"), {
    provider: "openai",
    model: "gpt-5.4",
  });
  assert.throws(() => parseRouteId("broken-route"));
});
