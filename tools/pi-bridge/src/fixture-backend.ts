import type {
  ExportShapePayload,
  ExportShapeResult,
  ExtractPayload,
  ExtractResult,
  GeneratePipelineJsPayload,
  GenerateRenderProfilePayload,
  GenerateTemplatePayload,
  HealthResult,
  PipelineJsResult,
  RenderProfileResult,
  ResearchRefinePayload,
  ResearchRefineResult,
  TemplateResult,
} from "./protocol.js";
import {
  validateExportShapeResult,
  validateExtractResult,
  validatePipelineJsResult,
  validateRenderProfileResult,
  validateResearchRefineResult,
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

  refineResearch(
    capability: string,
    payload: ResearchRefinePayload,
  ): ResearchRefineResult {
    const evidence = payload.result.evidence ?? [];
    const summary = payload.result.summary?.trim() || "Fixture refined research summary.";
    const conciseSummary = payload.result.query?.trim()
      ? `Fixture refinement for ${payload.result.query.trim()}.`
      : "Fixture concise research summary.";

    return validateResearchRefineResult({
      refined: {
        summary,
        conciseSummary,
        keyFindings: [
          payload.instructions?.trim()
            ? `Operator focus: ${payload.instructions.trim()}`
            : "Collected evidence was normalized into a bounded brief.",
        ],
        openQuestions: payload.result.query?.trim()
          ? [`What additional evidence would further validate ${payload.result.query.trim()}?`]
          : ["What additional evidence should be gathered next?"],
        recommendedNextSteps: [
          "Review the highlighted evidence before sharing the refined brief.",
        ],
        evidenceHighlights: evidence.slice(0, 2).map((item, index) => ({
          url: item.url,
          title: item.title,
          finding:
            item.snippet?.trim() ||
            `Fixture evidence highlight ${index + 1} from ${item.url}.`,
          relevance: "Top evidence selected by fixture backend.",
          citationUrl: item.citationUrl,
        })),
        confidence:
          typeof payload.result.confidence === "number"
            ? Math.min(1, Math.max(0, payload.result.confidence))
            : 0.75,
      },
      explanation: "Deterministic fixture research refinement from pi bridge.",
      route_id: this.firstRoute(capability),
      provider: "fixture",
      model: "fixture-model",
    });
  }

  shapeExport(
    capability: string,
    payload: ExportShapePayload,
  ): ExportShapeResult {
    const fieldOptions = payload.fieldOptions ?? [];
    const topLevelFields = fieldOptions
      .filter((option) => option.category === "top_level")
      .slice(0, 4)
      .map((option) => option.key);
    const normalizedFields = fieldOptions
      .filter((option) => option.category === "normalized")
      .slice(0, 4)
      .map((option) => option.key);
    const evidenceFields = fieldOptions
      .filter((option) => option.category === "evidence")
      .slice(0, 5)
      .map((option) => option.key);

    return validateExportShapeResult({
      shape: {
        topLevelFields,
        normalizedFields,
        evidenceFields,
        summaryFields: topLevelFields.slice(0, 3),
        fieldLabels: Object.fromEntries(
          fieldOptions.slice(0, 4).map((option) => [option.key, option.label || option.key]),
        ),
        formatting: {
          emptyValue: "—",
          multiValueJoin: payload.format === "md" ? ", " : "; ",
          markdownTitle:
            payload.jobKind === "research"
              ? "Fixture Research Export"
              : "Fixture Export",
        },
      },
      explanation: "Deterministic fixture export shaping from pi bridge.",
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
