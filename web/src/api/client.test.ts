import { describe, expect, it } from "vitest";
import { isDesktopAssetProtocol, parseSSEBlock, parseSSEBlockDetailed } from "./client";

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

  it("uses buffered responses for the Wails asset protocol", () => {
    expect(isDesktopAssetProtocol("wails:")).toBe(true);
    expect(isDesktopAssetProtocol("http:")).toBe(false);
    expect(isDesktopAssetProtocol("https:")).toBe(false);
  });

  it("extracts usage from a final streaming chunk", () => {
    const result = parseSSEBlockDetailed('data: {"choices":[],"usage":{"prompt_tokens":12,"completion_tokens":24,"total_tokens":36}}');
    expect(result).toEqual({
      content: [],
      usage: { prompt_tokens: 12, completion_tokens: 24, total_tokens: 36 },
    });
  });
});
