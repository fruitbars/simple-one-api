import { describe, expect, it } from "vitest";
import {
  conversationStorageKey,
  conversationTitle,
  loadConversations,
  persistedMessages,
  saveConversations,
  type StoredConversation,
} from "./conversationStore";

function memoryStorage(initial?: string): Storage {
  const values = new Map<string, string>();
  if (initial !== undefined) values.set(conversationStorageKey, initial);
  return {
    get length() { return values.size; },
    clear: () => values.clear(),
    getItem: (key) => values.get(key) ?? null,
    key: (index) => [...values.keys()][index] ?? null,
    removeItem: (key) => { values.delete(key); },
    setItem: (key, value) => { values.set(key, value); },
  };
}

function conversation(id: string, updatedAt: number): StoredConversation {
  return {
    id, title: id, model: "model-a", createdAt: updatedAt, updatedAt,
    messages: [{ id: `${id}-message`, role: "user", content: id, status: "complete" }],
  };
}

describe("conversation storage", () => {
  it("ignores malformed storage", () => {
    expect(loadConversations(memoryStorage("not json"))).toEqual([]);
    expect(loadConversations(memoryStorage(JSON.stringify([{ id: 1 }])))).toEqual([]);
  });

  it("sorts newest first and limits history", () => {
    const storage = memoryStorage();
    const saved = saveConversations(Array.from({ length: 55 }, (_, index) => conversation(String(index), index)), storage);
    expect(saved).toHaveLength(50);
    expect(loadConversations(storage)[0].id).toBe("54");
  });

  it("does not persist an in-progress assistant message", () => {
    const messages = [
      { id: "user", role: "user" as const, content: "hello", status: "complete" as const },
      { id: "assistant", role: "assistant" as const, content: "par", status: "streaming" as const },
    ];
    expect(persistedMessages(messages)).toEqual([messages[0]]);
    expect(conversationTitle(messages)).toBe("hello");
  });
});
