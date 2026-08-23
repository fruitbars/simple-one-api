import type { ChatMessage, ModelsResponse } from "../types";

interface ChatChunk {
  choices?: Array<{
    delta?: { content?: string; reasoning?: string; reasoning_content?: string };
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

function chatRequestBody(options: StreamChatOptions): string {
  return JSON.stringify({
    model: options.model,
    stream: true,
    stream_options: { include_usage: true },
    messages: options.messages.map(({ role, content }) => ({ role, content })),
    ...(options.thinking ? { reasoning_effort: "medium", enable_thinking: true, chat_template_kwargs: { enable_thinking: true } } : {}),
  });
}

function dispatchSSEBlock(block: string, options: StreamChatOptions): TokenUsage | undefined {
  const parsed = parseSSEBlockDetailed(block);
  for (const content of parsed.content) options.onDelta(content);
  for (const thought of parsed.reasoning) options.onReasoningDelta?.(thought);
  return parsed.usage;
}

function decodeBase64(value: string): Uint8Array {
  const binary = atob(value);
  return Uint8Array.from(binary, (character) => character.charCodeAt(0));
}

export async function desktopStreamChat(options: StreamChatOptions): Promise<TokenUsage | undefined> {
  const bridge = window.go?.main?.DesktopBridge;
  const events = window.runtime;
  if (!bridge || !events) {
    throw new Error("桌面流式接口不可用，请重新编译并启动桌面应用");
  }

  const requestID = crypto.randomUUID();
  const eventName = `simple-one-api:chat:${requestID}`;
  const decoder = new TextDecoder();
  let buffer = "";
  let usage: TokenUsage | undefined;
  let streamError: Error | undefined;

  const consumeText = (text: string) => {
    buffer += text;
    const blocks = buffer.split(/\r?\n\r?\n/);
    buffer = blocks.pop() ?? "";
    for (const block of blocks) {
      usage = dispatchSSEBlock(block, options) ?? usage;
    }
  };
  const unsubscribe = events.EventsOn(eventName, (...data: unknown[]) => {
    if (streamError) return;
    try {
      const encoded = data[0];
      if (typeof encoded !== "string") throw new Error("桌面应用返回了无效的流式数据");
      consumeText(decoder.decode(decodeBase64(encoded), { stream: true }));
    } catch (reason) {
      streamError = reason instanceof Error ? reason : new Error("无法解析桌面流式响应");
      void bridge.CancelChat(requestID).catch(() => undefined);
    }
  });
  const cancel = () => void bridge.CancelChat(requestID).catch(() => undefined);
  options.signal.addEventListener("abort", cancel, { once: true });

  try {
    if (options.signal.aborted) {
      cancel();
      throw new DOMException("The operation was aborted", "AbortError");
    }
    try {
      await bridge.StreamChat(requestID, options.apiKey.trim(), chatRequestBody(options));
    } catch (reason) {
      if (streamError) throw streamError;
      throw reason;
    }
    consumeText(decoder.decode());
    if (buffer.trim()) usage = dispatchSSEBlock(buffer, options) ?? usage;
    if (streamError) throw streamError;
    return usage;
  } finally {
    options.signal.removeEventListener("abort", cancel);
    unsubscribe();
  }
}

export async function streamChat(options: StreamChatOptions): Promise<TokenUsage | undefined> {
  if (isDesktopAssetProtocol()) {
    return desktopStreamChat(options);
  }
  const response = await fetch("/v1/chat/completions", {
    method: "POST",
    headers: headers(options.apiKey),
    signal: options.signal,
    body: chatRequestBody(options),
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
      usage = dispatchSSEBlock(block, options) ?? usage;
    }
    if (done) break;
  }

  if (buffer.trim()) {
    usage = dispatchSSEBlock(buffer, options) ?? usage;
  }
  return usage;
}
