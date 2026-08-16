import type { ChatMessage } from "./types";

export const conversationStorageKey = "simple-one-conversations-v1";
export const activeConversationStorageKey = "simple-one-active-conversation-v1";
const maximumConversations = 50;
const maximumSerializedBytes = 4 * 1024 * 1024;

export interface StoredConversation {
  id: string;
  title: string;
  model: string;
  messages: ChatMessage[];
  createdAt: number;
  updatedAt: number;
}

function validMessage(value: unknown): value is ChatMessage {
  if (!value || typeof value !== "object") return false;
  const message = value as Partial<ChatMessage>;
  return typeof message.id === "string"
    && (message.role === "user" || message.role === "assistant" || message.role === "system")
    && typeof message.content === "string";
}

function normalizeConversation(value: unknown): StoredConversation | null {
  if (!value || typeof value !== "object") return null;
  const item = value as Partial<StoredConversation>;
  if (typeof item.id !== "string" || !Array.isArray(item.messages)) return null;
  const messages = item.messages.filter(validMessage).map((message) => ({
    ...message,
    status: message.status === "error" ? "error" as const : "complete" as const,
  }));
  if (!messages.some((message) => message.role === "user")) return null;
  const fallbackTitle = messages.find((message) => message.role === "user")?.content.trim().slice(0, 34) || "新对话";
  return {
    id: item.id,
    title: typeof item.title === "string" && item.title.trim() ? item.title.trim().slice(0, 60) : fallbackTitle,
    model: typeof item.model === "string" ? item.model : "",
    messages,
    createdAt: typeof item.createdAt === "number" ? item.createdAt : Date.now(),
    updatedAt: typeof item.updatedAt === "number" ? item.updatedAt : Date.now(),
  };
}

export function loadConversations(storage: Storage = localStorage): StoredConversation[] {
  try {
    const raw = storage.getItem(conversationStorageKey);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) return [];
    return parsed
      .map(normalizeConversation)
      .filter((item): item is StoredConversation => item !== null)
      .sort((left, right) => right.updatedAt - left.updatedAt)
      .slice(0, maximumConversations);
  } catch {
    return [];
  }
}

export function saveConversations(conversations: StoredConversation[], storage: Storage = localStorage): StoredConversation[] {
  const next = [...conversations]
    .sort((left, right) => right.updatedAt - left.updatedAt)
    .slice(0, maximumConversations);
  while (next.length > 0 && new Blob([JSON.stringify(next)]).size > maximumSerializedBytes) {
    next.pop();
  }
  try {
    storage.setItem(conversationStorageKey, JSON.stringify(next));
  } catch {
    return conversations;
  }
  return next;
}

export function persistedMessages(messages: ChatMessage[]): ChatMessage[] {
  return messages
    .filter((message) => message.status !== "streaming")
    .map((message) => ({ ...message, status: message.status === "error" ? "error" : "complete" }));
}

export function conversationTitle(messages: ChatMessage[]): string {
  return messages.find((message) => message.role === "user")?.content.trim().slice(0, 34) || "新对话";
}
