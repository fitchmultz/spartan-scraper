import {
  Type,
  complete,
  validateToolCall,
  type Tool,
  type ToolCall,
  type Context,
} from "@mariozechner/pi-ai";
import { AuthStorage, ModelRegistry } from "@mariozechner/pi-coding-agent";
import type {
  ExtractPayload,
  GenerateTemplatePayload,
  HealthResult,
  HealthRouteStatus,
  TemplateResult,
} from "./protocol.js";
import { parseRouteId } from "./config.js";
import {
  normalizeExtractResult,
  validateTemplateResult,
} from "./validation.js";

export async function runWithFallback<T>(
  routes: string[],
  invoke: (routeId: string) => Promise<T>,
): Promise<T> {
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
  throw new Error(errors.join(" | "));
}

export class SDKBackend {
  readonly authStorage: AuthStorage;
  readonly modelRegistry: ModelRegistry;

  constructor(
    private readonly routes: Record<string, string[]>,
    deps: { authStorage?: AuthStorage; modelRegistry?: ModelRegistry } = {},
  ) {
    this.authStorage =
      deps.modelRegistry?.authStorage ?? deps.authStorage ?? AuthStorage.create();
    this.modelRegistry = deps.modelRegistry ?? new ModelRegistry(this.authStorage);
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
    return runWithFallback(this.routes[capability] || [], async (routeId) => {
      const selection = await this.selectRoute(routeId, { requiresImage: false });
      const tool = this.extractTool();
      const response = await complete(
        selection.model,
        this.buildContext({
          userPrompt: buildExtractPrompt(payload),
          systemPrompt:
            "You extract structured data from HTML. Call the submit_extraction tool exactly once with concise, precise field values.",
          tools: [tool],
        }),
        {
          apiKey: selection.apiKey,
          maxTokens: 4096,
          temperature: 0,
        },
      );

      const call = getRequiredToolCall(response.content, tool.name);
      const args = validateToolCall([tool], call);
      return normalizeExtractResult(args, {
        provider: response.provider,
        model: response.model,
        tokens_used: response.usage.totalTokens,
      });
    });
  }

  async generateTemplate(
    capability: string,
    payload: GenerateTemplatePayload,
  ): Promise<TemplateResult> {
    return runWithFallback(this.routes[capability] || [], async (routeId) => {
      const selection = await this.selectRoute(routeId, { requiresImage: false });
      const tool = this.templateTool();
      const response = await complete(
        selection.model,
        this.buildContext({
          userPrompt: buildTemplatePrompt(payload),
          systemPrompt:
            "You generate extraction templates from HTML. Call the submit_template tool exactly once. Prefer robust CSS selectors and only use jsonld/regex when they add real value.",
          tools: [tool],
        }),
        {
          apiKey: selection.apiKey,
          maxTokens: 4096,
          temperature: 0,
        },
      );

      const call = getRequiredToolCall(response.content, tool.name);
      const args = validateToolCall([tool], call);
      return validateTemplateResult({
        ...(args as TemplateResult),
        provider: response.provider,
        model: response.model,
      });
    });
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
    tools: Tool[];
  }): Context {
    return {
      systemPrompt: input.systemPrompt,
      tools: input.tools,
      messages: [
        {
          role: "user",
          content: input.userPrompt,
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
}

export function modelSupportsImages(model: { input: string[] }): boolean {
  return model.input.includes("image");
}

const WORD_BOUNDARY_THRESHOLD = 0.8;

function getRequiredToolCall(
  content: Array<{ type: string; id?: string; name?: string; arguments?: Record<string, unknown> }>,
  toolName: string,
) : ToolCall {
  const toolCall = content.find(
    (block): block is ToolCall =>
      block.type === "toolCall" &&
      typeof block.id === "string" &&
      typeof block.name === "string" &&
      !!block.arguments &&
      block.name === toolName,
  );
  if (!toolCall) {
    throw new Error(`model did not call ${toolName}`);
  }
  return toolCall;
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
  parts.push(`HTML:\n${payload.html}`);
  return parts.join("\n\n");
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
