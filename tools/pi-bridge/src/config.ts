import { readFileSync } from "node:fs";
import { getAgentDir } from "@mariozechner/pi-coding-agent";
import {
  CAPABILITY_EXTRACT_NATURAL,
  CAPABILITY_EXTRACT_SCHEMA,
  CAPABILITY_PIPELINE_JS_GENERATE,
  CAPABILITY_RENDER_PROFILE_GENERATE,
  CAPABILITY_TEMPLATE_GENERATE,
} from "./protocol.js";

export interface BridgeConfigFile {
  mode?: string;
  routes?: Record<string, string[]>;
}

export interface BridgeConfig {
  mode: string;
  configPath?: string;
  agentDir: string;
  routes: Record<string, string[]>;
}

export const DEFAULT_ROUTE_ORDER = [
  "openai/gpt-5.4",
  "kimi-coding/k2p5",
  "zai/glm-5",
] as const;

export function defaultRoutes(): Record<string, string[]> {
  return {
    [CAPABILITY_EXTRACT_NATURAL]: [...DEFAULT_ROUTE_ORDER],
    [CAPABILITY_EXTRACT_SCHEMA]: [...DEFAULT_ROUTE_ORDER],
    [CAPABILITY_TEMPLATE_GENERATE]: [...DEFAULT_ROUTE_ORDER],
    [CAPABILITY_RENDER_PROFILE_GENERATE]: [...DEFAULT_ROUTE_ORDER],
    [CAPABILITY_PIPELINE_JS_GENERATE]: [...DEFAULT_ROUTE_ORDER],
  };
}

export function loadBridgeConfig(env: NodeJS.ProcessEnv = process.env): BridgeConfig {
  const mode = (env.PI_MODE || "sdk").trim() || "sdk";
  const configPath = env.PI_CONFIG_PATH?.trim() || undefined;
  const routes = defaultRoutes();

  if (configPath) {
    const parsed = JSON.parse(readFileSync(configPath, "utf8")) as BridgeConfigFile;
    if (parsed.routes) {
      for (const [capability, routeList] of Object.entries(parsed.routes)) {
        const normalized = normalizeRouteList(routeList);
        if (normalized.length > 0) {
          routes[capability] = normalized;
        }
      }
    }
    if (parsed.mode?.trim()) {
      return {
        mode: parsed.mode.trim(),
        configPath,
        agentDir: getAgentDir(),
        routes,
      };
    }
  }

  return {
    mode,
    configPath,
    agentDir: getAgentDir(),
    routes,
  };
}

export function normalizeRouteList(routeList: string[] | undefined): string[] {
  if (!routeList) {
    return [];
  }
  return routeList.map((route) => route.trim()).filter(Boolean);
}

export function parseRouteId(routeId: string): { provider: string; model: string } {
  const [provider, ...rest] = routeId.split("/");
  const model = rest.join("/").trim();
  if (!provider?.trim() || !model) {
    throw new Error(`Invalid route ID: ${routeId}`);
  }
  return { provider: provider.trim(), model };
}
