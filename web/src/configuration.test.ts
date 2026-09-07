import { describe, expect, it } from "vitest";
import {
  createService,
  displayStringList,
  removeModelAlias,
  scalarCredentialEntries,
  setModelAlias,
  stringList,
  upstreamEndpointPreview,
} from "./configuration";

describe("configuration helpers", () => {
  it("normalizes comma and newline separated model lists", () => {
    expect(stringList(" model-a,model-b\nmodel-a, ,model-c ")).toEqual(["model-a", "model-b", "model-c"]);
    expect(displayStringList(["model-a", "model-b"])).toBe("model-a, model-b");
  });

  it("accepts common model separators", () => {
    expect(stringList("model-a，model-b； model-c\nmodel-d")).toEqual(["model-a", "model-b", "model-c", "model-d"]);
  });

  it("previews the final upstream endpoint for the selected protocol", () => {
    expect(upstreamEndpointPreview("openai", "responses", "https://example.com/v2")).toBe("https://example.com/v2/responses");
    expect(upstreamEndpointPreview("openai", "chat_completions", "https://example.com/v2")).toBe("https://example.com/v2/chat/completions");
  });

  it("creates a provider draft with a stable editable shape", () => {
    const service = createService("openai");
    expect(service.id).toMatch(/^[0-9a-f-]{36}$/i);
    expect(service).toMatchObject({
      provider: "openai",
      upstream_protocol: "auto",
      enabled: true,
      models: [],
      credentials: {},
      limit: {},
      timeout: 120,
    });
  });

  it("only exposes scalar credentials to the visual form", () => {
    expect(
      scalarCredentialEntries({ api_key: "secret", retries: 2, enabled: true, nested: { client: "hidden" } }),
    ).toEqual([
      ["api_key", "secret"],
      ["retries", 2],
      ["enabled", true],
    ]);
  });

  it("keeps provider aliases and public model names in sync", () => {
    expect(setModelAlias(["gpt-4o"], {}, "", "fast", "gpt-4o")).toEqual({
      models: ["gpt-4o", "fast"],
      modelMap: { fast: "gpt-4o" },
    });
    expect(setModelAlias(["gpt-4o", "fast"], { fast: "gpt-4o" }, "fast", "quick", "gpt-4.1-mini")).toEqual({
      models: ["gpt-4o", "quick"],
      modelMap: { quick: "gpt-4.1-mini" },
    });
    expect(removeModelAlias(["gpt-4o", "quick"], { quick: "gpt-4.1-mini" }, "quick")).toEqual({
      models: ["gpt-4o"],
      modelMap: {},
    });
  });
});
