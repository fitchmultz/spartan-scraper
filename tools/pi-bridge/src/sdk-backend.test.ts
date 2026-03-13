import test from "node:test";
import assert from "node:assert/strict";
import { AuthStorage, ModelRegistry } from "@mariozechner/pi-coding-agent";
import {
  modelSupportsImages,
  runWithFallback,
  truncateHTMLForPrompt,
} from "./sdk-backend.js";

test("ModelRegistry resolves verified pi model IDs", () => {
  const registry = new ModelRegistry(AuthStorage.inMemory());
  assert.ok(registry.find("openai", "gpt-5.4"));
  assert.ok(registry.find("kimi-coding", "k2p5"));
  assert.ok(registry.find("zai", "glm-5"));
});

test("runWithFallback returns the first successful route", async () => {
  const calls: string[] = [];
  const result = await runWithFallback(
    ["openai/gpt-5.4", "kimi-coding/k2p5"],
    async (routeId) => {
      calls.push(routeId);
      if (routeId === "openai/gpt-5.4") {
        throw new Error("simulated failure");
      }
      return routeId;
    },
  );

  assert.equal(result, "kimi-coding/k2p5");
  assert.deepEqual(calls, ["openai/gpt-5.4", "kimi-coding/k2p5"]);
});

test("modelSupportsImages matches current verified model capabilities", () => {
  const registry = new ModelRegistry(AuthStorage.inMemory());
  const openai = registry.find("openai", "gpt-5.4");
  const zai = registry.find("zai", "glm-5");
  assert.ok(openai);
  assert.ok(zai);
  assert.equal(modelSupportsImages(openai), true);
  assert.equal(modelSupportsImages(zai), false);
});

test("truncateHTMLForPrompt respects max_content_chars", () => {
  const html = "<article>" + "word ".repeat(20) + "</article>";
  const truncated = truncateHTMLForPrompt(html, 40);

  assert.ok(truncated.endsWith("..."));
  assert.ok(truncated.length <= 43);
  assert.notEqual(truncated, html);
});
