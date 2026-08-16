import type { ChatMessage, ModelsResponse } from "../types";

interface ChatChunk {
  choices?: Array<{
    delta?: { content?: string; reasoning?: string; reasoning_content?: string };
    message?: { content?: string; reasoning?: string; reasoning_content?: string };
  }>;
  usage?: TokenUsage;
}

interface ChatResponse {
  choices?: Array<{
    message?: { content?: string; reasoning?: string; reasoning_content?: string };
  }>;
  usage?: TokenUsage;
}

export interface TokenUsage {
  prompt_tokens?: number;
  completion_tokens?: number;
  total_tokens?: number;
}

interface StreamChatOptions {
  model: string;
  messages: ChatMessage[];
  apiKey: string;
  signal: AbortSignal;
  onDelta: (content: string) => void;
  onReasoningDelta?: (content: string) => void;
  thinking?: boolean;
}

function headers(apiKey: string): HeadersInit {
  const result: Record<string, string> = {
    "Content-Type": "application/json",
  };
  if (apiKey.trim()) {
    result.Authorization = `Bearer ${apiKey.trim()}`;
  }
  return result;
}

export async function getModels(apiKey: string): Promise<string[]> {
  const response = await fetch("/v1/models", { headers: headers(apiKey) });
  if (!response.ok) {
    throw new Error(`无法读取模型列表（HTTP ${response.status}）`);
  }
  const data = (await response.json()) as ModelsResponse;
  return (data.data ?? []).map((model) => model.id).filter(Boolean);
}

export function parseSSEBlock(block: string): string[] {
  return parseSSEBlockDetailed(block).content;
}

export function parseSSEBlockDetailed(block: string): { content: string[]; reasoning: string[]; usage?: TokenUsage } {
  const result: string[] = [];
  const reasoning: string[] = [];
  let usage: TokenUsage | undefined;
  for (const line of block.split(/\r?\n/)) {
    if (!line.startsWith("data:")) continue;
    const payload = line.slice(5).trimStart();
    if (!payload || payload === "[DONE]") continue;
    const parsed = JSON.parse(payload) as ChatChunk;
    if (parsed.usage) usage = parsed.usage;
    const choice = parsed.choices?.[0];
    const content = choice?.delta?.content ?? choice?.message?.content;
    const thought = choice?.delta?.reasoning_content ?? choice?.delta?.reasoning
      ?? choice?.message?.reasoning_content ?? choice?.message?.reasoning;
    if (content) result.push(content);
    if (thought) reasoning.push(thought);
  }
  return { content: result, reasoning, usage };
}

export function isDesktopAssetProtocol(protocol = window.location.protocol): boolean {
  return protocol === "wails:";
}

async function bufferedChat(options: StreamChatOptions): Promise<TokenUsage | undefined> {
  const response = await fetch("/v1/chat/completions", {
    method: "POST",
    headers: headers(options.apiKey),
    signal: options.signal,
    body: JSON.stringify({
      model: options.model,
      stream: false,
      messages: options.messages.map(({ role, content }) => ({ role, content })),
      ...(options.thinking ? { reasoning_effort: "medium", enable_thinking: true, chat_template_kwargs: { enable_thinking: true } } : {}),
    }),
  });
  if (!response.ok) {
    const detail = await response.text();
    throw new Error(detail || `请求失败（HTTP ${response.status}）`);
  }
  const payload = (await response.json()) as ChatResponse;
  const content = payload.choices?.[0]?.message?.content;
  const reasoning = payload.choices?.[0]?.message?.reasoning_content ?? payload.choices?.[0]?.message?.reasoning;
  if (reasoning) options.onReasoningDelta?.(reasoning);
  if (!content) throw new Error("模型返回了空响应");
  options.onDelta(content);
  return payload.usage;
}

export async function streamChat(options: StreamChatOptions): Promise<TokenUsage | undefined> {
  if (isDesktopAssetProtocol()) {
    return bufferedChat(options);
  }
  const response = await fetch("/v1/chat/completions", {
    method: "POST",
    headers: headers(options.apiKey),
    signal: options.signal,
    body: JSON.stringify({
      model: options.model,
      stream: true,
      stream_options: { include_usage: true },
      messages: options.messages.map(({ role, content }) => ({ role, content })),
      ...(options.thinking ? { reasoning_effort: "medium", enable_thinking: true, chat_template_kwargs: { enable_thinking: true } } : {}),
    }),
  });

  if (!response.ok) {
    const detail = await response.text();
    throw new Error(detail || `请求失败（HTTP ${response.status}）`);
  }
  if (!response.body) {
    throw new Error("浏览器没有收到流式响应体");
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  let usage: TokenUsage | undefined;

  while (true) {
    const { done, value } = await reader.read();
    buffer += decoder.decode(value, { stream: !done });
    const blocks = buffer.split(/\r?\n\r?\n/);
    buffer = blocks.pop() ?? "";
    for (const block of blocks) {
      const parsed = parseSSEBlockDetailed(block);
      for (const content of parsed.content) options.onDelta(content);
      for (const thought of parsed.reasoning) options.onReasoningDelta?.(thought);
      if (parsed.usage) usage = parsed.usage;
    }
    if (done) break;
  }

  if (buffer.trim()) {
    const parsed = parseSSEBlockDetailed(buffer);
    for (const content of parsed.content) options.onDelta(content);
    for (const thought of parsed.reasoning) options.onReasoningDelta?.(thought);
    if (parsed.usage) usage = parsed.usage;
  }
  return usage;
}
