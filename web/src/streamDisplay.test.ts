import { describe, expect, it } from "vitest";
import { takeDisplayChunk } from "./streamDisplay";

describe("stream display helpers", () => {
  it("keeps short deltas intact", () => {
    expect(takeDisplayChunk("hey")).toEqual(["hey", ""]);
  });

  it("splits large buffered responses into readable chunks", () => {
    const [chunk, rest] = takeDisplayChunk("abcdefghijklmnopqrstuvwxyz0123456789");
    expect(chunk.length).toBeGreaterThanOrEqual(4);
    expect(chunk.length).toBeLessThan(36);
    expect(chunk + rest).toBe("abcdefghijklmnopqrstuvwxyz0123456789");
  });

  it("prefers an early newline as a chunk boundary", () => {
    expect(takeDisplayChunk("first line\nsecond line")).toEqual(["first line\n", "second line"]);
  });
});
