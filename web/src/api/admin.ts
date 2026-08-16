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
