import { describe, expect, it } from "vitest";
import { createService, displayStringList, scalarCredentialEntries, stringList } from "./configuration";

describe("configuration helpers", () => {
  it("normalizes comma and newline separated model lists", () => {
    expect(stringList(" model-a,model-b\nmodel-a, ,model-c ")).toEqual(["model-a", "model-b", "model-c"]);
    expect(displayStringList(["model-a", "model-b"])).toBe("model-a, model-b");
  });

  it("creates a provider draft with a stable editable shape", () => {
    const service = createService("openai");
    expect(service.id).toMatch(/^[0-9a-f-]{36}$/i);
    expect(service).toMatchObject({
      provider: "openai",
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
});
