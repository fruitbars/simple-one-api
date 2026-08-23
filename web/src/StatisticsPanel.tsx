import { Activity, AlertCircle, ArrowDownToLine, ArrowUpFromLine, BarChart3, CalendarDays, Download, Gauge, RefreshCw } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  downloadStatisticsCSV,
  getStatisticsOverview,
  type StatisticsBreakdown,
  type StatisticsFilters,
  type StatisticsOverview,
  type StatisticsPoint,
  type StatisticsQuery,
  type StatisticsSummary,
} from "./api/admin";

type Range = "today" | "7d" | "30d" | "custom";

const ranges: Array<{ id: Exclude<Range, "custom">; label: string; hours: number }> = [
  { id: "today", label: "今日", hours: 0 },
  { id: "7d", label: "7 天", hours: 7 * 24 },
  { id: "30d", label: "30 天", hours: 30 * 24 },
];

const emptyFilters: StatisticsFilters = { provider: "", model: "", protocol: "", access_key: "", status: "" };

export function rangeQuery(range: Exclude<Range, "custom">, now = new Date()): { from: Date; to: Date; bucket: "hour" | "day" } {
  const selected = ranges.find((item) => item.id === range) ?? ranges[0];
  const from = range === "today"
    ? new Date(now.getFullYear(), now.getMonth(), now.getDate())
    : new Date(now.getTime() - selected.hours * 60 * 60 * 1000);
  return { from, to: now, bucket: range === "today" ? "hour" : "day" };
}

export function compactNumber(value: number): string {
  return new Intl.NumberFormat("zh-CN", { notation: value >= 10_000 ? "compact" : "standard", maximumFractionDigits: 1 }).format(value);
}

export function formatDuration(milliseconds: number): string {
  if (!Number.isFinite(milliseconds)) return "-";
  const value = Math.max(0, milliseconds);
  if (value < 1000) return `${Math.round(value)} ms`;
  if (value < 10_000) {
    const seconds = Math.round(value / 100) / 10;
    return `${seconds} 秒`;
  }

  const totalSeconds = Math.round(value / 1000);
  if (totalSeconds < 60) return `${totalSeconds} 秒`;
  const totalMinutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (totalMinutes < 60) return `${totalMinutes} 分${seconds ? ` ${seconds} 秒` : ""}`;
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  return `${hours} 小时${minutes ? ` ${minutes} 分` : ""}`;
}

function localDateTime(value: Date): string {
  const offset = value.getTimezoneOffset() * 60_000;
  return new Date(value.getTime() - offset).toISOString().slice(0, 16);
}

export default function StatisticsPanel({ apiKey }: { apiKey: string }) {
  const now = useMemo(() => new Date(), []);
  const [range, setRange] = useState<Range>("today");
  const [customFrom, setCustomFrom] = useState(localDateTime(new Date(now.getTime() - 24 * 60 * 60 * 1000)));
  const [customTo, setCustomTo] = useState(localDateTime(now));
  const [filters, setFilters] = useState<StatisticsFilters>(emptyFilters);
  const [overview, setOverview] = useState<StatisticsOverview | null>(null);
  const [previous, setPrevious] = useState<StatisticsSummary | null>(null);
  const [facets, setFacets] = useState<{ providers: StatisticsBreakdown[]; models: StatisticsBreakdown[]; protocols: StatisticsBreakdown[]; accessKeys: StatisticsBreakdown[] }>({ providers: [], models: [], protocols: [], accessKeys: [] });
  const [loading, setLoading] = useState(true);
  const [exporting, setExporting] = useState(false);
  const [error, setError] = useState("");

  const currentQuery = useCallback((): StatisticsQuery => {
    const selected = range === "custom"
      ? { from: new Date(customFrom), to: new Date(customTo), bucket: (new Date(customTo).getTime() - new Date(customFrom).getTime() <= 48 * 60 * 60 * 1000 ? "hour" : "day") as "hour" | "day" }
      : rangeQuery(range);
    if (!Number.isFinite(selected.from.getTime()) || !Number.isFinite(selected.to.getTime()) || selected.from >= selected.to) {
      throw new Error("自定义时间范围无效");
    }
    return { ...selected, ...filters };
  }, [customFrom, customTo, filters, range]);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const query = currentQuery();
      const duration = query.to.getTime() - query.from.getTime();
      const previousQuery = { ...query, from: new Date(query.from.getTime() - duration), to: query.from };
      const [current, prior] = await Promise.all([getStatisticsOverview(apiKey, query), getStatisticsOverview(apiKey, previousQuery)]);
      setOverview(current);
      setPrevious(prior.summary);
      setFacets((existing) => ({
        providers: mergeBreakdowns(existing.providers, current.providers),
        models: mergeBreakdowns(existing.models, current.models),
        protocols: mergeBreakdowns(existing.protocols, current.protocols),
        accessKeys: mergeBreakdowns(existing.accessKeys, current.access_keys),
      }));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "无法读取统计数据");
    } finally {
      setLoading(false);
    }
  }, [apiKey, currentQuery]);

  useEffect(() => { void load(); }, [load]);

  async function exportCSV() {
    setExporting(true);
    setError("");
    try {
      await downloadStatisticsCSV(apiKey, currentQuery());
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "无法导出统计数据");
    } finally {
      setExporting(false);
    }
  }

  function updateFilter(name: keyof StatisticsFilters, value: string) {
    setFilters((current) => ({ ...current, [name]: value }));
  }

  const hasFilters = Object.values(filters).some(Boolean);

  return (
    <div className="statistics-panel">
      <div className="statistics-toolbar">
        <div className="status-filter" aria-label="统计时间范围">
          {ranges.map((item) => <button key={item.id} className={range === item.id ? "active" : ""} onClick={() => setRange(item.id)}>{item.label}</button>)}
          <button className={range === "custom" ? "active" : ""} onClick={() => setRange("custom")}><CalendarDays size={13} />自定义</button>
        </div>
        <div className="statistics-toolbar-actions">
          <button className="icon-button statistics-refresh" title="导出 CSV" aria-label="导出 CSV" onClick={() => void exportCSV()} disabled={exporting || loading}><Download size={16} /></button>
          <button className="icon-button statistics-refresh" title="刷新统计" aria-label="刷新统计" onClick={() => void load()} disabled={loading}><RefreshCw size={16} className={loading ? "spin" : ""} /></button>
        </div>
      </div>

      {range === "custom" && <div className="statistics-custom-range">
        <label><span>开始</span><input type="datetime-local" value={customFrom} onChange={(event) => setCustomFrom(event.target.value)} /></label>
        <label><span>结束</span><input type="datetime-local" value={customTo} onChange={(event) => setCustomTo(event.target.value)} /></label>
      </div>}

      <div className="statistics-filters">
        <FilterSelect label="Provider" value={filters.provider ?? ""} items={facets.providers} onChange={(value) => updateFilter("provider", value)} />
        <FilterSelect label="模型" value={filters.model ?? ""} items={facets.models} onChange={(value) => updateFilter("model", value)} />
        <FilterSelect label="协议" value={filters.protocol ?? ""} items={facets.protocols} onChange={(value) => updateFilter("protocol", value)} />
        <FilterSelect label="访问密钥" value={filters.access_key ?? ""} items={facets.accessKeys} onChange={(value) => updateFilter("access_key", value)} />
        <label><span>状态</span><select value={filters.status ?? ""} onChange={(event) => updateFilter("status", event.target.value)}><option value="">全部状态</option><option value="success">成功</option><option value="failure">失败</option></select></label>
        {hasFilters && <button className="statistics-clear" onClick={() => setFilters(emptyFilters)}>清除筛选</button>}
      </div>

      {error && <div className="statistics-error" role="alert"><AlertCircle size={16} />{error}<button onClick={() => void load()}>重试</button></div>}
      {loading && !overview && <div className="statistics-loading"><RefreshCw size={20} className="spin" /><span>正在汇总请求数据...</span></div>}
      {!loading && !error && overview?.summary.requests === 0 && <div className="statistics-empty"><BarChart3 size={28} /><strong>这个范围没有匹配的请求</strong><span>{hasFilters ? "调整筛选条件后再试。" : "新的 API 调用会自动出现在这里。"}</span></div>}
      {overview && overview.summary.requests > 0 && <div className={loading ? "statistics-content refreshing" : "statistics-content"}><StatisticsContent overview={overview} previous={previous} /></div>}
    </div>
  );
}

function StatisticsContent({ overview, previous }: { overview: StatisticsOverview; previous: StatisticsSummary | null }) {
  const summary = overview.summary;
  return <>
    <section className="statistics-metrics" aria-label="统计摘要">
      <Metric icon={Activity} label="请求数" value={compactNumber(summary.requests)} detail={comparisonDetail(summary.requests, previous?.requests)} />
      <Metric icon={BarChart3} label="成功率" value={`${summary.success_rate.toFixed(1)}%`} detail={pointComparison(summary.success_rate, previous?.success_rate)} />
      <Metric icon={ArrowDownToLine} label="输入 Tokens" value={compactNumber(summary.input_tokens)} detail={comparisonDetail(summary.input_tokens, previous?.input_tokens)} />
      <Metric icon={ArrowUpFromLine} label="输出 Tokens" value={compactNumber(summary.output_tokens)} detail={comparisonDetail(summary.output_tokens, previous?.output_tokens)} />
      <Metric icon={Gauge} label="Usage 完整率" value={`${summary.usage_rate.toFixed(1)}%`} detail={`${summary.known_usage}/${summary.requests} 个请求`} />
      <Metric icon={Gauge} label="P95 延迟" value={formatDuration(summary.p95_latency_ms)} detail={`P50 ${formatDuration(summary.p50_latency_ms)}`} />
    </section>
    <section className="statistics-token-details" aria-label="性能与 Token 明细">
      <span>总 Tokens<strong>{compactNumber(summary.total_tokens)}</strong></span>
      <span>缓存读取<strong>{compactNumber(summary.cached_tokens)}</strong></span>
      <span>缓存写入<strong>{compactNumber(summary.cache_write_tokens)}</strong></span>
      <span>推理<strong>{compactNumber(summary.reasoning_tokens)}</strong></span>
      <span>平均 TTFT<strong>{summary.ttft_samples > 0 ? formatDuration(summary.average_ttft_ms) : "-"}</strong></span>
      <span>P95 TTFT<strong>{summary.ttft_samples > 0 ? formatDuration(summary.p95_ttft_ms) : "-"}</strong></span>
      <span>输出速率<strong>{summary.output_tokens_per_second > 0 ? `${summary.output_tokens_per_second.toFixed(1)} tok/s` : "-"}</strong></span>
    </section>
    <section className="statistics-trend">
      <div className="statistics-section-title"><div><strong>请求与 Token 趋势</strong><span>{formatRange(overview.from, overview.to)}</span></div><div className="chart-legend"><i className="requests" />请求<i className="input" />输入<i className="output" />输出</div></div>
      <TrendChart points={overview.series} />
    </section>
    <div className="statistics-breakdowns">
      <BreakdownTable title="Provider" items={overview.providers} />
      <BreakdownTable title="模型" items={overview.models} />
    </div>
    <section className="statistics-failures">
      <div className="statistics-section-title"><div><strong>最近失败</strong><span>HTTP 4xx 与 5xx 请求，不保存响应正文</span></div></div>
      {overview.recent_failures.length === 0 ? <div className="statistics-no-failures">当前范围内没有失败请求</div> : <div className="statistics-table-scroll"><table><thead><tr><th>时间</th><th>状态</th><th>协议</th><th>模型</th><th>Provider</th><th>延迟</th><th>Request ID</th></tr></thead><tbody>{overview.recent_failures.map((item) => <tr key={item.request_id}><td>{new Date(item.created_at).toLocaleString("zh-CN", { hour12: false })}</td><td><span className="failure-status">{item.status_code}</span></td><td>{item.protocol}</td><td>{item.client_model || "-"}</td><td>{item.provider_name || "-"}</td><td>{formatDuration(item.latency_ms)}</td><td><code title={item.request_id}>{item.request_id.slice(0, 12)}</code></td></tr>)}</tbody></table></div>}
    </section>
    <footer className="statistics-footnote">数据保留 {overview.retention_days} 天{overview.dropped_events > 0 ? ` · 写入队列繁忙时已丢弃 ${overview.dropped_events} 条` : ""}</footer>
  </>;
}

function Metric({ icon: Icon, label, value, detail }: { icon: typeof Activity; label: string; value: string; detail?: string }) {
  return <div><Icon size={18} /><span>{label}</span><strong>{value}</strong>{detail && <small>{detail}</small>}</div>;
}

function FilterSelect({ label, value, items, onChange }: { label: string; value: string; items: StatisticsBreakdown[]; onChange: (value: string) => void }) {
  const selectedMissing = value && !items.some((item) => item.id === value);
  return <label><span>{label}</span><select value={value} onChange={(event) => onChange(event.target.value)}><option value="">全部{label}</option>{selectedMissing && <option value={value}>{value}</option>}{items.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>;
}

function BreakdownTable({ title, items }: { title: string; items: StatisticsBreakdown[] }) {
  return <section><div className="statistics-section-title"><div><strong>{title}</strong><span>按请求量排序</span></div></div><div className="statistics-table-scroll statistics-breakdown-table"><table><thead><tr><th>名称</th><th>请求</th><th>Usage</th><th>输入</th><th>输出</th><th>总计</th></tr></thead><tbody>{items.map((item) => <tr key={`${item.id}-${item.name}`}><td title={`${item.name} · 成功率 ${item.success_rate.toFixed(1)}%`}>{item.name}</td><td>{compactNumber(item.requests)}</td><td>{item.usage_rate.toFixed(0)}%</td><td>{compactNumber(item.input_tokens)}</td><td>{compactNumber(item.output_tokens)}</td><td>{compactNumber(item.total_tokens)}</td></tr>)}</tbody></table>{items.length === 0 && <div className="statistics-table-empty">暂无数据</div>}</div></section>;
}

function TrendChart({ points }: { points: StatisticsPoint[] }) {
  const width = 960, height = 230, left = 44, right = 18, top = 16, bottom = 32;
  const plotWidth = width - left - right, plotHeight = height - top - bottom;
  const maxRequests = Math.max(1, ...points.map((point) => point.requests));
  const maxTokens = Math.max(1, ...points.map((point) => Math.max(point.input_tokens, point.output_tokens)));
  const path = useMemo(() => chartPath(points, (point) => point.requests, maxRequests, left, top, plotWidth, plotHeight), [points, maxRequests]);
  const inputPath = useMemo(() => chartPath(points, (point) => point.input_tokens, maxTokens, left, top, plotWidth, plotHeight), [points, maxTokens]);
  const outputPath = useMemo(() => chartPath(points, (point) => point.output_tokens, maxTokens, left, top, plotWidth, plotHeight), [points, maxTokens]);
  const labels = points.length > 1 ? [points[0], points[Math.floor((points.length - 1) / 2)], points[points.length - 1]] : points;
  const singlePoint = points.length === 1 ? points[0] : undefined;
  return <div className="statistics-chart"><svg viewBox={`0 0 ${width} ${height}`} role="img" aria-label="请求与 Token 趋势图" preserveAspectRatio="none">
    {[0, .25, .5, .75, 1].map((value) => <line key={value} x1={left} x2={width - right} y1={top + plotHeight * value} y2={top + plotHeight * value} className="chart-grid" />)}
    <path d={path} className="chart-request-line" /><path d={inputPath} className="chart-input-line" /><path d={outputPath} className="chart-output-line" />
    {singlePoint && singlePoint.requests > 0 && <path d={chartPointPath(singlePoint.requests, maxRequests, left, top, plotWidth, plotHeight, -16)} className="chart-request-point" aria-hidden="true" />}
    {singlePoint && singlePoint.input_tokens > 0 && <path d={chartPointPath(singlePoint.input_tokens, maxTokens, left, top, plotWidth, plotHeight)} className="chart-input-point" aria-hidden="true" />}
    {singlePoint && singlePoint.output_tokens > 0 && <path d={chartPointPath(singlePoint.output_tokens, maxTokens, left, top, plotWidth, plotHeight, 16)} className="chart-output-point" aria-hidden="true" />}
  </svg><span className="chart-y-label top">{compactNumber(maxRequests)}</span><span className="chart-y-label bottom">0</span><div className={`chart-x-labels ${points.length === 1 ? "single" : ""}`}>{labels.map((point) => <span key={point.time}>{new Date(point.time).toLocaleDateString("zh-CN", { month: "numeric", day: "numeric", hour: "2-digit" })}</span>)}</div></div>;
}

export function chartPointPath(value: number, max: number, left: number, top: number, width: number, height: number, xOffset = 0): string {
  const x = left + width / 2 + xOffset;
  const y = top + height - value / max * height;
  return `M ${x} ${y} l 0.01 0`;
}

function chartPath(points: StatisticsPoint[], value: (point: StatisticsPoint) => number, max: number, left: number, top: number, width: number, height: number): string {
  if (points.length === 0) return "";
  return points.map((point, index) => `${index === 0 ? "M" : "L"} ${left + (points.length === 1 ? width / 2 : index * width / (points.length - 1))} ${top + height - value(point) / max * height}`).join(" ");
}

function formatRange(from: string, to: string): string {
  const format = new Intl.DateTimeFormat("zh-CN", { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });
  return `${format.format(new Date(from))} - ${format.format(new Date(to))}`;
}

function comparisonDetail(current: number, previous?: number): string | undefined {
  if (previous === undefined) return undefined;
  if (previous === 0) return current === 0 ? "与上一周期持平" : "上一周期为 0";
  const change = (current - previous) * 100 / previous;
  return `${change >= 0 ? "+" : ""}${change.toFixed(1)}% 较上一周期`;
}

function pointComparison(current: number, previous?: number): string | undefined {
  if (previous === undefined) return undefined;
  const change = current - previous;
  return `${change >= 0 ? "+" : ""}${change.toFixed(1)} 个百分点`;
}

function mergeBreakdowns(existing: StatisticsBreakdown[], incoming: StatisticsBreakdown[]): StatisticsBreakdown[] {
  const result = new Map(existing.map((item) => [item.id, item]));
  for (const item of incoming) result.set(item.id, item);
  return [...result.values()].sort((a, b) => a.name.localeCompare(b.name, "zh-CN"));
}
