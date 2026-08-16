export type Role = "user" | "assistant" | "system";

export interface ChatMessage {
  id: string;
  role: Role;
  content: string;
  reasoningContent?: string;
  status?: "streaming" | "complete" | "error";
  metrics?: {
    promptTokens?: number;
    completionTokens?: number;
    totalTokens?: number;
    durationMs: number;
    tokensPerSecond?: number;
  };
}

export interface ModelItem {
  id: string;
}

export interface ModelsResponse {
  data?: ModelItem[];
}
