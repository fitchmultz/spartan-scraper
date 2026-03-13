import test from "node:test";
import assert from "node:assert/strict";
import { AuthStorage, ModelRegistry } from "@mariozechner/pi-coding-agent";
import {
  SDKBackend,
  modelSupportsImages,
  runWithFallback,
  truncateHTMLForPrompt,
} from "./sdk-backend.js";
import {
  CAPABILITY_EXTRACT_NATURAL,
  CAPABILITY_TEMPLATE_GENERATE,
} from "./protocol.js";

type FakeModel = {
  provider: string;
  id: string;
  input: string[];
};

function createFakeModelRegistry(options: {
  authStorage?: AuthStorage;
  models: Record<string, FakeModel>;
  apiKeys?: Record<string, string | undefined>;
  loadError?: string;
}): ModelRegistry {
  const authStorage = options.authStorage ?? AuthStorage.inMemory();
  const apiKeys = options.apiKeys ?? {};

  return {
    authStorage,
    find(provider: string, model: string) {
      return options.models[`${provider}/${model}`];
    },
    async getApiKey(model: FakeModel) {
      return apiKeys[model.provider];
    },
    getError() {
      return options.loadError;
    },
  } as unknown as ModelRegistry;
}

function createToolResponse(options: {
  toolName: string;
  arguments: Record<string, unknown>;
  provider: string;
  model: string;
  tokens?: number;
}) {
  return {
    provider: options.provider,
    model: options.model,
    usage: { totalTokens: options.tokens ?? 123 },
    content: [
      {
        type: "toolCall",
        id: `call-${options.toolName}`,
        name: options.toolName,
        arguments: options.arguments,
      },
    ],
  };
}

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
  const fakeModelRegistry = createFakeModelRegistry({
    authStorage,
    models: {
      "openai/gpt-5.4": { provider: "openai", id: "gpt-5.4", input: ["text", "image"] },
      "test-provider/test-model": { provider: "test-provider", id: "test-model", input: ["text"] },
    },
    apiKeys: {
      openai: "test-openai-key",
    },
  });
  const backend = new SDKBackend(
    {
      [CAPABILITY_EXTRACT_NATURAL]: [
        "openai/gpt-5.4",
        "test-provider/test-model",
      ],
    },
    { authStorage, modelRegistry: fakeModelRegistry },
  );

  const health = await backend.health("/tmp/pi-agent");

  assert.deepEqual(health.available?.[CAPABILITY_EXTRACT_NATURAL], [
    "openai/gpt-5.4",
  ]);
  assert.deepEqual(health.route_status?.[CAPABILITY_EXTRACT_NATURAL], [
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
      [CAPABILITY_TEMPLATE_GENERATE]: ["openai/not-a-real-model"],
    },
    { authStorage: AuthStorage.inMemory() },
  );

  const health = await backend.health("/tmp/pi-agent");

  assert.deepEqual(health.available?.[CAPABILITY_TEMPLATE_GENERATE], []);
  assert.deepEqual(health.route_status?.[CAPABILITY_TEMPLATE_GENERATE], [
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

test("runWithFallback reports empty capabilities clearly", async () => {
  await assert.rejects(
    () => runWithFallback([], async () => "unused", { capability: CAPABILITY_EXTRACT_NATURAL }),
    /no routes configured for capability extract\.natural_language/,
  );
});

test("runWithFallback aggregates ordered route failures", async () => {
  await assert.rejects(
    () =>
      runWithFallback(
        ["openai/gpt-5.4", "kimi-coding/k2p5"],
        async (routeId) => {
          if (routeId === "openai/gpt-5.4") {
            throw new Error("provider outage");
          }
          throw new Error("rate limited");
        },
        { capability: CAPABILITY_EXTRACT_NATURAL },
      ),
    /all routes failed for capability extract\.natural_language after 2 attempts: openai\/gpt-5\.4: provider outage \| kimi-coding\/k2p5: rate limited/,
  );
});

test("extract falls back to the next route after provider failure", async () => {
  const calls: string[] = [];
  const backend = new SDKBackend(
    {
      [CAPABILITY_EXTRACT_NATURAL]: ["openai/gpt-5.4", "kimi-coding/k2p5"],
    },
    {
      modelRegistry: createFakeModelRegistry({
        models: {
          "openai/gpt-5.4": { provider: "openai", id: "gpt-5.4", input: ["text", "image"] },
          "kimi-coding/k2p5": { provider: "kimi-coding", id: "k2p5", input: ["text"] },
        },
        apiKeys: {
          openai: "openai-key",
          "kimi-coding": "kimi-key",
        },
      }),
      completeFn: (async (model: FakeModel) => {
        calls.push(`${model.provider}/${model.id}`);
        if (model.provider === "openai") {
          throw new Error("provider outage");
        }
        return createToolResponse({
          toolName: "submit_extraction",
          arguments: {
            fields: {
              title: "Fallback success",
              price: "$19.99",
            },
            confidence: 0.88,
            explanation: "Recovered on the secondary route.",
          },
          provider: model.provider,
          model: model.id,
          tokens: 456,
        });
      }) as unknown as typeof import("@mariozechner/pi-ai").complete,
    },
  );

  const result = await backend.extract(CAPABILITY_EXTRACT_NATURAL, {
    html: "<html><h1>Fallback success</h1></html>",
    url: "https://example.com/product",
    mode: "natural_language",
    prompt: "Extract the title and price",
  });

  assert.deepEqual(calls, ["openai/gpt-5.4", "kimi-coding/k2p5"]);
  assert.equal(result.route_id, "kimi-coding/k2p5");
  assert.equal(result.provider, "kimi-coding");
  assert.equal(result.model, "k2p5");
  assert.equal(result.tokens_used, 456);
  assert.deepEqual(result.fields.title.values, ["Fallback success"]);
});

test("extract falls back after malformed model output", async () => {
  const calls: string[] = [];
  const backend = new SDKBackend(
    {
      [CAPABILITY_EXTRACT_NATURAL]: ["openai/gpt-5.4", "kimi-coding/k2p5"],
    },
    {
      modelRegistry: createFakeModelRegistry({
        models: {
          "openai/gpt-5.4": { provider: "openai", id: "gpt-5.4", input: ["text", "image"] },
          "kimi-coding/k2p5": { provider: "kimi-coding", id: "k2p5", input: ["text"] },
        },
        apiKeys: {
          openai: "openai-key",
          "kimi-coding": "kimi-key",
        },
      }),
      completeFn: (async (model: FakeModel) => {
        calls.push(`${model.provider}/${model.id}`);
        if (model.provider === "openai") {
          return {
            provider: model.provider,
            model: model.id,
            usage: { totalTokens: 111 },
            content: [{ type: "text", text: "No tool call" }],
          };
        }
        return createToolResponse({
          toolName: "submit_extraction",
          arguments: {
            fields: {
              title: "Recovered after malformed output",
            },
            confidence: 0.91,
          },
          provider: model.provider,
          model: model.id,
        });
      }) as unknown as typeof import("@mariozechner/pi-ai").complete,
    },
  );

  const result = await backend.extract(CAPABILITY_EXTRACT_NATURAL, {
    html: "<html><h1>Recovered</h1></html>",
    url: "https://example.com/recovered",
    mode: "natural_language",
    prompt: "Extract the title",
  });

  assert.deepEqual(calls, ["openai/gpt-5.4", "kimi-coding/k2p5"]);
  assert.equal(result.route_id, "kimi-coding/k2p5");
  assert.deepEqual(result.fields.title.values, ["Recovered after malformed output"]);
});

test("generateTemplate reports aggregated fallback failures", async () => {
  const backend = new SDKBackend(
    {
      [CAPABILITY_TEMPLATE_GENERATE]: ["openai/gpt-5.4", "kimi-coding/k2p5"],
    },
    {
      modelRegistry: createFakeModelRegistry({
        models: {
          "openai/gpt-5.4": { provider: "openai", id: "gpt-5.4", input: ["text", "image"] },
          "kimi-coding/k2p5": { provider: "kimi-coding", id: "k2p5", input: ["text"] },
        },
        apiKeys: {
          openai: "openai-key",
          "kimi-coding": "kimi-key",
        },
      }),
      completeFn: (async (model: FakeModel) => {
        if (model.provider === "openai") {
          return {
            provider: model.provider,
            model: model.id,
            usage: { totalTokens: 222 },
            content: [{ type: "text", text: "No tool call" }],
          };
        }
        return createToolResponse({
          toolName: "submit_template",
          arguments: {
            template: {
              name: "product",
              selectors: [],
            },
          },
          provider: model.provider,
          model: model.id,
        });
      }) as unknown as typeof import("@mariozechner/pi-ai").complete,
    },
  );

  await assert.rejects(
    () =>
      backend.generateTemplate(CAPABILITY_TEMPLATE_GENERATE, {
        html: "<html><h1>Product</h1></html>",
        url: "https://example.com/product",
        description: "Generate a product template",
      }),
    /all routes failed for capability template\.generate after 2 attempts: openai\/gpt-5\.4: model did not call submit_template \| kimi-coding\/k2p5: template result must include at least one selector/,
  );
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
