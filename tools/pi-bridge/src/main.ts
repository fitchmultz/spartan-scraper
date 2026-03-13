import readline from "node:readline";
import { loadBridgeConfig } from "./config.js";
import { FixtureBackend } from "./fixture-backend.js";
import {
  CAPABILITY_EXTRACT_NATURAL,
  CAPABILITY_EXTRACT_SCHEMA,
  CAPABILITY_TEMPLATE_GENERATE,
  OP_EXTRACT_PREVIEW,
  OP_GENERATE_TEMPLATE,
  OP_HEALTH,
  type BridgeRequest,
  type BridgeResponse,
  type ExtractPayload,
  type GenerateTemplatePayload,
} from "./protocol.js";
import { SDKBackend } from "./sdk-backend.js";

const config = loadBridgeConfig();
interface Backend {
  health(agentDir: string): unknown;
  extract(capability: string, payload: ExtractPayload): Promise<unknown> | unknown;
  generateTemplate(
    capability: string,
    payload: GenerateTemplatePayload,
  ): Promise<unknown> | unknown;
}

const backend: Backend =
  config.mode === "fixture"
    ? new FixtureBackend(config.mode, config.routes)
    : new SDKBackend(config.routes);

const rl = readline.createInterface({
  input: process.stdin,
  crlfDelay: Number.POSITIVE_INFINITY,
});

rl.on("line", async (line) => {
  if (!line.trim()) {
    return;
  }

  let request: BridgeRequest;
  try {
    request = JSON.parse(line) as BridgeRequest;
  } catch (error) {
    writeResponse({
      id: "unknown",
      ok: false,
      error: {
        code: "bad_request",
        message: error instanceof Error ? error.message : String(error),
      },
    });
    return;
  }

  try {
    switch (request.op) {
      case OP_HEALTH:
        writeResponse({
          id: request.id,
          ok: true,
          result: backend.health(config.agentDir),
        });
        return;
      case OP_EXTRACT_PREVIEW: {
        const payload = request.payload as ExtractPayload;
        const capability =
          payload.mode === "schema_guided"
            ? CAPABILITY_EXTRACT_SCHEMA
            : CAPABILITY_EXTRACT_NATURAL;
        const result = await backend.extract(capability, payload);
        writeResponse({ id: request.id, ok: true, result });
        return;
      }
      case OP_GENERATE_TEMPLATE: {
        const payload = request.payload as GenerateTemplatePayload;
        const result = await backend.generateTemplate(
          CAPABILITY_TEMPLATE_GENERATE,
          payload,
        );
        writeResponse({ id: request.id, ok: true, result });
        return;
      }
      default:
        writeResponse({
          id: request.id,
          ok: false,
          error: {
            code: "bad_request",
            message: `unknown operation: ${request.op}`,
          },
        });
    }
  } catch (error) {
    writeResponse({
      id: request.id,
      ok: false,
      error: {
        code: "bridge_error",
        message: error instanceof Error ? error.message : String(error),
      },
    });
  }
});

function writeResponse(response: BridgeResponse) {
  process.stdout.write(`${JSON.stringify(response)}\n`);
}
