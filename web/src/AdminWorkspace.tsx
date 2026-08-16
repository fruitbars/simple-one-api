import {
  Boxes,
  Braces,
  CheckCircle2,
  ChevronDown,
  CircleCheck,
  Code2,
  Copy,
  Database,
  Download,
  KeyRound,
  LockKeyhole,
  Network,
  Pencil,
  Plus,
  RefreshCw,
  Save,
  Search,
  Settings2,
  ShieldCheck,
  Trash2,
  Upload,
  WandSparkles,
  MessageSquare,
  ScrollText,
  X,
} from "lucide-react";
import { lazy, Suspense, useEffect, useRef, useState } from "react";
import {
  AdminRequestError,
  getConfigDraft,
  getAdminLogs,
  publishConfig,
  validateConfig,
  type ConfigurationDocument,
  type ValidationIssue,
  type LiveLogEntry,
} from "./api/admin";
import {
  asConfiguration,
  createService,
  displayStringList,
  scalarCredentialEntries,
  serviceTypes,
  stringList,
  type AccessKeyConfiguration,
  type AppConfiguration,
  type ServiceConfiguration,
} from "./configuration";
import type { SourceFormat } from "./sourceDocument";

const ConfigurationSourceEditor = lazy(() => import("./ConfigurationSourceEditor"));

interface AdminWorkspaceProps {
  apiKey: string;
  onApiKeyChange: (value: string) => void;
  onBack: () => void;
}

type AdminSection = "system" | "providers" | "access" | "network" | "advanced" | "logs";

const sectionMeta: Array<{ id: AdminSection; label: string; description: string; icon: typeof Settings2 }> = [
  { id: "system", label: "基础设置", description: "服务、日志与负载均衡", icon: Settings2 },
  { id: "providers", label: "Provider", description: "模型、端点和凭证", icon: Boxes },
  { id: "access", label: "访问控制", description: "网关密钥和模型权限", icon: KeyRound },
  { id: "network", label: "网络代理", description: "HTTP、HTTPS 与 SOCKS5", icon: Network },
  { id: "advanced", label: "配置源码", description: "迁移与高级字段", icon: Braces },
  { id: "logs", label: "实时日志", description: "运行状态与错误诊断", icon: ScrollText },
];

export function AdminWorkspace({ apiKey, onApiKeyChange, onBack }: AdminWorkspaceProps) {
  const [configuration, setConfiguration] = useState<AppConfiguration>({});
  const [document, setDocument] = useState("");
  const [section, setSection] = useState<AdminSection>("system");
  const [databasePath, setDatabasePath] = useState("");
  const [issues, setIssues] = useState<ValidationIssue[]>([]);
  const [message, setMessage] = useState("");
  const [messagePersistent, setMessagePersistent] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [bootstrapRequired, setBootstrapRequired] = useState(false);
  const [bootstrapCredential, setBootstrapCredential] = useState("");
  const [sourceEditing, setSourceEditing] = useState(false);
  const [sourceChanged, setSourceChanged] = useState(false);
  const importInputRef = useRef<HTMLInputElement | null>(null);

  const services = configuration.services ?? {};
  const providerCount = Object.values(services).reduce((count, entries) => count + entries.length, 0);
  const enabledCount = Object.values(services).reduce(
    (count, entries) => count + entries.filter((entry) => entry.enabled).length,
    0,
  );
  const modelCount = Object.values(services).reduce(
    (count, entries) => count + entries.reduce((total, entry) => total + (entry.models?.length ?? 0), 0),
    0,
  );

  useEffect(() => {
    void refresh();
  }, [apiKey]);

  useEffect(() => {
    if (!message || messagePersistent) return;
    const timer = window.setTimeout(() => setMessage(""), 5000);
    return () => window.clearTimeout(timer);
  }, [message, messagePersistent]);

  function showMessage(value: string, persistent = false) {
    setMessagePersistent(persistent);
    setMessage(value);
  }

  async function refresh(credential = apiKey): Promise<boolean> {
    setLoading(true);
    setError("");
    try {
      const draft = await getConfigDraft(credential);
      const next = asConfiguration(draft.config);
      setConfiguration(next);
      setDocument(JSON.stringify(next, null, 2));
      setDatabasePath(draft.database_path);
      setIssues([]);
      setDirty(false);
      setSourceEditing(false);
      setSourceChanged(false);
      setBootstrapRequired(false);
      return true;
    } catch (reason) {
      if (reason instanceof AdminRequestError && reason.code === "admin_bootstrap_required") {
        setBootstrapRequired(true);
        setError("");
      } else {
        setError(reason instanceof Error ? reason.message : "无法读取配置仓库");
      }
      return false;
    } finally {
      setLoading(false);
    }
  }

  async function unlockBootstrap() {
    const credential = bootstrapCredential.trim();
    if (!credential) {
      setError("请输入服务启动日志中的初始化密钥。");
      return;
    }
    if (await refresh(credential)) {
      onApiKeyChange(credential);
      setBootstrapCredential("");
      setSection("system");
      showMessage("后台已临时解锁。请设置网关主密钥并发布配置，完成首次初始化。", true);
    }
  }

  function replaceConfiguration(update: (current: AppConfiguration) => AppConfiguration) {
    setConfiguration((current) => {
      const next = update(current);
      setDocument(JSON.stringify(next, null, 2));
      return next;
    });
    setDirty(true);
    setIssues([]);
    setMessage("");
  }

  function parseDocument(): AppConfiguration | null {
    try {
      const value = JSON.parse(document) as unknown;
      if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error("配置根节点必须是对象。");
      const parsed = asConfiguration(value as ConfigurationDocument);
      setError("");
      return parsed;
    } catch (reason) {
      setIssues([{ path: "JSON", message: reason instanceof Error ? reason.message : "格式错误" }]);
      return null;
    }
  }

  function currentDraft(): AppConfiguration | null {
    return section === "advanced" ? parseDocument() : configuration;
  }

  function selectSection(next: AdminSection) {
    if (section === "advanced" && next !== "advanced") {
      if (sourceChanged) {
        setIssues([{ path: "配置源码", message: "请先应用或取消源码更改。" }]);
        return;
      }
    }
    if (next === "advanced") setDocument(JSON.stringify(configuration, null, 2));
    setSection(next);
  }

  function startSourceEditing() {
    setDocument(JSON.stringify(configuration, null, 2));
    setSourceEditing(true);
    setSourceChanged(false);
    setIssues([]);
  }

  function cancelSourceEditing() {
    setDocument(JSON.stringify(configuration, null, 2));
    setSourceEditing(false);
    setSourceChanged(false);
    setIssues([]);
  }

  function applySourceChanges() {
    const parsed = parseDocument();
    if (!parsed) return;
    setConfiguration(parsed);
    setDocument(JSON.stringify(parsed, null, 2));
    setDirty(true);
    setSourceEditing(false);
    setSourceChanged(false);
    showMessage("源码更改已应用到当前草稿，保存配置后生效。");
  }

  function formatSource() {
    const parsed = parseDocument();
    if (parsed) setDocument(JSON.stringify(parsed, null, 2));
  }

  async function importSource(file: File) {
    try {
      const { parseSourceDocument, sourceFormatFromFilename } = await import("./sourceDocument");
      const content = await file.text();
      const parsed = parseSourceDocument(content, sourceFormatFromFilename(file.name));
      const next = asConfiguration(parsed as ConfigurationDocument);
      setDocument(JSON.stringify(next, null, 2));
      setSourceEditing(true);
      setSourceChanged(true);
      setIssues([]);
      showMessage(`${file.name} 已导入，请检查后应用更改。`);
    } catch (reason) {
      setIssues([{ path: file.name, message: reason instanceof Error ? reason.message : "无法解析文件" }]);
    } finally {
      if (importInputRef.current) importInputRef.current.value = "";
    }
  }

  async function exportSource(format: SourceFormat) {
    const parsed = parseDocument();
    if (!parsed) return;
    const { serializeSourceDocument } = await import("./sourceDocument");
    const content = serializeSourceDocument(parsed, format);
    const blob = new Blob([content], { type: format === "json" ? "application/json" : "application/yaml" });
    const url = URL.createObjectURL(blob);
    const anchor = window.document.createElement("a");
    anchor.href = url;
    anchor.download = `simple-one-api.${format === "json" ? "json" : "yaml"}`;
    anchor.click();
    URL.revokeObjectURL(url);
  }

  async function validate() {
    const draft = currentDraft();
    if (!draft) return;
    setSaving(true);
    setMessage("");
    setError("");
    try {
      const result = await validateConfig(apiKey, draft);
      setIssues(result.issues ?? []);
      if (result.valid) showMessage("配置校验通过，可以保存。");
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "校验失败");
    } finally {
      setSaving(false);
    }
  }

  async function publish() {
    const draft = currentDraft();
    if (!draft) return;
    setSaving(true);
    setMessage("");
    setError("");
    try {
      const validation = await validateConfig(apiKey, draft);
      setIssues(validation.issues ?? []);
      if (!validation.valid) return;
      const result = await publishConfig(apiKey, draft, "桌面端保存");
      const configuredKey = typeof draft.api_key === "string" ? draft.api_key : "";
      const nextKey = result.auth_changed && !configuredKey.startsWith("__SIMPLE_ONE_REDACTED__") ? configuredKey : apiKey;
      if (nextKey !== apiKey) onApiKeyChange(nextKey);
      showMessage(
        result.restart_required
          ? `配置已保存；${result.restart_fields.join("、")} 需要重启后生效。`
          : "配置已保存并立即生效。",
      );
      await refresh(nextKey);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "发布失败");
    } finally {
      setSaving(false);
    }
  }

  function updateService(serviceName: string, index: number, patch: Partial<ServiceConfiguration>) {
    replaceConfiguration((current) => {
      const nextServices = { ...(current.services ?? {}) };
      const entries = [...(nextServices[serviceName] ?? [])];
      entries[index] = { ...entries[index], ...patch };
      nextServices[serviceName] = entries;
      return { ...current, services: nextServices };
    });
  }

  function moveService(serviceName: string, index: number, target: string) {
    if (target === serviceName) return;
    replaceConfiguration((current) => {
      const nextServices = { ...(current.services ?? {}) };
      const source = [...(nextServices[serviceName] ?? [])];
      const [entry] = source.splice(index, 1);
      if (!entry) return current;
      if (source.length) nextServices[serviceName] = source;
      else delete nextServices[serviceName];
      nextServices[target] = [...(nextServices[target] ?? []), { ...entry, provider: target }];
      return { ...current, services: nextServices };
    });
  }

  function addService() {
    replaceConfiguration((current) => ({
      ...current,
      services: {
        ...(current.services ?? {}),
        openai: [...(current.services?.openai ?? []), createService("openai")],
      },
    }));
    setSection("providers");
  }

  function removeService(serviceName: string, index: number) {
    replaceConfiguration((current) => {
      const nextServices = { ...(current.services ?? {}) };
      const entries = [...(nextServices[serviceName] ?? [])];
      entries.splice(index, 1);
      if (entries.length) nextServices[serviceName] = entries;
      else delete nextServices[serviceName];
      return { ...current, services: nextServices };
    });
  }

  function updateCredential(serviceName: string, index: number, oldKey: string, key: string, value: unknown) {
    const service = services[serviceName]?.[index];
    if (!service) return;
    const credentials = { ...(service.credentials ?? {}) };
    if (oldKey && oldKey !== key) delete credentials[oldKey];
    if (key) credentials[key] = value;
    updateService(serviceName, index, { credentials });
  }

  function removeCredential(serviceName: string, index: number, key: string) {
    const service = services[serviceName]?.[index];
    if (!service) return;
    const credentials = { ...(service.credentials ?? {}) };
    delete credentials[key];
    updateService(serviceName, index, { credentials });
  }

  function addCredential(serviceName: string, index: number) {
    const service = services[serviceName]?.[index];
    if (!service) return;
    const credentials = { ...(service.credentials ?? {}) };
    let number = Object.keys(credentials).length + 1;
    let key = `credential_${number}`;
    while (key in credentials) key = `credential_${++number}`;
    credentials[key] = "";
    updateService(serviceName, index, { credentials });
  }

  const activeMeta = sectionMeta.find((item) => item.id === section) ?? sectionMeta[0];

  return (
    <div className="admin-shell">
      <aside className="admin-nav">
        <div className="admin-brand"><span className="brand-mark">S</span><span>Simple One <small>API</small></span></div>
        <div className="workspace-switch admin-workspace-switch" aria-label="工作区切换">
          <button className="active"><Settings2 size={16} />配置</button>
          <button onClick={onBack}><MessageSquare size={16} />Chat</button>
        </div>
        <div className="admin-nav-title">配置管理</div>
        {sectionMeta.map((item) => {
          const Icon = item.icon;
          return (
            <button key={item.id} className={`admin-nav-item ${section === item.id ? "active" : ""}`} onClick={() => selectSection(item.id)}>
              <Icon size={17} /><span><strong>{item.label}</strong><small>{item.description}</small></span>
            </button>
          );
        })}
        <div className="admin-nav-note">修改会先保存在当前草稿中，点击保存配置后生效。</div>
      </aside>

      <main className="admin-main">
        <header className="admin-header">
          <div><h1>{activeMeta.label}</h1><p>{activeMeta.description}</p></div>
          <div className="admin-header-actions">
            {dirty && <span className="dirty-badge">未发布更改</span>}
            <button className="secondary-button" onClick={() => void refresh()} disabled={loading || saving}><RefreshCw size={16} />刷新</button>
          </div>
        </header>

        {error && <div className="admin-alert error" role="alert">{error}</div>}
        {message && <div className="admin-alert success" role="status" aria-live="polite"><CheckCircle2 size={17} />{message}</div>}

        {bootstrapRequired && (
          <section className="admin-bootstrap-panel">
            <div className="bootstrap-icon"><LockKeyhole size={24} /></div>
            <div className="bootstrap-copy">
              <h2>首次初始化需要解锁</h2>
              <p>这是远程新部署的安全保护。请从服务启动日志复制临时初始化密钥，解锁后在“基础设置”中填写正式网关主密钥并发布。</p>
              <label className="visual-field">
                <span>临时初始化密钥</span>
                <input type="password" autoComplete="off" value={bootstrapCredential} onChange={(event) => setBootstrapCredential(event.target.value)} onKeyDown={(event) => { if (event.key === "Enter") void unlockBootstrap(); }} placeholder="粘贴启动日志中的 bootstrap token" autoFocus />
              </label>
              <button className="primary-button" onClick={() => void unlockBootstrap()} disabled={loading}><LockKeyhole size={16} />解锁并进入配置</button>
            </div>
          </section>
        )}

        {!bootstrapRequired && section !== "logs" && <section className="admin-metrics">
          <div className="metric-card"><Database size={19} /><div><span>SQLite 数据库</span><strong title={databasePath}>{databasePath || "未连接"}</strong></div></div>
          <div className="metric-card"><Boxes size={19} /><div><span>Provider</span><strong>{enabledCount}/{providerCount} 启用</strong></div></div>
          <div className="metric-card"><Settings2 size={19} /><div><span>可用模型</span><strong>{modelCount}</strong></div></div>
        </section>}

        {!bootstrapRequired && <div className="admin-grid visual-admin-grid">
          <section className={`admin-panel visual-config-panel ${section === "providers" ? "provider-config-panel" : ""} ${section === "advanced" ? "source-config-panel" : ""}`}>
            {section === "system" && (
              <SystemForm configuration={configuration} onChange={replaceConfiguration} />
            )}
            {section === "providers" && (
              <ProviderForm
                services={services}
                onAdd={addService}
                onMove={moveService}
                onRemove={removeService}
                onUpdate={updateService}
                onAddCredential={addCredential}
                onRemoveCredential={removeCredential}
                onUpdateCredential={updateCredential}
              />
            )}
            {section === "access" && (
              <AccessForm configuration={configuration} onChange={replaceConfiguration} />
            )}
            {section === "network" && (
              <NetworkForm configuration={configuration} onChange={replaceConfiguration} />
            )}
            {section === "advanced" && (
              <div className="advanced-editor-wrap">
                <div className="source-heading">
                  <div><h2>配置源码</h2><p>JSON 是内部标准格式；YAML 可用于导入和导出。</p></div>
                  <span className={`source-mode ${sourceEditing ? "editing" : ""}`}>{sourceEditing ? "正在编辑" : "只读"}</span>
                </div>
                <div className="source-toolbar">
                  <div className="source-toolbar-group">
                    {!sourceEditing ? <button className="secondary-button" onClick={startSourceEditing}><Pencil size={15} />开始编辑</button> : <>
                      <button className="secondary-button" onClick={formatSource}><WandSparkles size={15} />格式化</button>
                      <button className="secondary-button" onClick={cancelSourceEditing}><X size={15} />取消</button>
                      <button className="primary-button" onClick={applySourceChanges} disabled={!sourceChanged}><CheckCircle2 size={15} />应用更改</button>
                    </>}
                  </div>
                  <div className="source-toolbar-group">
                    <button className="icon-text-button" onClick={() => void navigator.clipboard.writeText(document)}><Copy size={15} />复制</button>
                    <button className="icon-text-button" onClick={() => importInputRef.current?.click()}><Upload size={15} />导入</button>
                    <button className="icon-text-button" onClick={() => void exportSource("json")}><Download size={15} />JSON</button>
                    <button className="icon-text-button" onClick={() => void exportSource("yaml")}><Download size={15} />YAML</button>
                    <input ref={importInputRef} className="source-file-input" type="file" accept=".json,.yaml,.yml,application/json,application/yaml,text/yaml" onChange={(event) => { const file = event.target.files?.[0]; if (file) void importSource(file); }} />
                  </div>
                </div>
                <div className={`source-editor ${sourceEditing ? "editing" : "readonly"}`}>
                  <Suspense fallback={<div className="source-editor-loading">正在加载源码编辑器…</div>}>
                    <ConfigurationSourceEditor value={document} editing={sourceEditing} loading={loading} onChange={(value) => { setDocument(value); setSourceChanged(true); setIssues([]); setMessage(""); }} />
                  </Suspense>
                </div>
              </div>
            )}
            {section === "logs" && <LiveLogsPanel apiKey={apiKey} />}

            {issues.length > 0 && (
              <div className="validation-list" role="alert">
                {issues.map((issue, index) => <div key={`${issue.path}-${index}`}><code>{issue.path}</code><span>{issue.message}</span></div>)}
              </div>
            )}
            {section !== "logs" && <div className="publish-row visual-publish-row">
              <div className={`publish-state ${sourceEditing ? "attention" : dirty ? "pending" : "synced"}`}>
                {sourceEditing ? <Braces size={15} /> : dirty ? <RefreshCw size={15} /> : <CheckCircle2 size={15} />}
                <span>{sourceChanged ? "源码有未应用的更改" : sourceEditing ? "正在编辑源码，应用后才能校验和保存" : dirty ? "配置有未保存的更改" : "配置已与运行状态同步"}</span>
              </div>
              <div className="publish-actions">
                <button className="secondary-button" onClick={() => void validate()} disabled={saving || loading || sourceEditing}><ShieldCheck size={16} />校验配置</button>
                <button className="primary-button" onClick={() => void publish()} disabled={saving || loading || !dirty || sourceEditing}><Save size={16} />保存配置</button>
              </div>
            </div>}
          </section>
        </div>}
      </main>
    </div>
  );
}

function LiveLogsPanel({ apiKey }: { apiKey: string }) {
  const [enabled, setEnabled] = useState(true);
  const [entries, setEntries] = useState<LiveLogEntry[]>([]);
  const [level, setLevel] = useState("all");
  const [loadError, setLoadError] = useState("");
  const cursorRef = useRef(0);
  const viewportRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!enabled) return;
    let disposed = false;
    const load = async () => {
      try {
        const next = await getAdminLogs(apiKey, cursorRef.current);
        if (disposed || next.length === 0) return;
        cursorRef.current = next[next.length - 1].id;
        setEntries((current) => [...current, ...next].slice(-500));
        setLoadError("");
      } catch (reason) {
        if (!disposed) setLoadError(reason instanceof Error ? reason.message : "无法读取实时日志");
      }
    };
    void load();
    const timer = window.setInterval(() => void load(), 1000);
    return () => {
      disposed = true;
      window.clearInterval(timer);
    };
  }, [apiKey, enabled]);

  useEffect(() => {
    if (!enabled || !viewportRef.current) return;
    viewportRef.current.scrollTop = viewportRef.current.scrollHeight;
  }, [enabled, entries]);

  const visibleEntries = level === "all" ? entries : entries.filter((entry) => entry.level === level);

  return (
    <div className="live-logs-panel">
      <div className="section-heading with-action">
        <div><h2>实时日志</h2><p>仅展示脱敏后的运行摘要，最多保留最近 500 条。</p></div>
        <Toggle compact label={enabled ? "已开启" : "已关闭"} checked={enabled} onChange={setEnabled} />
      </div>
      <div className="log-toolbar">
        <div className="status-filter" aria-label="日志级别筛选">
          {["all", "debug", "info", "warn", "error"].map((item) => (
            <button key={item} className={level === item ? "active" : ""} onClick={() => setLevel(item)}>{item === "all" ? "全部" : item.toUpperCase()}</button>
          ))}
        </div>
        <button className="icon-text-button" onClick={() => setEntries([])}><Trash2 size={15} />清空视图</button>
      </div>
      {loadError && <div className="log-inline-error" role="alert">{loadError}</div>}
      <div className="log-viewport" ref={viewportRef} aria-live="off">
        {!enabled && <div className="log-empty"><ScrollText size={24} /><span>实时日志已关闭</span></div>}
        {enabled && visibleEntries.length === 0 && <div className="log-empty"><ScrollText size={24} /><span>等待新的日志...</span></div>}
        {enabled && visibleEntries.map((entry) => (
          <div className="log-row" key={entry.id}>
            <time>{new Date(entry.time).toLocaleTimeString("zh-CN", { hour12: false })}</time>
            <span className={`log-level ${entry.level}`}>{entry.level.toUpperCase()}</span>
            <code>{entry.caller || "runtime"}</code>
            <span>{entry.message}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function SystemForm({ configuration, onChange }: { configuration: AppConfiguration; onChange: (update: (current: AppConfiguration) => AppConfiguration) => void }) {
  const update = (patch: Partial<AppConfiguration>) => onChange((current) => ({ ...current, ...patch }));
  const protocols = [
    { name: "OpenAI Chat Completions", client: "通用 OpenAI 客户端", path: "/v1/chat/completions" },
    { name: "OpenAI Responses", client: "Codex", path: "/v1/responses" },
    { name: "Anthropic Messages", client: "Claude Code", path: "/v1/messages" },
  ];
  return (
    <div className="visual-form">
      <div className="section-heading"><div><h2>服务基础设置</h2><p>这些设置决定网关监听方式、日志输出和模型选择策略。</p></div></div>
      <div className="form-grid two-columns">
        <Field label="监听地址" hint="例如 :9090 或 127.0.0.1:9090">
          <input value={configuration.server_port ?? ""} onChange={(event) => update({ server_port: event.target.value })} placeholder=":9090" />
        </Field>
        <Field label="负载均衡" hint="多个可用 Provider 之间的选择策略">
          <select value={configuration.load_balancing ?? "random"} onChange={(event) => update({ load_balancing: event.target.value })}>
            <option value="random">随机</option><option value="first">优先第一个</option><option value="round_robin">轮询</option><option value="hash">哈希</option>
          </select>
        </Field>
        <Field label="日志级别" hint="生产环境建议使用 info 或 warn">
          <select value={configuration.log_level ?? "info"} onChange={(event) => update({ log_level: event.target.value })}>
            <option value="debug">Debug</option><option value="info">Info</option><option value="warn">Warn</option><option value="error">Error</option><option value="prodj">Production JSON</option>
          </select>
        </Field>
        <Field label="网关主密钥" hint="同时用于当前单用户后台鉴权">
          <input type="password" autoComplete="off" value={configuration.api_key ?? ""} onChange={(event) => update({ api_key: event.target.value })} placeholder="留空时仅允许本机初始化" />
        </Field>
      </div>
      <section className="protocol-section" aria-labelledby="protocol-heading">
        <div className="subsection-heading">
          <div><strong id="protocol-heading">客户端接口</strong><span>三种协议共用 Provider、模型路由、鉴权、限流与代理配置</span></div>
          <span className="protocol-limit">请求上限 8 MiB</span>
        </div>
        <div className="protocol-list">
          {protocols.map((protocol) => (
            <div className="protocol-row" key={protocol.path}>
              <Code2 size={17} />
              <div className="protocol-copy"><strong>{protocol.name}</strong><span>{protocol.client}</span></div>
              <code>{protocol.path}</code>
              <span className="protocol-stream"><CircleCheck size={13} />实时 SSE</span>
              <button className="icon-button protocol-copy-button" type="button" title={`复制 ${protocol.path}`} aria-label={`复制 ${protocol.path}`} onClick={() => void navigator.clipboard.writeText(protocol.path)}><Copy size={15} /></button>
            </div>
          ))}
        </div>
      </section>
      <div className="toggle-list">
        <Toggle label="启用 Web 与后台" description="关闭后需要重启，浏览器界面将不再提供。" checked={configuration.enable_web ?? false} onChange={(value) => update({ enable_web: value })} />
        <Toggle label="调试模式" description="输出更多诊断信息；生产环境不建议开启。" checked={configuration.debug ?? false} onChange={(value) => update({ debug: value })} />
      </div>
    </div>
  );
}

interface ProviderFormProps {
  services: Record<string, ServiceConfiguration[]>;
  onAdd: () => void;
  onMove: (serviceName: string, index: number, target: string) => void;
  onRemove: (serviceName: string, index: number) => void;
  onUpdate: (serviceName: string, index: number, patch: Partial<ServiceConfiguration>) => void;
  onAddCredential: (serviceName: string, index: number) => void;
  onRemoveCredential: (serviceName: string, index: number, key: string) => void;
  onUpdateCredential: (serviceName: string, index: number, oldKey: string, key: string, value: unknown) => void;
}

function ProviderForm(props: ProviderFormProps) {
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState<"all" | "enabled" | "disabled">("all");
  const [collapsed, setCollapsed] = useState<Set<string>>(() => new Set());
  const entries = Object.entries(props.services).flatMap(([serviceName, items]) => items.map((service, index) => ({ serviceName, service, index })));
  const normalizedQuery = query.trim().toLowerCase();
  const filteredEntries = entries.filter(({ serviceName, service }) => {
    if (status === "enabled" && !service.enabled) return false;
    if (status === "disabled" && service.enabled) return false;
    if (!normalizedQuery) return true;
    return [serviceName, service.name, service.id, service.server_url, ...(service.models ?? [])]
      .some((value) => String(value ?? "").toLowerCase().includes(normalizedQuery));
  });
  const toggleCollapsed = (key: string) => setCollapsed((current) => {
    const next = new Set(current);
    if (next.has(key)) next.delete(key);
    else next.add(key);
    return next;
  });
  return (
    <div className="visual-form">
      <div className="section-heading with-action"><div><h2>Provider 与模型</h2><p>一个 Provider 条目对应一个稳定的路由、端点和凭证集合。</p></div><button className="secondary-button" onClick={props.onAdd}><Plus size={16} />添加 Provider</button></div>
      {entries.length > 0 && <div className="provider-toolbar">
        <label className="provider-search"><Search size={16} /><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="搜索 Provider、模型或地址" aria-label="搜索 Provider" /></label>
        <div className="status-filter" aria-label="Provider 状态筛选">
          {([['all', '全部'], ['enabled', '已启用'], ['disabled', '已停用']] as const).map(([value, label]) => <button key={value} className={status === value ? "active" : ""} onClick={() => setStatus(value)}>{label}</button>)}
        </div>
      </div>}
      {entries.length === 0 && <div className="visual-empty"><Boxes size={28} /><strong>还没有 Provider</strong><span>添加第一个 Provider，填写模型和凭证后即可开始聊天。</span><button className="primary-button" onClick={props.onAdd}><Plus size={16} />添加 OpenAI 兼容 Provider</button></div>}
      {entries.length > 0 && filteredEntries.length === 0 && <div className="visual-empty compact"><Search size={25} /><strong>没有匹配的 Provider</strong><span>换个关键词或状态筛选试试。</span></div>}
      <div className="provider-list">
        {filteredEntries.map(({ serviceName, service, index }) => {
          const cardKey = service.id ?? `${serviceName}-${index}`;
          const isCollapsed = collapsed.has(cardKey);
          return <article className={`provider-card ${isCollapsed ? "collapsed" : ""}`} key={cardKey}>
            <header className="provider-card-header">
              <button className="provider-title" onClick={() => toggleCollapsed(cardKey)} aria-expanded={!isCollapsed}>
                <span className={`provider-status ${service.enabled ? "enabled" : ""}`} />
                <div><strong>{service.name?.trim() || serviceName}</strong><code>{serviceName} · {service.models?.length ?? 0} 个模型</code></div>
              </button>
              <div className="provider-actions"><Toggle compact label="启用" checked={service.enabled ?? false} onChange={(enabled) => props.onUpdate(serviceName, index, { enabled })} /><button className="provider-collapse-button" onClick={() => toggleCollapsed(cardKey)} aria-label={isCollapsed ? `展开 ${service.name || serviceName}` : `折叠 ${service.name || serviceName}`}><ChevronDown size={17} /></button><button className="icon-danger-button" onClick={() => props.onRemove(serviceName, index)} aria-label={`删除 ${service.name || serviceName}`}><Trash2 size={16} /></button></div>
            </header>
            <div className="provider-card-body">
              <div className="form-grid two-columns provider-fields">
              <Field label="Provider 名称" hint="仅用于界面识别，例如“生产环境 OpenAI”">
                <input value={service.name ?? ""} onChange={(event) => props.onUpdate(serviceName, index, { name: event.target.value })} placeholder={`${serviceName} Provider`} />
              </Field>
              <Field label="服务类型" hint="决定请求使用哪个适配器">
                <div className="select-wrap"><select value={serviceName} onChange={(event) => props.onMove(serviceName, index, event.target.value)}>{serviceTypes.map((item) => <option key={item} value={item}>{item}</option>)}</select><ChevronDown size={15} /></div>
              </Field>
              <Field label="服务地址" hint="OpenAI 兼容服务可填写完整 API 地址">
                <input value={service.server_url ?? ""} onChange={(event) => props.onUpdate(serviceName, index, { server_url: event.target.value })} placeholder="https://api.example.com/v1" />
              </Field>
              <div className="provider-key-field">
                <Field label="上游 API Key" hint="用于访问模型服务商；保存后会脱敏显示">
                  <input type="password" autoComplete="off" value={String(service.credentials?.api_key ?? "")} onChange={(event) => props.onUpdateCredential(serviceName, index, "api_key", "api_key", event.target.value)} placeholder="粘贴服务商提供的 API Key" />
                </Field>
              </div>
              <Field label="聊天模型" hint="使用逗号或换行分隔">
                <StringListEditor multiline values={service.models} onChange={(models) => props.onUpdate(serviceName, index, { models })} placeholder="gpt-4o-mini, deepseek-chat" />
              </Field>
              <Field label="Embedding 模型" hint="可选，使用逗号或换行分隔">
                <StringListEditor multiline values={service.embedding_models} onChange={(embedding_models) => props.onUpdate(serviceName, index, { embedding_models })} placeholder="text-embedding-3-small" />
              </Field>
            </div>
            <div className="credential-section">
              <div className="subsection-heading"><div><strong>更多凭证</strong><span>Access Key、Secret Key 等服务商专用字段。</span></div><button onClick={() => props.onAddCredential(serviceName, index)}><Plus size={14} />添加字段</button></div>
              <div className="credential-list">
                {scalarCredentialEntries(service.credentials).filter(([key]) => key !== "api_key").map(([key, value]) => (
                  <div className="credential-row" key={key}>
                    <input aria-label="凭证字段名" value={key} onChange={(event) => props.onUpdateCredential(serviceName, index, key, event.target.value, value)} placeholder="api_key" />
                    {typeof value === "boolean" ? (
                      <select aria-label={`${key} 值`} value={String(value)} onChange={(event) => props.onUpdateCredential(serviceName, index, key, key, event.target.value === "true")}>
                        <option value="true">true</option><option value="false">false</option>
                      </select>
                    ) : (
                      <input aria-label={`${key} 值`} type={typeof value === "number" ? "number" : isSensitiveKey(key) ? "password" : "text"} autoComplete="off" value={value ?? ""} onChange={(event) => props.onUpdateCredential(serviceName, index, key, key, typeof value === "number" ? Number(event.target.value) : event.target.value)} placeholder="凭证值" />
                    )}
                    <button onClick={() => props.onRemoveCredential(serviceName, index, key)} aria-label={`删除 ${key}`}><Trash2 size={14} /></button>
                  </div>
                ))}
                {scalarCredentialEntries(service.credentials).filter(([key]) => key !== "api_key").length === 0 && <div className="credential-empty">大多数服务只需填写上方 API Key，无需添加其他凭证。</div>}
              </div>
            </div>
            <details className="provider-advanced">
              <summary>高级路由与限流</summary>
              <div className="form-grid four-columns">
                <Field label="稳定 ID"><input value={service.id ?? ""} onChange={(event) => props.onUpdate(serviceName, index, { id: event.target.value })} /></Field>
                <Field label="请求超时（秒）"><NumberInput value={service.timeout} onChange={(timeout) => props.onUpdate(serviceName, index, { timeout })} /></Field>
                <Field label="QPS"><NumberInput value={service.limit?.qps} onChange={(qps) => props.onUpdate(serviceName, index, { limit: { ...(service.limit ?? {}), qps } })} /></Field>
                <Field label="并发数"><NumberInput value={service.limit?.concurrency} onChange={(concurrency) => props.onUpdate(serviceName, index, { limit: { ...(service.limit ?? {}), concurrency } })} /></Field>
              </div>
              <Toggle label="使用代理" description="是否允许这个 Provider 按全局代理策略转发。" checked={service.use_proxy ?? false} onChange={(use_proxy) => props.onUpdate(serviceName, index, { use_proxy })} />
              {(service.credential_list?.length ?? 0) > 0 && <p className="advanced-note">该条目还有 {service.credential_list?.length} 组轮换凭证，可在配置源码中管理。</p>}
            </details>
            </div>
          </article>;
        })}
      </div>
    </div>
  );
}

function AccessForm({ configuration, onChange }: { configuration: AppConfiguration; onChange: (update: (current: AppConfiguration) => AppConfiguration) => void }) {
  const keys = configuration.api_keys ?? [];
  const updateKeys = (next: AccessKeyConfiguration[]) => onChange((current) => ({ ...current, api_keys: next }));
  return (
    <div className="visual-form">
      <div className="section-heading with-action"><div><h2>客户端访问密钥</h2><p>为不同客户端分配独立密钥，并限制可调用的模型。</p></div><button className="secondary-button" onClick={() => updateKeys([...keys, { api_key: "", supported_models: { all: ["*"] } }])}><Plus size={16} />添加密钥</button></div>
      {keys.length === 0 && <div className="visual-empty compact"><KeyRound size={26} /><strong>未配置细粒度访问密钥</strong><span>当前只使用基础设置中的网关主密钥。</span></div>}
      <div className="access-key-list">
        {keys.map((item, index) => (
          <article className="access-key-card" key={index}>
            <div className="access-key-header"><strong>客户端密钥 #{index + 1}</strong><button className="icon-danger-button" onClick={() => updateKeys(keys.filter((_, keyIndex) => keyIndex !== index))}><Trash2 size={16} /></button></div>
            <Field label="Access Key" hint="建议每个客户端使用不同的随机值"><input type="password" autoComplete="off" value={item.api_key ?? ""} onChange={(event) => { const next = [...keys]; next[index] = { ...item, api_key: event.target.value }; updateKeys(next); }} /></Field>
            <div className="scope-list">
              {Object.entries(item.supported_models ?? {}).map(([scope, models]) => (
                <div className="scope-row" key={scope}>
                  <input value={scope} aria-label="权限分组" onChange={(event) => { const supported = { ...(item.supported_models ?? {}) }; delete supported[scope]; if (event.target.value) supported[event.target.value] = models; const next = [...keys]; next[index] = { ...item, supported_models: supported }; updateKeys(next); }} placeholder="分组名称" />
                  <StringListEditor values={models} ariaLabel="允许模型" onChange={(allowedModels) => { const next = [...keys]; next[index] = { ...item, supported_models: { ...(item.supported_models ?? {}), [scope]: allowedModels } }; updateKeys(next); }} placeholder="* 或模型列表" />
                  <button onClick={() => { const supported = { ...(item.supported_models ?? {}) }; delete supported[scope]; const next = [...keys]; next[index] = { ...item, supported_models: supported }; updateKeys(next); }}><Trash2 size={14} /></button>
                </div>
              ))}
              <button className="inline-add-button" onClick={() => { const supported = { ...(item.supported_models ?? {}) }; let name = "models"; let count = 2; while (name in supported) name = `models-${count++}`; supported[name] = ["*"]; const next = [...keys]; next[index] = { ...item, supported_models: supported }; updateKeys(next); }}><Plus size={14} />添加模型权限组</button>
            </div>
          </article>
        ))}
      </div>
    </div>
  );
}

function NetworkForm({ configuration, onChange }: { configuration: AppConfiguration; onChange: (update: (current: AppConfiguration) => AppConfiguration) => void }) {
  const proxy = configuration.proxy ?? {};
  const update = (patch: Record<string, unknown>) => onChange((current) => ({ ...current, proxy: { ...(current.proxy ?? {}), ...patch } }));
  const enabled = (proxy.strategy ?? "disabled") !== "disabled";
  return (
    <div className="visual-form">
      <div className="section-heading"><div><h2>全局网络代理</h2><p>统一控制 Provider 是否通过 HTTP、HTTPS 或 SOCKS5 代理访问上游。</p></div></div>
      <div className="form-grid two-columns">
        <Field label="代理策略" hint="默认策略还会参考每个 Provider 的开关">
          <select value={proxy.strategy ?? "disabled"} onChange={(event) => update({ strategy: event.target.value })}>
            <option value="disabled">全部禁用</option><option value="default">按 Provider 设置</option><option value="all">默认启用，可单独关闭</option><option value="force_all">强制全部启用</option>
          </select>
        </Field>
        <Field label="代理类型"><select disabled={!enabled} value={proxy.type ?? ""} onChange={(event) => update({ type: event.target.value })}><option value="">请选择</option><option value="http">HTTP / HTTPS</option><option value="socks5">SOCKS5</option></select></Field>
        <Field label="HTTP 代理"><input disabled={!enabled || proxy.type === "socks5"} value={proxy.http_proxy ?? ""} onChange={(event) => update({ http_proxy: event.target.value })} placeholder="http://127.0.0.1:7890" /></Field>
        <Field label="HTTPS 代理"><input disabled={!enabled || proxy.type === "socks5"} value={proxy.https_proxy ?? ""} onChange={(event) => update({ https_proxy: event.target.value })} placeholder="http://127.0.0.1:7890" /></Field>
        <Field label="SOCKS5 代理"><input disabled={!enabled || proxy.type !== "socks5"} value={proxy.socks5_proxy ?? ""} onChange={(event) => update({ socks5_proxy: event.target.value })} placeholder="socks5://127.0.0.1:1080" /></Field>
        <Field label="连接超时（秒）"><NumberInput disabled={!enabled} value={proxy.timeout} onChange={(timeout) => update({ timeout })} /></Field>
      </div>
      <div className="network-note"><ShieldCheck size={17} /><div><strong>代理凭证会自动脱敏</strong><span>URL 中的用户名和密码不会原样返回浏览器；保留占位符即可继续使用原凭证。</span></div></div>
    </div>
  );
}

function Field({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }) {
  return <label className="visual-field"><span>{label}</span>{children}{hint && <small>{hint}</small>}</label>;
}

function Toggle({ label, description, checked, onChange, compact = false }: { label: string; description?: string; checked: boolean; onChange: (value: boolean) => void; compact?: boolean }) {
  return (
    <label className={`toggle-control ${compact ? "compact" : ""}`}>
      <input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} />
      <span className="toggle-track"><i /></span>
      <span className="toggle-copy"><strong>{label}</strong>{description && <small>{description}</small>}</span>
    </label>
  );
}

function NumberInput({ value, onChange, disabled = false }: { value?: number; onChange: (value: number) => void; disabled?: boolean }) {
  return <input type="number" min="0" disabled={disabled} value={value ?? 0} onChange={(event) => onChange(Number(event.target.value))} />;
}

function StringListEditor({ values, onChange, placeholder, multiline = false, ariaLabel }: { values?: string[]; onChange: (values: string[]) => void; placeholder: string; multiline?: boolean; ariaLabel?: string }) {
  const normalized = displayStringList(values);
  const [draft, setDraft] = useState(normalized);
  const [editing, setEditing] = useState(false);

  useEffect(() => {
    if (!editing) setDraft(normalized);
  }, [editing, normalized]);

  const update = (value: string) => {
    setDraft(value);
    onChange(stringList(value));
  };
  const finish = () => {
    setEditing(false);
    setDraft(displayStringList(stringList(draft)));
  };

  if (multiline) {
    return <textarea rows={2} value={draft} onFocus={() => setEditing(true)} onBlur={finish} onChange={(event) => update(event.target.value)} placeholder={placeholder} aria-label={ariaLabel} />;
  }
  return <input value={draft} onFocus={() => setEditing(true)} onBlur={finish} onChange={(event) => update(event.target.value)} placeholder={placeholder} aria-label={ariaLabel} />;
}

function isSensitiveKey(key: string): boolean {
  const normalized = key.toLowerCase();
  return ["key", "secret", "token", "password", "authorization"].some((fragment) => normalized.includes(fragment));
}
