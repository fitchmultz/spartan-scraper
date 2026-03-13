import type {
  ExtractPayload,
  ExtractResult,
  GenerateTemplatePayload,
  HealthResult,
  TemplateResult,
} from "./protocol.js";
import { validateExtractResult, validateTemplateResult } from "./validation.js";

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

  extract(_capability: string, payload: ExtractPayload): ExtractResult {
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
      provider: "fixture",
      model: "fixture-model",
    });
  }

  generateTemplate(
    _capability: string,
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
      provider: "fixture",
      model: "fixture-model",
    });
  }
}
