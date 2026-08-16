import { parse as parseYaml, stringify as stringifyYaml } from "yaml";
import type { ConfigurationDocument } from "./api/admin";

export type SourceFormat = "json" | "yaml";

function asDocument(value: unknown): ConfigurationDocument {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    throw new Error("配置根节点必须是对象。");
  }
  return value as ConfigurationDocument;
}

export function parseSourceDocument(content: string, format: SourceFormat): ConfigurationDocument {
  return asDocument(format === "json" ? JSON.parse(content) : parseYaml(content));
}

export function sourceFormatFromFilename(filename: string): SourceFormat {
  return filename.toLowerCase().endsWith(".json") ? "json" : "yaml";
}

export function serializeSourceDocument(document: ConfigurationDocument, format: SourceFormat): string {
  return format === "json" ? `${JSON.stringify(document, null, 2)}\n` : stringifyYaml(document);
}
