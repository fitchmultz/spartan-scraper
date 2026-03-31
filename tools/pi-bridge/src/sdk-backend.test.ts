/**
 * Purpose: Verify SDK backend route fallback and tolerant response extraction behavior.
 * Responsibilities: Assert tool-call selection, text fallback, multimodal handling, and strict failure cases.
 * Scope: SDK backend tests only.
 * Usage: Run with `pnpm --dir tools/pi-bridge test`.
 * Invariants/Assumptions: `validation.ts` remains strict, matching tool calls beat text fallback, and empty or malformed assistant content still fails.
 */
import test from "node:test";
import assert from "node:assert/strict";
import type { AssistantMessage } from "@mariozechner/pi-ai";
import { AuthStorage, ModelRegistry } from "@mariozechner/pi-coding-agent";
import {
  SDKBackend,
  modelSupportsImages,
  runWithFallback,
  truncateHTMLForPrompt,
} from "./sdk-backend.js";
import {
  CAPABILITY_EXPORT_SHAPE,
  CAPABILITY_EXTRACT_NATURAL,
  CAPABILITY_PIPELINE_JS_GENERATE,
  CAPABILITY_RENDER_PROFILE_GENERATE,
  CAPABILITY_RESEARCH_REFINE,
  CAPABILITY_TEMPLATE_GENERATE,
  CAPABILITY_TRANSFORM_GENERATE,
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
    hasConfiguredAuth(model: FakeModel) {
      return Boolean(apiKeys[model.provider]);
    },
    async getApiKeyAndHeaders(model: FakeModel) {
      const apiKey = apiKeys[model.provider];
      return apiKey
        ? { ok: true as const, apiKey }
        : {
            ok: false as const,
            error: `no auth configured for provider ${model.provider}`,
          };
    },
    getError() {
      return options.loadError;
    },
  } as unknown as ModelRegistry;
}

function createAssistantResponse(options: {
  content: AssistantMessage["content"];
  provider: string;
  model: string;
  tokens?: number;
}) {
  return {
    provider: options.provider,
    model: options.model,
    usage: { totalTokens: options.tokens ?? 123 },
    content: options.content,
  };
}

function createToolResponse(options: {
  toolName: string;
  arguments: Record<string, unknown>;
  provider: string;
  model: string;
  tokens?: number;
}) {
  return createAssistantResponse({
    provider: options.provider,
    model: options.model,
    tokens: options.tokens,
    content: [
      {
        type: "toolCall",
        id: `call-${options.toolName}`,
        name: options.toolName,
        arguments: options.arguments,
      },
    ],
  });
}

function createExtractBackend(content: AssistantMessage["content"]) {
  return new SDKBackend(
    {
      [CAPABILITY_EXTRACT_NATURAL]: ["openai/gpt-5.4"],
    },
    {
      modelRegistry: createFakeModelRegistry({
        models: {
          "openai/gpt-5.4": {
            provider: "openai",
            id: "gpt-5.4",
            input: ["text", "image"],
          },
        },
        apiKeys: {
          openai: "openai-key",
        },
      }),
      completeFn: (async () =>
        createAssistantResponse({
          provider: "openai",
          model: "gpt-5.4",
          content,
        })) as unknown as typeof import("@mariozechner/pi-ai").complete,
    },
  );
}

test("ModelRegistry resolves preferred default pi model IDs", () => {
  const registry = ModelRegistry.inMemory(AuthStorage.inMemory());
  assert.ok(registry.find("kimi-coding", "k2p5"));
  assert.ok(registry.find("zai", "glm-5"));
  assert.ok(registry.find("openai-codex", "gpt-5.4"));
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

test("extract prefers the last valid matching tool call", async () => {
  const backend = createExtractBackend([
    {
      type: "toolCall",
      id: "call-1",
      name: "submit_extraction",
      arguments: {
        fields: { title: "First attempt" },
        confidence: 0.31,
      },
    },
    {
      type: "toolCall",
      id: "call-2",
      name: "submit_extraction",
      arguments: {
        fields: { title: "Corrected attempt" },
        confidence: 0.93,
      },
    },
  ]);

  const result = await backend.extract(CAPABILITY_EXTRACT_NATURAL, {
    html: "<html><h1>Corrected attempt</h1></html>",
    url: "https://example.com/corrected",
    mode: "natural_language",
    prompt: "Extract the title",
  });

  assert.deepEqual(result.fields.title.values, ["Corrected attempt"]);
  assert.equal(result.confidence, 0.93);
});

test("extract skips malformed matching tool calls and uses the later schema-valid one", async () => {
  const backend = createExtractBackend([
    {
      type: "toolCall",
      id: "call-1",
      name: "submit_extraction",
      arguments: {
        fields: { title: "Broken attempt" },
        confidence: "not-a-number",
      },
    },
    {
      type: "toolCall",
      id: "call-2",
      name: "submit_extraction",
      arguments: {
        fields: { title: "Recovered attempt" },
        confidence: 0.89,
      },
    },
  ]);

  const result = await backend.extract(CAPABILITY_EXTRACT_NATURAL, {
    html: "<html><h1>Recovered attempt</h1></html>",
    url: "https://example.com/recovered",
    mode: "natural_language",
    prompt: "Extract the title",
  });

  assert.deepEqual(result.fields.title.values, ["Recovered attempt"]);
  assert.equal(result.confidence, 0.89);
});

test("extract falls back to structured JSON returned in text blocks", async () => {
  const backend = createExtractBackend([
    {
      type: "text",
      text: "```json\n{\n  \"fields\": {\n    \"title\": \"JSON fallback\"\n  },",
    },
    {
      type: "text",
      text: "  \"confidence\": 0.87,\n  \"explanation\": \"Returned as plain JSON text.\"\n}\n```",
    },
  ]);

  const result = await backend.extract(CAPABILITY_EXTRACT_NATURAL, {
    html: "<html><h1>JSON fallback</h1></html>",
    url: "https://example.com/json-fallback",
    mode: "natural_language",
    prompt: "Extract the title",
  });

  assert.deepEqual(result.fields.title.values, ["JSON fallback"]);
  assert.equal(result.confidence, 0.87);
  assert.equal(result.explanation, "Returned as plain JSON text.");
});

test("extract prefers a matching tool call over text fallback when both are present", async () => {
  const backend = createExtractBackend([
    {
      type: "text",
      text: JSON.stringify({
        fields: { title: "Text fallback result" },
        confidence: 0.41,
      }),
    },
    {
      type: "toolCall",
      id: "call-1",
      name: "submit_extraction",
      arguments: {
        fields: { title: "Tool call result" },
        confidence: 0.96,
      },
    },
  ]);

  const result = await backend.extract(CAPABILITY_EXTRACT_NATURAL, {
    html: "<html><h1>Tool call result</h1></html>",
    url: "https://example.com/mixed",
    mode: "natural_language",
    prompt: "Extract the title",
  });

  assert.deepEqual(result.fields.title.values, ["Tool call result"]);
  assert.equal(result.confidence, 0.96);
});

test("extract rejects completely empty assistant content", async () => {
  const backend = createExtractBackend([]);

  await assert.rejects(
    () =>
      backend.extract(CAPABILITY_EXTRACT_NATURAL, {
        html: "<html></html>",
        url: "https://example.com/empty",
        mode: "natural_language",
        prompt: "Extract the title",
      }),
    /model did not call submit_extraction/,
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
    /all routes failed for capability template\.generate after 2 attempts: openai\/gpt-5\.4: model did not call submit_template \| kimi-coding\/k2p5: template result must include at least one selector, jsonld rule, or regex rule/,
  );
});

test("generateRenderProfile validates structured profile output", async () => {
  const backend = new SDKBackend(
    {
      [CAPABILITY_RENDER_PROFILE_GENERATE]: ["openai/gpt-5.4"],
    },
    {
      modelRegistry: createFakeModelRegistry({
        models: {
          "openai/gpt-5.4": { provider: "openai", id: "gpt-5.4", input: ["text", "image"] },
        },
        apiKeys: {
          openai: "openai-key",
        },
      }),
      completeFn: (async () =>
        createToolResponse({
          toolName: "submit_render_profile",
          arguments: {
            profile: {
              preferHeadless: true,
              wait: { mode: "selector", selector: "main" },
            },
            explanation: "Use a headless browser and wait for the main content.",
          },
          provider: "openai",
          model: "gpt-5.4",
        })) as unknown as typeof import("@mariozechner/pi-ai").complete,
    },
  );

  const result = await backend.generateRenderProfile(
    CAPABILITY_RENDER_PROFILE_GENERATE,
    {
      html: "<html><body><main>Widget</main></body></html>",
      url: "https://example.com/widget",
      instructions: "Use headless mode if the main content needs to settle.",
      context_summary: "HTTP fetch returned sparse shell HTML.",
    },
  );

  assert.equal(result.route_id, "openai/gpt-5.4");
  assert.equal(result.provider, "openai");
  assert.equal(result.model, "gpt-5.4");
  assert.equal(result.profile.preferHeadless, true);
});

test("generateRenderProfile uses a default operator goal when instructions are omitted", async () => {
  const backend = new SDKBackend(
    {
      [CAPABILITY_RENDER_PROFILE_GENERATE]: ["openai/gpt-5.4"],
    },
    {
      modelRegistry: createFakeModelRegistry({
        models: {
          "openai/gpt-5.4": { provider: "openai", id: "gpt-5.4", input: ["text", "image"] },
        },
        apiKeys: {
          openai: "openai-key",
        },
      }),
      completeFn: (async (
        _model: FakeModel,
        context: import("@mariozechner/pi-ai").Context,
      ) => {
        const userMessage = context.messages[0];
        assert.equal(userMessage.role, "user");
        assert.equal(typeof userMessage.content, "string");
        assert.match(
          userMessage.content as string,
          /Operator goal: Infer the minimal useful render-profile objective/,
        );
        return createToolResponse({
          toolName: "submit_render_profile",
          arguments: {
            profile: {
              preferHeadless: true,
              wait: { mode: "selector", selector: "main" },
            },
          },
          provider: "openai",
          model: "gpt-5.4",
        });
      }) as unknown as typeof import("@mariozechner/pi-ai").complete,
    },
  );

  const result = await backend.generateRenderProfile(
    CAPABILITY_RENDER_PROFILE_GENERATE,
    {
      html: "<html><body><main>Widget</main></body></html>",
      url: "https://example.com/widget",
      context_summary: "HTTP fetch returned sparse shell HTML.",
    },
  );

  assert.equal(result.profile.preferHeadless, true);
});

test("refineResearch validates structured bounded research output", async () => {
  const backend = new SDKBackend(
    {
      [CAPABILITY_RESEARCH_REFINE]: ["openai/gpt-5.4"],
    },
    {
      modelRegistry: createFakeModelRegistry({
        models: {
          "openai/gpt-5.4": { provider: "openai", id: "gpt-5.4", input: ["text", "image"] },
        },
        apiKeys: {
          openai: "openai-key",
        },
      }),
      completeFn: (async (
        _model: FakeModel,
        context: import("@mariozechner/pi-ai").Context,
      ) => {
        const userMessage = context.messages[0];
        assert.equal(userMessage.role, "user");
        assert.equal(typeof userMessage.content, "string");
        assert.match(
          userMessage.content as string,
          /Research result JSON:/,
        );
        return createToolResponse({
          toolName: "submit_research_refinement",
          arguments: {
            refined: {
              summary:
                "Enterprise pricing appears to be handled through direct sales, with support commitments described in the vendor documentation.",
              conciseSummary: "Direct-sales pricing with documented support commitments.",
              keyFindings: [
                "Pricing is described through sales-led or enterprise channels.",
              ],
              openQuestions: ["Are SLA terms documented publicly?"],
              recommendedNextSteps: [
                "Confirm final SLA details with the vendor sales team.",
              ],
              evidenceHighlights: [
                {
                  url: "https://example.com/pricing",
                  title: "Pricing",
                  finding: "The pricing page routes users to contact sales.",
                  citationUrl: "https://example.com/pricing",
                },
              ],
              confidence: 0.81,
            },
            explanation: "Rewrote the research brief into a tighter operator summary.",
          },
          provider: "openai",
          model: "gpt-5.4",
        });
      }) as unknown as typeof import("@mariozechner/pi-ai").complete,
    },
  );

  const result = await backend.refineResearch(CAPABILITY_RESEARCH_REFINE, {
    result: {
      query: "pricing and support commitments",
      summary: "Original summary",
      evidence: [
        {
          url: "https://example.com/pricing",
          title: "Pricing",
          snippet: "Contact sales for enterprise pricing.",
          citationUrl: "https://example.com/pricing",
        },
      ],
      citations: [
        {
          canonical: "https://example.com/pricing",
          url: "https://example.com/pricing",
        },
      ],
    },
    instructions: "Produce a concise operator-ready brief.",
  });

  assert.equal(result.route_id, "openai/gpt-5.4");
  assert.equal(result.provider, "openai");
  assert.equal(result.model, "gpt-5.4");
  assert.equal(
    result.refined.conciseSummary,
    "Direct-sales pricing with documented support commitments.",
  );
  assert.equal(result.refined.evidenceHighlights?.[0]?.url, "https://example.com/pricing");
});

test("shapeExport validates bounded export shape output", async () => {
  const backend = new SDKBackend(
    {
      [CAPABILITY_EXPORT_SHAPE]: ["openai/gpt-5.4"],
    },
    {
      modelRegistry: createFakeModelRegistry({
        models: {
          "openai/gpt-5.4": { provider: "openai", id: "gpt-5.4", input: ["text", "image"] },
        },
        apiKeys: {
          openai: "openai-key",
        },
      }),
      completeFn: (async (
        _model: FakeModel,
        context: import("@mariozechner/pi-ai").Context,
      ) => {
        const userMessage = context.messages[0];
        assert.equal(userMessage.role, "user");
        assert.equal(typeof userMessage.content, "string");
        assert.match(userMessage.content as string, /Field options:/);
        return createToolResponse({
          toolName: "submit_export_shape",
          arguments: {
            shape: {
              topLevelFields: ["url", "title", "status"],
              normalizedFields: ["field.price"],
              summaryFields: ["title", "field.price"],
              fieldLabels: {
                "field.price": "Price",
              },
              formatting: {
                emptyValue: "—",
                multiValueJoin: "; ",
                markdownTitle: "Pricing Export",
              },
            },
            explanation: "Selected export-ready fields and labels.",
          },
          provider: "openai",
          model: "gpt-5.4",
        });
      }) as unknown as typeof import("@mariozechner/pi-ai").complete,
    },
  );

  const result = await backend.shapeExport(CAPABILITY_EXPORT_SHAPE, {
    jobKind: "scrape",
    format: "md",
    fieldOptions: [
      { key: "url", category: "top_level", label: "URL" },
      { key: "title", category: "top_level", label: "Title" },
      { key: "status", category: "top_level", label: "Status" },
      { key: "field.price", category: "normalized", label: "Price" },
    ],
    instructions: "Focus on pricing fields.",
  });

  assert.equal(result.route_id, "openai/gpt-5.4");
  assert.equal(result.provider, "openai");
  assert.equal(result.model, "gpt-5.4");
  assert.deepEqual(result.shape.topLevelFields, ["url", "title", "status"]);
  assert.deepEqual(result.shape.normalizedFields, ["field.price"]);
  assert.equal(result.shape.formatting?.markdownTitle, "Pricing Export");
});

test("generateTransform validates bounded transform output", async () => {
  const backend = new SDKBackend(
    {
      [CAPABILITY_TRANSFORM_GENERATE]: ["openai/gpt-5.4"],
    },
    {
      modelRegistry: createFakeModelRegistry({
        models: {
          "openai/gpt-5.4": { provider: "openai", id: "gpt-5.4", input: ["text", "image"] },
        },
        apiKeys: {
          openai: "openai-key",
        },
      }),
      completeFn: (async (
        _model: FakeModel,
        context: import("@mariozechner/pi-ai").Context,
      ) => {
        const userMessage = context.messages[0];
        assert.equal(userMessage.role, "user");
        assert.equal(typeof userMessage.content, "string");
        assert.match(userMessage.content as string, /Sample records:/);
        return createToolResponse({
          toolName: "submit_transform",
          arguments: {
            transform: {
              expression: "{title: title, url: url}",
              language: "jmespath",
            },
            explanation: "Projected the high-signal fields needed for export.",
          },
          provider: "openai",
          model: "gpt-5.4",
        });
      }) as unknown as typeof import("@mariozechner/pi-ai").complete,
    },
  );

  const result = await backend.generateTransform(CAPABILITY_TRANSFORM_GENERATE, {
    jobKind: "crawl",
    sampleFields: [
      { path: "url", sampleValues: ["https://example.com"] },
      { path: "title", sampleValues: ["Example"] },
    ],
    sampleRecords: [
      {
        url: "https://example.com",
        title: "Example",
        status: 200,
      },
    ],
    preferredLanguage: "jmespath",
    instructions: "Project the URL and title for a lightweight export.",
  });

  assert.equal(result.route_id, "openai/gpt-5.4");
  assert.equal(result.provider, "openai");
  assert.equal(result.model, "gpt-5.4");
  assert.equal(result.transform.language, "jmespath");
  assert.equal(result.transform.expression, "{title: title, url: url}");
});

test("generatePipelineJs sends screenshot context as multimodal user content", async () => {
  const backend = new SDKBackend(
    {
      [CAPABILITY_PIPELINE_JS_GENERATE]: ["openai/gpt-5.4"],
    },
    {
      modelRegistry: createFakeModelRegistry({
        models: {
          "openai/gpt-5.4": { provider: "openai", id: "gpt-5.4", input: ["text", "image"] },
        },
        apiKeys: {
          openai: "openai-key",
        },
      }),
      completeFn: (async (
        _model: FakeModel,
        context: import("@mariozechner/pi-ai").Context,
      ) => {
        const userMessage = context.messages[0];
        assert.equal(userMessage.role, "user");
        assert.ok(Array.isArray(userMessage.content));
        assert.equal(userMessage.content[0]?.type, "text");
        assert.equal(userMessage.content[1]?.type, "image");
        return createToolResponse({
          toolName: "submit_pipeline_js",
          arguments: {
            script: {
              selectors: ["main", "[data-ready='true']"],
              postNav: "window.scrollTo(0, 0);",
            },
            explanation: "Wait for the main container and normalize scroll position.",
          },
          provider: "openai",
          model: "gpt-5.4",
        });
      }) as unknown as typeof import("@mariozechner/pi-ai").complete,
    },
  );

  const result = await backend.generatePipelineJs(
    CAPABILITY_PIPELINE_JS_GENERATE,
    {
      html: "<html><body><main data-ready='true'>Widget</main></body></html>",
      url: "https://example.com/widget",
      instructions: "Wait for the main app shell and scroll back to the top.",
      images: [{ data: "ZmFrZQ==", mime_type: "image/png" }],
    },
  );

  assert.equal(result.route_id, "openai/gpt-5.4");
  assert.deepEqual(result.script.selectors, ["main", "[data-ready='true']"]);
});

test("generatePipelineJs uses a default operator goal when instructions are omitted", async () => {
  const backend = new SDKBackend(
    {
      [CAPABILITY_PIPELINE_JS_GENERATE]: ["openai/gpt-5.4"],
    },
    {
      modelRegistry: createFakeModelRegistry({
        models: {
          "openai/gpt-5.4": { provider: "openai", id: "gpt-5.4", input: ["text", "image"] },
        },
        apiKeys: {
          openai: "openai-key",
        },
      }),
      completeFn: (async (
        _model: FakeModel,
        context: import("@mariozechner/pi-ai").Context,
      ) => {
        const userMessage = context.messages[0];
        assert.equal(userMessage.role, "user");
        assert.equal(typeof userMessage.content, "string");
        assert.match(
          userMessage.content as string,
          /Operator goal: Infer the minimal useful pipeline-JS objective/,
        );
        return createToolResponse({
          toolName: "submit_pipeline_js",
          arguments: {
            script: {
              selectors: ["main"],
            },
          },
          provider: "openai",
          model: "gpt-5.4",
        });
      }) as unknown as typeof import("@mariozechner/pi-ai").complete,
    },
  );

  const result = await backend.generatePipelineJs(
    CAPABILITY_PIPELINE_JS_GENERATE,
    {
      html: "<html><body><main>Widget</main></body></html>",
      url: "https://example.com/widget",
      context_summary: "JS heaviness suggests a browser-driven page shell.",
    },
  );

  assert.deepEqual(result.script.selectors, ["main"]);
});

test("extract sends screenshot context as multimodal user content", async () => {
  const backend = new SDKBackend(
    {
      [CAPABILITY_EXTRACT_NATURAL]: ["openai/gpt-5.4"],
    },
    {
      modelRegistry: createFakeModelRegistry({
        models: {
          "openai/gpt-5.4": { provider: "openai", id: "gpt-5.4", input: ["text", "image"] },
        },
        apiKeys: {
          openai: "openai-key",
        },
      }),
      completeFn: (async (
        _model: FakeModel,
        context: import("@mariozechner/pi-ai").Context,
      ) => {
        const userMessage = context.messages[0];
        assert.equal(userMessage.role, "user");
        assert.ok(Array.isArray(userMessage.content));
        assert.equal(userMessage.content[0]?.type, "text");
        assert.equal(userMessage.content[1]?.type, "image");
        return createToolResponse({
          toolName: "submit_extraction",
          arguments: {
            fields: { title: "Widget" },
            confidence: 0.95,
          },
          provider: "openai",
          model: "gpt-5.4",
        });
      }) as unknown as typeof import("@mariozechner/pi-ai").complete,
    },
  );

  const result = await backend.extract(CAPABILITY_EXTRACT_NATURAL, {
    html: "<html><h1>Widget</h1></html>",
    url: "https://example.com/widget",
    mode: "natural_language",
    prompt: "Extract the title",
    images: [{ data: "ZmFrZQ==", mime_type: "image/png" }],
  });

  assert.equal(result.route_id, "openai/gpt-5.4");
  assert.deepEqual(result.fields.title.values, ["Widget"]);
});

test("modelSupportsImages matches current verified model capabilities", () => {
  const registry = ModelRegistry.inMemory(AuthStorage.inMemory());
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
