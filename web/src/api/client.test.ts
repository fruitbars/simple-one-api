import { afterEach, describe, expect, it, vi } from "vitest";
import { desktopStreamChat, isDesktopAssetProtocol, parseSSEBlock, parseSSEBlockDetailed } from "./client";

afterEach(() => {
  delete window.go;
  delete window.runtime;
  vi.restoreAllMocks();
});

describe("parseSSEBlock", () => {
  it("extracts streamed OpenAI deltas", () => {
    const result = parseSSEBlock(
      'data: {"choices":[{"delta":{"content":"hello"}}]}\n' +
        'data: {"choices":[{"delta":{"content":" world"}}]}',
    );
    expect(result).toEqual(["hello", " world"]);
  });

  it("ignores comments, empty data, and the done marker", () => {
    expect(parseSSEBlock(": keepalive\ndata:\ndata: [DONE]")).toEqual([]);
  });

  it("recognizes the Wails asset protocol", () => {
    expect(isDesktopAssetProtocol("wails:")).toBe(true);
    expect(isDesktopAssetProtocol("http:")).toBe(false);
    expect(isDesktopAssetProtocol("https:")).toBe(false);
  });

  it("extracts usage from a final streaming chunk", () => {
    const result = parseSSEBlockDetailed('data: {"choices":[],"usage":{"prompt_tokens":12,"completion_tokens":24,"total_tokens":36}}');
    expect(result).toEqual({
      content: [],
      reasoning: [],
      usage: { prompt_tokens: 12, completion_tokens: 24, total_tokens: 36 },
    });
  });

  it("extracts streamed reasoning separately from answer content", () => {
    const result = parseSSEBlockDetailed(
      'data: {"choices":[{"delta":{"reasoning_content":"inspect","content":"answer"}}]}',
    );
    expect(result).toEqual({ content: ["answer"], reasoning: ["inspect"], usage: undefined });
  });

  it("streams desktop reasoning and answer chunks before completion", async () => {
    vi.spyOn(crypto, "randomUUID").mockReturnValue("00000000-0000-4000-8000-000000000001");
    let listener: ((...data: unknown[]) => void) | undefined;
    const unsubscribe = vi.fn();
    const streamChat = vi.fn(async (_id: string, _key: string, payload: string) => {
      expect(JSON.parse(payload)).toMatchObject({ stream: true, enable_thinking: true });
      const reasoningChunk = new TextEncoder().encode(
        'data: {"choices":[{"delta":{"reasoning_content":"先分析"}}]}\n\n',
      );
      const firstNonASCIIByte = reasoningChunk.findIndex((value) => value > 127);
      const chunks = [
        reasoningChunk.slice(0, firstNonASCIIByte + 1),
        reasoningChunk.slice(firstNonASCIIByte + 1),
        new TextEncoder().encode(
        'data: {"choices":[{"delta":{"content":"答案"}}]}\n\n',
        ),
        new TextEncoder().encode(
        'data: {"choices":[],"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}}\n\n',
        ),
      ];
      for (const chunk of chunks) listener?.(btoa(String.fromCharCode(...chunk)));
    });
    window.go = { main: { DesktopBridge: { StreamChat: streamChat, CancelChat: vi.fn(async () => undefined) } } };
    window.runtime = {
      EventsOn: vi.fn((_name, callback) => {
        listener = callback;
        return unsubscribe;
      }),
    };
    const content: string[] = [];
    const reasoning: string[] = [];

    const usage = await desktopStreamChat({
      model: "reasoner",
      messages: [{ id: "message", role: "user", content: "hello" }],
      apiKey: " key ",
      signal: new AbortController().signal,
      thinking: true,
      onDelta: (value) => content.push(value),
      onReasoningDelta: (value) => reasoning.push(value),
    });

    expect(reasoning).toEqual(["先分析"]);
    expect(content).toEqual(["答案"]);
    expect(usage).toEqual({ prompt_tokens: 3, completion_tokens: 2, total_tokens: 5 });
    expect(streamChat).toHaveBeenCalledWith("00000000-0000-4000-8000-000000000001", "key", expect.any(String));
    expect(unsubscribe).toHaveBeenCalledOnce();
  });

  it("cancels an in-progress desktop request", async () => {
    vi.spyOn(crypto, "randomUUID").mockReturnValue("00000000-0000-4000-8000-000000000002");
    const controller = new AbortController();
    const cancelChat = vi.fn(async () => undefined);
    window.runtime = { EventsOn: vi.fn(() => vi.fn()) };
    window.go = { main: { DesktopBridge: {
      CancelChat: cancelChat,
      StreamChat: vi.fn(() => new Promise<void>((_resolve, reject) => {
        controller.signal.addEventListener("abort", () => reject(new Error("context canceled")), { once: true });
      })),
    } } };

    const request = desktopStreamChat({
      model: "model",
      messages: [],
      apiKey: "",
      signal: controller.signal,
      onDelta: vi.fn(),
    });
    controller.abort();

    await expect(request).rejects.toThrow("context canceled");
    expect(cancelChat).toHaveBeenCalledWith("00000000-0000-4000-8000-000000000002");
  });
});
