import { describe, expect, it } from "vitest";
import { parseSourceDocument, serializeSourceDocument, sourceFormatFromFilename } from "./sourceDocument";

describe("source documents", () => {
  it("parses JSON and YAML configuration objects", () => {
    expect(parseSourceDocument('{"server_port":":9090"}', "json")).toEqual({ server_port: ":9090" });
    expect(parseSourceDocument("server_port: :9090\nenable_web: true\n", "yaml")).toEqual({
      server_port: ":9090",
      enable_web: true,
    });
  });

  it("rejects non-object roots", () => {
    expect(() => parseSourceDocument("[]", "json")).toThrow("配置根节点必须是对象");
    expect(() => parseSourceDocument("hello", "yaml")).toThrow("配置根节点必须是对象");
  });

  it("detects filenames and serializes both formats", () => {
    const document = { debug: false, models: ["demo"] };
    expect(sourceFormatFromFilename("config.JSON")).toBe("json");
    expect(sourceFormatFromFilename("config.yml")).toBe("yaml");
    expect(serializeSourceDocument(document, "json")).toContain('"debug": false');
    expect(serializeSourceDocument(document, "yaml")).toContain("models:\n  - demo");
  });
});
