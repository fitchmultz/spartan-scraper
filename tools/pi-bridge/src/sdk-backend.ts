/**
 * Purpose: Execute SDK-backed pi-bridge capability calls against configured LLM routes.
 * Responsibilities: Select routes, build prompts/tools, tolerate valid stochastic response shapes, and normalize validated results.
 * Scope: SDK-backed bridge execution only.
 * Usage: Instantiated by `main.ts` when bridge mode is `sdk`.
 * Invariants/Assumptions: `validation.ts` remains the strict result boundary, matching tool calls beat text fallback, and text fallback only runs when no matching tool call exists.
 */
import {
  Type,
  complete,
  validateToolCall,
  type AssistantMessage,
  type Tool,
  type ToolCall,
  type Context,
  type ImageContent,
  type TextContent,
} from "@mariozechner/pi-ai";
import { AuthStorage, ModelRegistry } from "@mariozechner/pi-coding-agent";
import type {
  ExportShapePayload,
  ExportShapeResult,
  ExtractPayload,
  GeneratePipelineJsPayload,
  GenerateRenderProfilePayload,
  GenerateTemplatePayload,
  GenerateTransformPayload,
  HealthResult,
  HealthRouteStatus,
  PipelineJsResult,
  RenderProfileResult,
  ResearchRefinePayload,
  ResearchRefineResult,
  TemplateResult,
  TransformResult,
} from "./protocol.js";
import { parseRouteId } from "./config.js";
import {
  normalizeExtractResult,
  validateExportShapeResult,
  validatePipelineJsResult,
  validateRenderProfileResult,
  validateResearchRefineResult,
  validateTemplateResult,
  validateTransformResult,
} from "./validation.js";

export async function runWithFallback<T>(
  routes: string[],
  invoke: (routeId: string) => Promise<T>,
  options: { capability?: string } = {},
): Promise<T> {
  if (routes.length === 0) {
    const label = options.capability
      ? ` for capability ${options.capability}`
      : "";
    throw new Error(`no routes configured${label}`);
  }

  const errors: string[] = [];
  for (const routeId of routes) {
    try {
      return await invoke(routeId);
    } catch (error) {
      errors.push(
        `${routeId}: ${error instanceof Error ? error.message : String(error)}`,
      );
    }
  }

  const label = options.capability ? ` for capability ${options.capability}` : "";
  throw new Error(
    `all routes failed${label} after ${routes.length} attempt${routes.length === 1 ? "" : "s"}: ${errors.join(" | ")}`,
  );
}

export class SDKBackend {
  readonly authStorage: AuthStorage;
  readonly modelRegistry: ModelRegistry;
  private readonly completeFn: typeof complete;

  constructor(
    private readonly routes: Record<string, string[]>,
    deps: {
      authStorage?: AuthStorage;
      modelRegistry?: ModelRegistry;
      completeFn?: typeof complete;
    } = {},
  ) {
    this.authStorage =
      deps.modelRegistry?.authStorage ?? deps.authStorage ?? AuthStorage.create();
    this.modelRegistry = deps.modelRegistry ?? new ModelRegistry(this.authStorage);
    this.completeFn = deps.completeFn ?? complete;
  }

  async health(agentDir: string): Promise<HealthResult> {
    const available: Record<string, string[]> = {};
    const routeStatus: Record<string, HealthRouteStatus[]> = {};

    for (const [capability, routeIds] of Object.entries(this.routes)) {
      const statuses = routeIds.map((routeId) => this.inspectRoute(routeId));
      routeStatus[capability] = statuses;
      available[capability] = statuses
        .filter((status) => status.status === "ready")
        .map((status) => status.route_id);
    }

    const authErrors = this.authStorage
      .drainErrors()
      .map((error) => error.message)
      .filter(Boolean);

    return {
      mode: "sdk",
      agent_dir: agentDir,
      resolved: this.routes,
      available,
      route_status: routeStatus,
      load_error: this.modelRegistry.getError(),
      auth_errors: authErrors.length > 0 ? authErrors : undefined,
    };
  }

  async extract(capability: string, payload: ExtractPayload) {
    return runWithFallback(
      this.routes[capability] || [],
      async (routeId) => {
        const selection = await this.selectRoute(routeId, {
          requiresImage: Boolean(payload.images?.length),
        });
        const tool = this.extractTool();
        const response = await this.completeFn(
          selection.model,
          this.buildContext({
            userPrompt: buildExtractPrompt(payload),
            images: payload.images,
            systemPrompt:
              "You extract structured data from HTML. Call the submit_extraction tool once with concise, precise field values.",
            tools: [tool],
          }),
          {
            apiKey: selection.apiKey,
            maxTokens: 4096,
            temperature: 0,
          },
        );

        const call = getLastValidToolCall(response.content, tool);
        const args = validateToolCall([tool], call);
        return normalizeExtractResult(args, {
          route_id: routeId,
          provider: response.provider,
          model: response.model,
          tokens_used: response.usage.totalTokens,
        });
      },
      { capability },
    );
  }

  async generateTemplate(
    capability: string,
    payload: GenerateTemplatePayload,
  ): Promise<TemplateResult> {
    return runWithFallback(
      this.routes[capability] || [],
      async (routeId) => {
        const selection = await this.selectRoute(routeId, {
          requiresImage: Boolean(payload.images?.length),
        });
        const tool = this.templateTool();
        const response = await this.completeFn(
          selection.model,
          this.buildContext({
            userPrompt: buildTemplatePrompt(payload),
            images: payload.images,
            systemPrompt:
              "You generate extraction templates from HTML. Use the submit_template tool to submit your result. Prefer robust CSS selectors and only use jsonld/regex when they add real value.",
            tools: [tool],
          }),
          {
            apiKey: selection.apiKey,
            maxTokens: 4096,
            temperature: 0,
          },
        );

        const call = getLastValidToolCall(response.content, tool);
        const args = validateToolCall([tool], call);
        return validateTemplateResult({
          ...(args as TemplateResult),
          route_id: routeId,
          provider: response.provider,
          model: response.model,
        });
      },
      { capability },
    );
  }

  async generateRenderProfile(
    capability: string,
    payload: GenerateRenderProfilePayload,
  ): Promise<RenderProfileResult> {
    return runWithFallback(
      this.routes[capability] || [],
      async (routeId) => {
        const selection = await this.selectRoute(routeId, {
          requiresImage: Boolean(payload.images?.length),
        });
        const tool = this.renderProfileTool();
        const response = await this.completeFn(
          selection.model,
          this.buildContext({
            userPrompt: buildRenderProfilePrompt(payload),
            images: payload.images,
            systemPrompt:
              "You author Spartan render profile patches for difficult sites. Use the submit_render_profile tool to submit your result. Prefer omission over speculative settings and only set fields that materially improve fetch behavior.",
            tools: [tool],
          }),
          {
            apiKey: selection.apiKey,
            maxTokens: 4096,
            temperature: 0,
          },
        );

        const call = getLastValidToolCall(response.content, tool);
        const args = validateToolCall([tool], call);
        return validateRenderProfileResult({
          ...(args as RenderProfileResult),
          route_id: routeId,
          provider: response.provider,
          model: response.model,
        });
      },
      { capability },
    );
  }

  async generatePipelineJs(
    capability: string,
    payload: GeneratePipelineJsPayload,
  ): Promise<PipelineJsResult> {
    return runWithFallback(
      this.routes[capability] || [],
      async (routeId) => {
        const selection = await this.selectRoute(routeId, {
          requiresImage: Boolean(payload.images?.length),
        });
        const tool = this.pipelineJsTool();
        const response = await this.completeFn(
          selection.model,
          this.buildContext({
            userPrompt: buildPipelineJsPrompt(payload),
            images: payload.images,
            systemPrompt:
              "You author Spartan pipeline JS scripts for page automation. Use the submit_pipeline_js tool to submit your result. Keep scripts focused, deterministic, and minimal; prefer wait selectors unless JavaScript is clearly necessary.",
            tools: [tool],
          }),
          {
            apiKey: selection.apiKey,
            maxTokens: 4096,
            temperature: 0,
          },
        );

        const call = getLastValidToolCall(response.content, tool);
        const args = validateToolCall([tool], call);
        return validatePipelineJsResult({
          ...(args as PipelineJsResult),
          route_id: routeId,
          provider: response.provider,
          model: response.model,
        });
      },
      { capability },
    );
  }

  async refineResearch(
    capability: string,
    payload: ResearchRefinePayload,
  ): Promise<ResearchRefineResult> {
    return runWithFallback(
      this.routes[capability] || [],
      async (routeId) => {
        const selection = await this.selectRoute(routeId, {
          requiresImage: false,
        });
        const tool = this.researchRefineTool();
        const response = await this.completeFn(
          selection.model,
          this.buildContext({
            userPrompt: buildResearchRefinePrompt(payload),
            systemPrompt:
              "You refine bounded Spartan research outputs. Use only the supplied research result. Do not invent sources, URLs, or evidence. Use the submit_research_refinement tool to submit your result with a grounded rewrite that preserves uncertainty.",
            tools: [tool],
          }),
          {
            apiKey: selection.apiKey,
            maxTokens: 4096,
            temperature: 0,
          },
        );

        const call = getLastValidToolCall(response.content, tool);
        const args = validateToolCall([tool], call);
        return validateResearchRefineResult({
          ...(args as ResearchRefineResult),
          route_id: routeId,
          provider: response.provider,
          model: response.model,
        });
      },
      { capability },
    );
  }

  async shapeExport(
    capability: string,
    payload: ExportShapePayload,
  ): Promise<ExportShapeResult> {
    return runWithFallback(
      this.routes[capability] || [],
      async (routeId) => {
        const selection = await this.selectRoute(routeId, {
          requiresImage: false,
        });
        const tool = this.exportShapeTool();
        const response = await this.completeFn(
          selection.model,
          this.buildContext({
            userPrompt: buildExportShapePrompt(payload),
            systemPrompt:
              "You shape bounded Spartan export configurations. Use only the supplied field catalog and existing shape. Do not invent field keys. Use the submit_export_shape tool to submit your result as a deterministic export shape configuration.",
            tools: [tool],
          }),
          {
            apiKey: selection.apiKey,
            maxTokens: 4096,
            temperature: 0,
          },
        );

        const call = getLastValidToolCall(response.content, tool);
        const args = validateToolCall([tool], call);
        return validateExportShapeResult({
          ...(args as ExportShapeResult),
          route_id: routeId,
          provider: response.provider,
          model: response.model,
        });
      },
      { capability },
    );
  }

  async generateTransform(
    capability: string,
    payload: GenerateTransformPayload,
  ): Promise<TransformResult> {
    return runWithFallback(
      this.routes[capability] || [],
      async (routeId) => {
        const selection = await this.selectRoute(routeId, {
          requiresImage: false,
        });
        const tool = this.transformTool();
        const response = await this.completeFn(
          selection.model,
          this.buildContext({
            userPrompt: buildTransformPrompt(payload),
            systemPrompt:
              "You author bounded Spartan result transformations. Use only the supplied sample records, field paths, and optional current transform. Do not invent fields outside the provided samples. Use the submit_transform tool to submit your result as a deterministic JMESPath or JSONata transform.",
            tools: [tool],
          }),
          {
            apiKey: selection.apiKey,
            maxTokens: 4096,
            temperature: 0,
          },
        );

        const call = getLastValidToolCall(response.content, tool);
        const args = validateToolCall([tool], call);
        return validateTransformResult({
          ...(args as TransformResult),
          route_id: routeId,
          provider: response.provider,
          model: response.model,
        });
      },
      { capability },
    );
  }

  private inspectRoute(routeId: string): HealthRouteStatus {
    try {
      const { provider, model } = parseRouteId(routeId);
      const selectedModel = this.modelRegistry.find(provider, model);
      if (!selectedModel) {
        return {
          route_id: routeId,
          provider,
          model,
          status: "missing_model",
          message: `model not found for route ${routeId}`,
          model_found: false,
          auth_configured: false,
        };
      }

      const authConfigured = this.authStorage.hasAuth(provider);
      if (!authConfigured) {
        return {
          route_id: routeId,
          provider,
          model,
          status: "missing_auth",
          message: `no auth configured for provider ${provider}`,
          model_found: true,
          auth_configured: false,
        };
      }

      return {
        route_id: routeId,
        provider,
        model,
        status: "ready",
        model_found: true,
        auth_configured: true,
      };
    } catch (error) {
      return {
        route_id: routeId,
        status: "invalid_route",
        message: error instanceof Error ? error.message : String(error),
        model_found: false,
        auth_configured: false,
      };
    }
  }

  private async selectRoute(
    routeId: string,
    requirements: { requiresImage: boolean },
  ) {
    const { provider, model } = parseRouteId(routeId);
    const selectedModel = this.modelRegistry.find(provider, model);
    if (!selectedModel) {
      throw new Error(`model not found for route ${routeId}`);
    }
    if (requirements.requiresImage && !modelSupportsImages(selectedModel)) {
      throw new Error(`model ${routeId} does not support image input`);
    }
    const apiKey = await this.modelRegistry.getApiKey(selectedModel);
    if (!apiKey) {
      throw new Error(`no auth available for provider ${provider}`);
    }
    return { model: selectedModel, apiKey };
  }

  private buildContext(input: {
    systemPrompt: string;
    userPrompt: string;
    images?: Array<{ data: string; mime_type: string }>;
    tools: Tool[];
  }): Context {
    return {
      systemPrompt: input.systemPrompt,
      tools: input.tools,
      messages: [
        {
          role: "user",
          content: buildUserContent(input.userPrompt, input.images),
          timestamp: Date.now(),
        },
      ],
    };
  }

  private extractTool(): Tool {
    return {
      name: "submit_extraction",
      description: "Submit extracted structured fields for the requested page.",
      parameters: Type.Object({
        fields: Type.Record(
          Type.String(),
          Type.Any(),
        ),
        confidence: Type.Number(),
        explanation: Type.Optional(Type.String()),
      }),
    };
  }

  private templateTool(): Tool {
    return {
      name: "submit_template",
      description: "Submit an extraction template tailored to the provided HTML.",
      parameters: Type.Object({
        template: Type.Object({
          name: Type.String(),
          selectors: Type.Array(
            Type.Object({
              name: Type.String(),
              selector: Type.String(),
              attr: Type.Optional(Type.String()),
              all: Type.Optional(Type.Boolean()),
              join: Type.Optional(Type.String()),
              trim: Type.Optional(Type.Boolean()),
              required: Type.Optional(Type.Boolean()),
            }),
          ),
          jsonld: Type.Optional(
            Type.Array(
              Type.Object({
                name: Type.String(),
                type: Type.Optional(Type.String()),
                path: Type.Optional(Type.String()),
                all: Type.Optional(Type.Boolean()),
                required: Type.Optional(Type.Boolean()),
              }),
            ),
          ),
          regex: Type.Optional(
            Type.Array(
              Type.Object({
                name: Type.String(),
                pattern: Type.String(),
                group: Type.Optional(Type.Number()),
                all: Type.Optional(Type.Boolean()),
                source: Type.Optional(Type.String()),
                required: Type.Optional(Type.Boolean()),
              }),
            ),
          ),
          normalize: Type.Optional(
            Type.Object({
              titleField: Type.Optional(Type.String()),
              descriptionField: Type.Optional(Type.String()),
              textField: Type.Optional(Type.String()),
              metaFields: Type.Optional(Type.Record(Type.String(), Type.String())),
            }),
          ),
        }),
        explanation: Type.Optional(Type.String()),
      }),
    };
  }

  private renderProfileTool(): Tool {
    return {
      name: "submit_render_profile",
      description: "Submit a Spartan render profile patch for the provided page.",
      parameters: Type.Object({
        profile: Type.Object({
          forceEngine: Type.Optional(Type.String()),
          preferHeadless: Type.Optional(Type.Boolean()),
          assumeJsHeavy: Type.Optional(Type.Boolean()),
          neverHeadless: Type.Optional(Type.Boolean()),
          jsHeavyThreshold: Type.Optional(Type.Number()),
          rateLimitQPS: Type.Optional(Type.Number()),
          rateLimitBurst: Type.Optional(Type.Number()),
          block: Type.Optional(
            Type.Object({
              resourceTypes: Type.Optional(Type.Array(Type.String())),
              urlPatterns: Type.Optional(Type.Array(Type.String())),
            }),
          ),
          wait: Type.Optional(
            Type.Object({
              mode: Type.Optional(Type.String()),
              selector: Type.Optional(Type.String()),
              networkIdleQuietMs: Type.Optional(Type.Number()),
              minTextLength: Type.Optional(Type.Number()),
              stabilityPollMs: Type.Optional(Type.Number()),
              stabilityIterations: Type.Optional(Type.Number()),
              extraSleepMs: Type.Optional(Type.Number()),
            }),
          ),
          timeouts: Type.Optional(
            Type.Object({
              maxRenderMs: Type.Optional(Type.Number()),
              scriptEvalMs: Type.Optional(Type.Number()),
              navigationMs: Type.Optional(Type.Number()),
            }),
          ),
          screenshot: Type.Optional(
            Type.Object({
              enabled: Type.Optional(Type.Boolean()),
              fullPage: Type.Optional(Type.Boolean()),
              format: Type.Optional(Type.String()),
              quality: Type.Optional(Type.Number()),
              width: Type.Optional(Type.Number()),
              height: Type.Optional(Type.Number()),
            }),
          ),
        }),
        explanation: Type.Optional(Type.String()),
      }),
    };
  }

  private pipelineJsTool(): Tool {
    return {
      name: "submit_pipeline_js",
      description: "Submit a Spartan pipeline JS script for the provided page.",
      parameters: Type.Object({
        script: Type.Object({
          engine: Type.Optional(Type.String()),
          preNav: Type.Optional(Type.String()),
          postNav: Type.Optional(Type.String()),
          selectors: Type.Optional(Type.Array(Type.String())),
        }),
        explanation: Type.Optional(Type.String()),
      }),
    };
  }

  private researchRefineTool(): Tool {
    return {
      name: "submit_research_refinement",
      description:
        "Submit a grounded refinement of the provided Spartan research result.",
      parameters: Type.Object({
        refined: Type.Object({
          summary: Type.String(),
          conciseSummary: Type.String(),
          keyFindings: Type.Array(Type.String()),
          openQuestions: Type.Optional(Type.Array(Type.String())),
          recommendedNextSteps: Type.Optional(Type.Array(Type.String())),
          evidenceHighlights: Type.Optional(
            Type.Array(
              Type.Object({
                url: Type.String(),
                title: Type.Optional(Type.String()),
                finding: Type.String(),
                relevance: Type.Optional(Type.String()),
                citationUrl: Type.Optional(Type.String()),
              }),
            ),
          ),
          confidence: Type.Optional(Type.Number()),
        }),
        explanation: Type.Optional(Type.String()),
      }),
    };
  }

  private exportShapeTool(): Tool {
    return {
      name: "submit_export_shape",
      description:
        "Submit a deterministic Spartan export shape using only the supplied field catalog.",
      parameters: Type.Object({
        shape: Type.Object({
          topLevelFields: Type.Optional(Type.Array(Type.String())),
          normalizedFields: Type.Optional(Type.Array(Type.String())),
          evidenceFields: Type.Optional(Type.Array(Type.String())),
          summaryFields: Type.Optional(Type.Array(Type.String())),
          fieldLabels: Type.Optional(Type.Record(Type.String(), Type.String())),
          formatting: Type.Optional(
            Type.Object({
              emptyValue: Type.Optional(Type.String()),
              multiValueJoin: Type.Optional(Type.String()),
              markdownTitle: Type.Optional(Type.String()),
            }),
          ),
        }),
        explanation: Type.Optional(Type.String()),
      }),
    };
  }

  private transformTool(): Tool {
    return {
      name: "submit_transform",
      description:
        "Submit a deterministic Spartan result transform using JMESPath or JSONata.",
      parameters: Type.Object({
        transform: Type.Object({
          expression: Type.String(),
          language: Type.String(),
        }),
        explanation: Type.Optional(Type.String()),
      }),
    };
  }
}

export function modelSupportsImages(model: { input: string[] }): boolean {
  return model.input.includes("image");
}

type AssistantContent = AssistantMessage["content"];

const WORD_BOUNDARY_THRESHOLD = 0.8;

function getLastValidToolCall(content: AssistantContent, tool: Tool): ToolCall {
  const matchingToolCalls = content.filter(
    (block): block is ToolCall =>
      block.type === "toolCall" &&
      typeof block.id === "string" &&
      typeof block.name === "string" &&
      block.name === tool.name &&
      !!block.arguments &&
      typeof block.arguments === "object" &&
      !Array.isArray(block.arguments),
  );

  if (matchingToolCalls.length > 0) {
    for (let index = matchingToolCalls.length - 1; index >= 0; index -= 1) {
      const candidate = matchingToolCalls[index];
      try {
        validateToolCall([tool], candidate);
        return candidate;
      } catch {
        // Models can self-correct with a later tool call; keep searching backwards.
      }
    }

    return matchingToolCalls[matchingToolCalls.length - 1];
  }

  const fallbackToolCall = getStructuredResponseFallback(content, tool);
  if (fallbackToolCall) {
    return fallbackToolCall;
  }

  throw new Error(`model did not call ${tool.name}`);
}

function getStructuredResponseFallback(
  content: AssistantContent,
  tool: Tool,
): ToolCall | null {
  const textBlocks = content.filter(
    (block): block is TextContent =>
      block.type === "text" &&
      typeof block.text === "string" &&
      block.text.trim().length > 0,
  );
  if (textBlocks.length === 0) {
    return null;
  }

  const candidates = textBlocks
    .slice()
    .reverse()
    .map((block) => block.text);
  if (textBlocks.length > 1) {
    candidates.push(textBlocks.map((block) => block.text).join("\n"));
  }

  for (const candidate of candidates) {
    const parsed = tryParseStructuredJson(candidate);
    if (!parsed) {
      continue;
    }

    const syntheticToolCall: ToolCall = {
      type: "toolCall",
      id: `text-fallback-${tool.name}`,
      name: tool.name,
      arguments: parsed,
    };

    try {
      validateToolCall([tool], syntheticToolCall);
      return syntheticToolCall;
    } catch {
      // Ignore JSON blobs that do not satisfy the tool schema.
    }
  }

  return null;
}

function tryParseStructuredJson(text: string): Record<string, unknown> | null {
  const trimmed = text.trim();
  if (!trimmed) {
    return null;
  }

  const candidates = [
    trimmed,
    stripMarkdownCodeFence(trimmed),
    extractJSONObject(stripMarkdownCodeFence(trimmed)),
  ].filter((candidate): candidate is string => Boolean(candidate?.trim()));

  for (const candidate of candidates) {
    try {
      const parsed = JSON.parse(candidate);
      if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
        return parsed as Record<string, unknown>;
      }
    } catch {
      // Keep trying alternate structured text shapes.
    }
  }

  return null;
}

function stripMarkdownCodeFence(text: string): string {
  const match = text.match(/^```(?:json)?\s*([\s\S]*?)\s*```$/i);
  return match?.[1] ?? text;
}

function extractJSONObject(text: string): string | null {
  const start = text.indexOf("{");
  const end = text.lastIndexOf("}");
  if (start === -1 || end === -1 || end <= start) {
    return null;
  }
  return text.slice(start, end + 1);
}

function buildExtractPrompt(payload: ExtractPayload): string {
  const html = truncateHTMLForPrompt(payload.html, payload.max_content_chars);
  const parts: string[] = [
    `URL: ${payload.url}`,
    `Mode: ${payload.mode}`,
  ];
  if (payload.prompt?.trim()) {
    parts.push(`Instructions: ${payload.prompt.trim()}`);
  }
  if (payload.schema_example && Object.keys(payload.schema_example).length > 0) {
    parts.push(
      `Schema example:\n${JSON.stringify(payload.schema_example, null, 2)}`,
    );
  }
  if (payload.fields?.length) {
    parts.push(`Fields: ${payload.fields.join(", ")}`);
  }
  if (payload.images?.length) {
    parts.push("A screenshot is attached. Use it as supplemental visual context for layout, visibility, and extraction accuracy.");
  }
  parts.push(`HTML:\n${html}`);
  return parts.join("\n\n");
}

function buildTemplatePrompt(payload: GenerateTemplatePayload): string {
  const parts: string[] = [
    `URL: ${payload.url}`,
    `Goal: ${payload.description}`,
  ];
  if (payload.sample_fields?.length) {
    parts.push(`Sample fields: ${payload.sample_fields.join(", ")}`);
  }
  if (payload.feedback?.trim()) {
    parts.push(`Validation feedback: ${payload.feedback.trim()}`);
  }
  if (payload.images?.length) {
    parts.push("A screenshot is attached. Use it as supplemental visual context for layout, visibility, and selector robustness.");
  }
  parts.push(`HTML:\n${payload.html}`);
  return parts.join("\n\n");
}

function buildRenderProfilePrompt(payload: GenerateRenderProfilePayload): string {
  const parts: string[] = [
    `URL: ${payload.url}`,
    `Operator goal: ${payload.instructions}`,
  ];
  if (payload.context_summary?.trim()) {
    parts.push(`Context summary:\n${payload.context_summary.trim()}`);
  }
  if (payload.feedback?.trim()) {
    parts.push(`Validation feedback: ${payload.feedback.trim()}`);
  }
  if (payload.images?.length) {
    parts.push("A screenshot is attached. Use it as supplemental visual context for choosing wait strategy, browser engine, and other fetch settings.");
  }
  parts.push(`HTML:\n${payload.html}`);
  return parts.join("\n\n");
}

function buildPipelineJsPrompt(payload: GeneratePipelineJsPayload): string {
  const parts: string[] = [
    `URL: ${payload.url}`,
    `Operator goal: ${payload.instructions}`,
  ];
  if (payload.context_summary?.trim()) {
    parts.push(`Context summary:\n${payload.context_summary.trim()}`);
  }
  if (payload.feedback?.trim()) {
    parts.push(`Validation feedback: ${payload.feedback.trim()}`);
  }
  if (payload.images?.length) {
    parts.push("A screenshot is attached. Use it as supplemental visual context for deciding selectors, waits, and any necessary browser-side automation.");
  }
  parts.push(`HTML:\n${payload.html}`);
  return parts.join("\n\n");
}

function buildResearchRefinePrompt(payload: ResearchRefinePayload): string {
  const parts: string[] = [
    `Research query: ${payload.result.query?.trim() || "(missing)"}`,
  ];
  if (payload.instructions?.trim()) {
    parts.push(`Operator goal: ${payload.instructions.trim()}`);
  }
  if (payload.feedback?.trim()) {
    parts.push(`Validation feedback: ${payload.feedback.trim()}`);
  }
  parts.push(
    "Refine only the supplied research output. Stay grounded in the provided evidence and preserve uncertainty where the evidence is incomplete.",
  );
  parts.push(
    `Research result JSON:\n${JSON.stringify(payload.result, null, 2)}`,
  );
  return parts.join("\n\n");
}

function buildExportShapePrompt(payload: ExportShapePayload): string {
  const parts: string[] = [
    `Job kind: ${payload.jobKind}`,
    `Target export format: ${payload.format}`,
  ];
  if (payload.instructions?.trim()) {
    parts.push(`Operator goal: ${payload.instructions.trim()}`);
  }
  if (payload.feedback?.trim()) {
    parts.push(`Validation feedback: ${payload.feedback.trim()}`);
  }
  parts.push(
    "Use only the supplied field options. Do not invent field keys or categories.",
  );
  if (payload.currentShape && Object.keys(payload.currentShape).length > 0) {
    parts.push(`Current shape:\n${JSON.stringify(payload.currentShape, null, 2)}`);
  }
  parts.push(
    `Field options:\n${JSON.stringify(payload.fieldOptions ?? [], null, 2)}`,
  );
  return parts.join("\n\n");
}

function buildTransformPrompt(payload: GenerateTransformPayload): string {
  const parts: string[] = [];
  if (payload.jobKind?.trim()) {
    parts.push(`Job kind: ${payload.jobKind.trim()}`);
  }
  if (payload.preferredLanguage?.trim()) {
    parts.push(`Preferred language: ${payload.preferredLanguage.trim()}`);
  }
  if (payload.instructions?.trim()) {
    parts.push(`Operator goal: ${payload.instructions.trim()}`);
  }
  if (payload.feedback?.trim()) {
    parts.push(`Validation feedback: ${payload.feedback.trim()}`);
  }
  parts.push(
    "Use only the supplied sample records and field paths. Do not invent fields outside the provided result structure.",
  );
  if (payload.currentTransform && Object.keys(payload.currentTransform).length > 0) {
    parts.push(
      `Current transform:\n${JSON.stringify(payload.currentTransform, null, 2)}`,
    );
  }
  if (payload.sampleFields?.length) {
    parts.push(`Sample fields:\n${JSON.stringify(payload.sampleFields, null, 2)}`);
  }
  parts.push(
    `Sample records:\n${JSON.stringify(payload.sampleRecords ?? [], null, 2)}`,
  );
  return parts.join("\n\n");
}

function buildUserContent(
  userPrompt: string,
  images?: Array<{ data: string; mime_type: string }>,
): string | (TextContent | ImageContent)[] {
  if (!images?.length) {
    return userPrompt;
  }
  return [
    { type: "text", text: userPrompt },
    ...images.map((image) => ({
      type: "image" as const,
      data: image.data,
      mimeType: image.mime_type,
    })),
  ];
}

export function truncateHTMLForPrompt(html: string, maxChars?: number): string {
  if (!maxChars || maxChars <= 0 || html.length <= maxChars) {
    return html;
  }

  let truncated = html.slice(0, maxChars);
  const lastSpace = truncated.lastIndexOf(" ");
  if (lastSpace > maxChars * WORD_BOUNDARY_THRESHOLD) {
    truncated = truncated.slice(0, lastSpace);
  }

  return `${truncated}...`;
}
