export type ConfigurationDocument = Record<string, unknown>;

export interface ValidationIssue {
  path: string;
  message: string;
}

export interface Revision {
  id: number;
  created_at: string;
  source: string;
  note: string;
  checksum: string;
  active: boolean;
}

export interface ConfigDraftResponse {
  config: ConfigurationDocument;
  database_path: string;
  revision?: Revision;
}

export interface ValidationResponse {
  valid: boolean;
  issues: ValidationIssue[];
}

export interface PublishResponse {
  revision: Revision;
  restart_required: boolean;
  restart_fields: string[];
  auth_changed: boolean;
}

export interface LiveLogEntry {
  id: number;
  time: string;
  level: string;
  message: string;
  caller?: string;
}

export interface CredentialCapacityStatus {
  provider_id: string;
  provider_name: string;
  credential_id: string;
  credential_name: string;
  model: string;
  tpm_limit: number;
  reserved_tokens: number;
  remaining_tokens: number;
  available: boolean;
  cooldown_until?: string;
  available_at?: string;
}

export interface StatisticsSummary {
  requests: number;
  successful: number;
  success_rate: number;
  input_tokens: number;
  output_tokens: number;
  cached_tokens: number;
  cache_write_tokens: number;
  reasoning_tokens: number;
  total_tokens: number;
  known_usage: number;
  usage_rate: number;
  average_latency_ms: number;
  p50_latency_ms: number;
  p95_latency_ms: number;
  average_ttft_ms: number;
  ttft_samples: number;
  p50_ttft_ms: number;
  p95_ttft_ms: number;
  output_tokens_per_second: number;
}

export interface StatisticsPoint {
  time: string;
  requests: number;
  input_tokens: number;
  output_tokens: number;
  total_tokens: number;
}

export interface StatisticsBreakdown {
  id: string;
  name: string;
  requests: number;
  success_rate: number;
  known_usage: number;
  usage_rate: number;
  input_tokens: number;
  output_tokens: number;
  total_tokens: number;
}

export interface StatisticsFailure {
  request_id: string;
  created_at: string;
  status_code: number;
  protocol: string;
  client_model: string;
  provider_name: string;
  latency_ms: number;
}

export interface StatisticsOverview {
  from: string;
  to: string;
  bucket: "hour" | "day";
  filters: StatisticsFilters;
  summary: StatisticsSummary;
  series: StatisticsPoint[];
  providers: StatisticsBreakdown[];
  models: StatisticsBreakdown[];
  access_keys: StatisticsBreakdown[];
  protocols: StatisticsBreakdown[];
  recent_failures: StatisticsFailure[];
  dropped_events: number;
  retention_days: number;
}

export interface StatisticsFilters {
  provider?: string;
  model?: string;
  protocol?: string;
  access_key?: string;
  status?: "success" | "failure" | "";
}

export interface StatisticsQuery extends StatisticsFilters {
  from: Date;
  to: Date;
  bucket: "hour" | "day";
}

export class AdminRequestError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly code = "",
  ) {
    super(message);
    this.name = "AdminRequestError";
  }
}

function adminHeaders(apiKey: string): HeadersInit {
  const result: Record<string, string> = { "Content-Type": "application/json" };
  if (apiKey.trim()) result.Authorization = `Bearer ${apiKey.trim()}`;
  return result;
}

async function adminRequest<T>(path: string, apiKey: string, init: RequestInit = {}): Promise<T> {
  let response: Response;
  try {
    response = await fetch(path, { ...init, headers: { ...adminHeaders(apiKey), ...init.headers } });
  } catch {
    throw new AdminRequestError("无法连接配置服务，请先启动 Go/Wails 后端。", 0);
  }
  if (!response.ok) {
    if ([502, 503, 504].includes(response.status)) {
      throw new AdminRequestError("无法连接配置服务，请先启动 Go/Wails 后端。", response.status);
    }
    const detail = await response.text();
    let message = detail || `管理请求失败（HTTP ${response.status}）`;
    let code = "";
    try {
      const payload = JSON.parse(detail) as { error?: string; code?: string };
      message = payload.error || message;
      code = payload.code || "";
    } catch {
      // Keep the plain-text response as the actionable error message.
    }
    throw new AdminRequestError(message, response.status, code);
  }
  const contentType = response.headers.get("content-type") ?? "";
  if (!contentType.includes("application/json")) {
    throw new AdminRequestError("配置服务返回了无效响应，请确认桌面后端已启动。", response.status);
  }
  return (await response.json()) as T;
}

export function getConfigDraft(apiKey: string): Promise<ConfigDraftResponse> {
  return adminRequest("/api/admin/config/draft", apiKey);
}

export async function getConfigRevisions(apiKey: string): Promise<Revision[]> {
  const response = await adminRequest<{ data: Revision[] }>("/api/admin/config/revisions", apiKey);
  return response.data ?? [];
}

export function validateConfig(apiKey: string, config: ConfigurationDocument): Promise<ValidationResponse> {
  return adminRequest("/api/admin/config/validate", apiKey, {
    method: "POST",
    body: JSON.stringify({ config }),
  });
}

export function publishConfig(
  apiKey: string,
  config: ConfigurationDocument,
  note: string,
): Promise<PublishResponse> {
  return adminRequest("/api/admin/config/revisions", apiKey, {
    method: "POST",
    body: JSON.stringify({ config, note }),
  });
}

export function activateRevision(apiKey: string, id: number): Promise<PublishResponse> {
  return adminRequest(`/api/admin/config/revisions/${id}/activate`, apiKey, { method: "POST" });
}

export async function getAdminLogs(apiKey: string, after = 0, limit = 200): Promise<LiveLogEntry[]> {
  const response = await adminRequest<{ data: LiveLogEntry[] }>(`/api/admin/logs?after=${after}&limit=${limit}`, apiKey);
  return response.data ?? [];
}

export async function getCredentialCapacity(apiKey: string): Promise<CredentialCapacityStatus[]> {
  const response = await adminRequest<{ data: CredentialCapacityStatus[] }>("/api/admin/capacity", apiKey);
  return response.data ?? [];
}

function statisticsParams(query: StatisticsQuery): URLSearchParams {
  const params = new URLSearchParams({ from: query.from.toISOString(), to: query.to.toISOString(), bucket: query.bucket });
  for (const [name, value] of Object.entries({ provider: query.provider, model: query.model, protocol: query.protocol, access_key: query.access_key, status: query.status })) {
    if (value) params.set(name, value);
  }
  return params;
}

export function getStatisticsOverview(apiKey: string, query: StatisticsQuery): Promise<StatisticsOverview> {
  const params = statisticsParams(query);
  return adminRequest(`/api/admin/statistics/overview?${params}`, apiKey);
}

export async function downloadStatisticsCSV(apiKey: string, query: StatisticsQuery): Promise<void> {
  const response = await fetch(`/api/admin/statistics/export?${statisticsParams(query)}`, { headers: adminHeaders(apiKey) });
  if (!response.ok) throw new AdminRequestError(await response.text() || `导出失败（HTTP ${response.status}）`, response.status);
  const blob = await response.blob();
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = `simple-one-api-statistics-${query.from.toISOString().slice(0, 10)}-${query.to.toISOString().slice(0, 10)}.csv`;
  anchor.click();
  URL.revokeObjectURL(url);
}
