import type {
  ExtractPayload,
  ExtractResult,
  GeneratePipelineJsPayload,
  GenerateRenderProfilePayload,
  GenerateTemplatePayload,
  HealthResult,
  PipelineJsResult,
  RenderProfileResult,
  TemplateResult,
} from "./protocol.js";
import {
  validateExtractResult,
  validatePipelineJsResult,
  validateRenderProfileResult,
  validateTemplateResult,
} from "./validation.js";

export class FixtureBackend {
  constructor(
    private readonly mode: string,
    private readonly resolvedRoutes: Record<string, string[]>,
  ) {}

  health(agentDir: string): HealthResult {
    return {
      mode: this.mode,
      agent_dir: agentDir,
      resolved: this.resolvedRoutes,
      available: this.resolvedRoutes,
    };
  }

  extract(capability: string, payload: ExtractPayload): ExtractResult {
    const fieldNames =
      payload.fields && payload.fields.length > 0
        ? payload.fields
        : payload.schema_example
          ? Object.keys(payload.schema_example)
          : ["summary"];

    const fields = Object.fromEntries(
      fieldNames.map((name) => [
        name,
        {
          values: [`fixture:${name}`],
          source: "llm" as const,
        },
      ]),
    );

    return validateExtractResult({
      fields,
      confidence: 0.75,
      explanation: "Deterministic fixture response from pi bridge.",
      tokens_used: 0,
      route_id: this.firstRoute(capability),
      provider: "fixture",
      model: "fixture-model",
    });
  }

  generateTemplate(
    capability: string,
    payload: GenerateTemplatePayload,
  ): TemplateResult {
    const sampleFields =
      payload.sample_fields && payload.sample_fields.length > 0
        ? payload.sample_fields
        : ["title", "description"];

    return validateTemplateResult({
      template: {
        name: "fixture-template",
        selectors: sampleFields.map((field) => ({
          name: field,
          selector: `[data-field="${field}"]`,
          attr: "text",
          trim: true,
        })),
        normalize: {
          titleField: sampleFields[0],
        },
      },
      explanation: "Deterministic fixture template from pi bridge.",
      route_id: this.firstRoute(capability),
      provider: "fixture",
      model: "fixture-model",
    });
  }

  generateRenderProfile(
    capability: string,
    _payload: GenerateRenderProfilePayload,
  ): RenderProfileResult {
    return validateRenderProfileResult({
      profile: {
        preferHeadless: true,
        wait: {
          mode: "selector",
          selector: "main",
        },
      },
      explanation: "Deterministic fixture render profile from pi bridge.",
      route_id: this.firstRoute(capability),
      provider: "fixture",
      model: "fixture-model",
    });
  }

  generatePipelineJs(
    capability: string,
    _payload: GeneratePipelineJsPayload,
  ): PipelineJsResult {
    return validatePipelineJsResult({
      script: {
        selectors: ["main"],
        postNav: "window.scrollTo(0, 0);",
      },
      explanation: "Deterministic fixture pipeline JS from pi bridge.",
      route_id: this.firstRoute(capability),
      provider: "fixture",
      model: "fixture-model",
    });
  }

  private firstRoute(capability: string): string | undefined {
    const routes = this.resolvedRoutes[capability];
    return routes && routes.length > 0 ? routes[0] : undefined;
  }
}
