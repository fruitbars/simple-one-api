import type { ConfigurationDocument } from "./api/admin";

export interface LimitConfiguration {
  qps?: number;
  qpm?: number;
  rpm?: number;
  concurrency?: number;
  timeout?: number;
}

export interface ServiceConfiguration extends ConfigurationDocument {
  id?: string;
  name?: string;
  provider?: string;
  models?: string[];
  embedding_models?: string[];
  enabled?: boolean;
  credentials?: Record<string, unknown>;
  credential_list?: Array<Record<string, unknown>>;
  server_url?: string;
  model_map?: Record<string, string>;
  model_redirect?: Record<string, string>;
  limit?: LimitConfiguration;
  embedding_limit?: LimitConfiguration;
  use_proxy?: boolean;
  timeout?: number;
}

export interface AccessKeyConfiguration extends ConfigurationDocument {
  api_key?: string;
  supported_models?: Record<string, string[]>;
}

export interface ProxyConfiguration extends ConfigurationDocument {
  strategy?: string;
  type?: string;
  http_proxy?: string;
  https_proxy?: string;
  socks5_proxy?: string;
  timeout?: number;
}

export interface AppConfiguration extends ConfigurationDocument {
  server_port?: string;
  debug?: boolean;
  log_level?: string;
  api_key?: string;
  load_balancing?: string;
  circuit_breaker?: {
    enabled?: boolean;
    failure_threshold?: number;
    recovery_timeout_seconds?: number;
    half_open_max_requests?: number;
  };
  statistics?: {
    enabled?: boolean;
    retention_days?: number;
  };
  enable_web?: boolean;
  services?: Record<string, ServiceConfiguration[]>;
  api_keys?: AccessKeyConfiguration[];
  proxy?: ProxyConfiguration;
}

export const serviceTypes = [
  "openai",
  "azure",
  "deepseek",
  "zhipu",
  "groq",
  "ollama",
  "gemini",
  "claude",
  "qianfan",
  "hunyuan",
  "xinghuo",
  "minimax",
  "huoshan",
  "dashscope",
  "bailian",
  "dify",
  "vertexai",
] as const;

export function asConfiguration(document: ConfigurationDocument): AppConfiguration {
  return document as AppConfiguration;
}

export function stringList(value: string): string[] {
  return Array.from(
    new Set(
      value
        .split(/[\n,]/)
        .map((item) => item.trim())
        .filter(Boolean),
    ),
  );
}

export function displayStringList(values: string[] | undefined): string {
  return (values ?? []).join(", ");
}

export function setModelAlias(
  models: string[] | undefined,
  modelMap: Record<string, string> | undefined,
  previousAlias: string,
  alias: string,
  upstreamModel: string,
): { models: string[]; modelMap: Record<string, string> } {
  const normalizedAlias = alias.trim();
  const normalizedTarget = upstreamModel.trim();
  const nextMap: Record<string, string> = {};
  let replaced = false;
  for (const [key, value] of Object.entries(modelMap ?? {})) {
    if (key === previousAlias) {
      nextMap[normalizedAlias] = normalizedTarget;
      replaced = true;
    } else {
      nextMap[key] = value;
    }
  }
  if (!replaced) nextMap[normalizedAlias] = normalizedTarget;

  const nextModels = (models ?? []).map((model) => model === previousAlias ? normalizedAlias : model);
  if (!nextModels.includes(normalizedAlias)) nextModels.push(normalizedAlias);
  return { models: Array.from(new Set(nextModels)), modelMap: nextMap };
}

export function removeModelAlias(
  models: string[] | undefined,
  modelMap: Record<string, string> | undefined,
  alias: string,
): { models: string[]; modelMap: Record<string, string> } {
  const nextMap = { ...(modelMap ?? {}) };
  delete nextMap[alias];
  return { models: (models ?? []).filter((model) => model !== alias), modelMap: nextMap };
}

export type ScalarCredential = string | number | boolean | null;

export function scalarCredentialEntries(credentials: Record<string, unknown> | undefined): Array<[string, ScalarCredential]> {
  return Object.entries(credentials ?? {})
    .filter(([, value]) => value == null || ["string", "number", "boolean"].includes(typeof value))
    .map(([key, value]) => [key, value as ScalarCredential]);
}

export function createService(serviceName: string): ServiceConfiguration {
  return {
    id: crypto.randomUUID(),
    provider: serviceName,
    enabled: true,
    models: [],
    credentials: {},
    limit: {},
    timeout: 120,
  };
}
