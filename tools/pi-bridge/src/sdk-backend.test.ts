import test from "node:test";
import assert from "node:assert/strict";
import { AuthStorage, ModelRegistry } from "@mariozechner/pi-coding-agent";
import {
  SDKBackend,
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

test("health reports only auth-ready routes as available", async () => {
  const authStorage = AuthStorage.inMemory({
    openai: { type: "api_key", key: "test-openai-key" },
  });
  const fakeModelRegistry = {
    authStorage,
    find(provider: string, model: string) {
      if (provider === "openai" && model === "gpt-5.4") {
        return { provider, id: model, input: ["text", "image"] };
      }
      if (provider === "test-provider" && model === "test-model") {
        return { provider, id: model, input: ["text"] };
      }
      return undefined;
    },
    getError() {
      return undefined;
    },
  } as unknown as ModelRegistry;
  const backend = new SDKBackend(
    {
      "extract.natural_language": [
        "openai/gpt-5.4",
        "test-provider/test-model",
      ],
    },
    { authStorage, modelRegistry: fakeModelRegistry },
  );

  const health = await backend.health("/tmp/pi-agent");

  assert.deepEqual(health.available?.["extract.natural_language"], [
    "openai/gpt-5.4",
  ]);
  assert.deepEqual(health.route_status?.["extract.natural_language"], [
    {
      route_id: "openai/gpt-5.4",
      provider: "openai",
      model: "gpt-5.4",
      status: "ready",
      model_found: true,
      auth_configured: true,
    },
    {
      route_id: "test-provider/test-model",
      provider: "test-provider",
      model: "test-model",
      status: "missing_auth",
      message: "no auth configured for provider test-provider",
      model_found: true,
      auth_configured: false,
    },
  ]);
});

test("health reports missing models distinctly", async () => {
  const backend = new SDKBackend(
    {
      "template.generate": ["openai/not-a-real-model"],
    },
    { authStorage: AuthStorage.inMemory() },
  );

  const health = await backend.health("/tmp/pi-agent");

  assert.deepEqual(health.available?.["template.generate"], []);
  assert.deepEqual(health.route_status?.["template.generate"], [
    {
      route_id: "openai/not-a-real-model",
      provider: "openai",
      model: "not-a-real-model",
      status: "missing_model",
      message: "model not found for route openai/not-a-real-model",
      model_found: false,
      auth_configured: false,
    },
  ]);
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
