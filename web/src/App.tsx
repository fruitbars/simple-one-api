import {
  Activity,
  Brain,
  ArrowDown,
  Check,
  ChevronDown,
  CircleStop,
  Copy,
  Database,
  Menu,
  MessageSquare,
  Moon,
  Plus,
  RefreshCw,
  SendHorizontal,
  Settings,
  Sun,
  Trash2,
  X,
} from "lucide-react";
import { FormEvent, lazy, Suspense, useEffect, useRef, useState } from "react";
import { getModels, streamChat } from "./api/client";
import { AdminWorkspace } from "./AdminWorkspace";
import {
  activeConversationStorageKey,
  conversationTitle,
  loadConversations,
  persistedMessages,
  saveConversations,
  type StoredConversation,
} from "./conversationStore";
import { takeDisplayChunk } from "./streamDisplay";
import type { ChatMessage } from "./types";

const MarkdownMessage = lazy(() => import("./MarkdownMessage"));

const welcomePrompts = [
  "解释一下这个模型网关的工作方式",
  "帮我写一个 OpenAI 兼容接口调用示例",
  "比较当前可用模型的特点",
];

function id(): string {
  return crypto.randomUUID();
}

function formatDuration(durationMs: number): string {
  return durationMs < 1000 ? `${Math.round(durationMs)} ms` : `${(durationMs / 1000).toFixed(1)} s`;
}

export function App() {
  const [surface, setSurface] = useState<"chat" | "admin">(() =>
    window.location.pathname.startsWith("/chat") ? "chat" : "admin",
  );
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [theme, setTheme] = useState<"light" | "dark">(() =>
    localStorage.getItem("simple-one-theme") === "dark" ? "dark" : "light",
  );
  const [apiKey, setApiKey] = useState(() => sessionStorage.getItem("simple-one-api-key") ?? "");
  const [conversations, setConversations] = useState<StoredConversation[]>(() => loadConversations());
  const [activeConversationID, setActiveConversationID] = useState<string | null>(() => {
    const stored = localStorage.getItem(activeConversationStorageKey);
    return conversations.some((conversation) => conversation.id === stored) ? stored : conversations[0]?.id ?? null;
  });
  const initialConversation = conversations.find((conversation) => conversation.id === activeConversationID);
  const [models, setModels] = useState<string[]>([]);
  const [model, setModel] = useState(initialConversation?.model ?? "");
  const [modelMenuOpen, setModelMenuOpen] = useState(false);
  const [messages, setMessages] = useState<ChatMessage[]>(initialConversation?.messages ?? []);
  const [draft, setDraft] = useState("");
  const [thinking, setThinking] = useState(() => localStorage.getItem("simple-one-thinking") === "true");
  const [loadingModels, setLoadingModels] = useState(true);
  const [error, setError] = useState("");
  const [atChatBottom, setAtChatBottom] = useState(true);
  const abortRef = useRef<AbortController | null>(null);
  const chatScrollRef = useRef<HTMLElement | null>(null);
  const followOutputRef = useRef(true);
  const endRef = useRef<HTMLDivElement | null>(null);
  const streaming = messages.some((message) => message.status === "streaming");

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem("simple-one-theme", theme);
  }, [theme]);

  useEffect(() => {
    if (!activeConversationID || !messages.some((message) => message.role === "user")) return;
    const timer = window.setTimeout(() => {
      const snapshot = persistedMessages(messages);
      if (!snapshot.some((message) => message.role === "user")) return;
      setConversations((current) => {
        const existing = current.find((conversation) => conversation.id === activeConversationID);
        const now = Date.now();
        const updated: StoredConversation = {
          id: activeConversationID,
          title: conversationTitle(snapshot),
          model,
          messages: snapshot,
          createdAt: existing?.createdAt ?? now,
          updatedAt: now,
        };
        return saveConversations([updated, ...current.filter((conversation) => conversation.id !== activeConversationID)]);
      });
      localStorage.setItem(activeConversationStorageKey, activeConversationID);
    }, 350);
    return () => window.clearTimeout(timer);
  }, [activeConversationID, messages, model]);

  useEffect(() => {
    void refreshModels();
  }, []);

  useEffect(() => {
    const onPopState = () => setSurface(window.location.pathname.startsWith("/chat") ? "chat" : "admin");
    window.addEventListener("popstate", onPopState);
    return () => window.removeEventListener("popstate", onPopState);
  }, []);

  useEffect(() => {
    if (!followOutputRef.current) return;
    const frame = window.requestAnimationFrame(() => {
      const container = chatScrollRef.current;
      if (!container) return;
      container.scrollTo({ top: container.scrollHeight, behavior: streaming ? "auto" : "smooth" });
    });
    return () => window.cancelAnimationFrame(frame);
  }, [messages, streaming]);

  function updateChatScrollState() {
    const container = chatScrollRef.current;
    if (!container) return;
    const nearBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 96;
    followOutputRef.current = nearBottom;
    setAtChatBottom(nearBottom);
  }

  function scrollChatToBottom() {
    followOutputRef.current = true;
    setAtChatBottom(true);
    const container = chatScrollRef.current;
    container?.scrollTo({ top: container.scrollHeight, behavior: "smooth" });
  }

  async function refreshModels() {
    setLoadingModels(true);
    setError("");
    try {
      const next = await getModels(apiKey);
      setModels(next);
      setModel((current) => (current && next.includes(current) ? current : next[0] ?? ""));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "无法连接到服务");
    } finally {
      setLoadingModels(false);
    }
  }

  function saveApiKey(value: string) {
    setApiKey(value);
    if (value) sessionStorage.setItem("simple-one-api-key", value);
    else sessionStorage.removeItem("simple-one-api-key");
  }

  function newConversation() {
    abortRef.current?.abort();
    setActiveConversationID(null);
    localStorage.removeItem(activeConversationStorageKey);
    setMessages([]);
    setDraft("");
    setError("");
    followOutputRef.current = true;
    setAtChatBottom(true);
    setSidebarOpen(false);
  }

  function selectConversation(conversation: StoredConversation) {
    abortRef.current?.abort();
    setActiveConversationID(conversation.id);
    localStorage.setItem(activeConversationStorageKey, conversation.id);
    setMessages(conversation.messages);
    if (conversation.model) setModel(conversation.model);
    setDraft("");
    setError("");
    followOutputRef.current = true;
    setAtChatBottom(true);
    setSidebarOpen(false);
  }

  function deleteConversation(conversationID: string) {
    const target = conversations.find((conversation) => conversation.id === conversationID);
    if (target && !window.confirm(`删除对话“${target.title}”？此操作无法撤销。`)) return;
    const remaining = conversations.filter((conversation) => conversation.id !== conversationID);
    setConversations(saveConversations(remaining));
    if (activeConversationID !== conversationID) return;
    abortRef.current?.abort();
    const next = remaining[0];
    setActiveConversationID(next?.id ?? null);
    setMessages(next?.messages ?? []);
    if (next?.model) setModel(next.model);
    if (next) localStorage.setItem(activeConversationStorageKey, next.id);
    else localStorage.removeItem(activeConversationStorageKey);
  }

  function navigate(next: "chat" | "admin") {
    const path = next === "admin" ? "/" : "/chat";
    window.history.pushState({}, "", path);
    setSurface(next);
    setSidebarOpen(false);
  }

  async function submit(event?: FormEvent, prompt = draft) {
    event?.preventDefault();
    const content = prompt.trim();
    if (!content || streaming || !model) return;

    const userMessage: ChatMessage = { id: id(), role: "user", content, status: "complete" };
    const assistantID = id();
    const nextMessages = [...messages, userMessage];
    if (!activeConversationID) {
      const conversationID = id();
      setActiveConversationID(conversationID);
      localStorage.setItem(activeConversationStorageKey, conversationID);
    }
    followOutputRef.current = true;
    setAtChatBottom(true);
    setMessages([...nextMessages, { id: assistantID, role: "assistant", content: "", status: "streaming" }]);
    setDraft("");
    setError("");

    const controller = new AbortController();
    abortRef.current = controller;
    const requestStartedAt = performance.now();
    let displayQueue = "";
    let displayTimer: number | null = null;
    const appendAssistantContent = (delta: string) => {
      if (!delta) return;
      setMessages((current) =>
        current.map((message) =>
          message.id === assistantID ? { ...message, content: message.content + delta } : message,
        ),
      );
    };
    const appendReasoningContent = (delta: string) => {
      if (!delta) return;
      setMessages((current) => current.map((message) =>
        message.id === assistantID ? { ...message, reasoningContent: (message.reasoningContent ?? "") + delta } : message,
      ));
    };
    const pumpDisplayQueue = () => {
      displayTimer = null;
      if (!displayQueue) return;
      const [chunk, rest] = takeDisplayChunk(displayQueue);
      displayQueue = rest;
      appendAssistantContent(chunk);
      if (displayQueue) displayTimer = window.setTimeout(pumpDisplayQueue, 18);
    };
    const scheduleDisplay = () => {
      if (displayTimer === null) displayTimer = window.setTimeout(pumpDisplayQueue, 12);
    };
    const drainDisplayQueue = () => new Promise<void>((resolve) => {
      if (displayTimer !== null) {
        window.clearTimeout(displayTimer);
        displayTimer = null;
      }
      const drain = () => {
        if (!displayQueue) {
          resolve();
          return;
        }
        const [chunk, rest] = takeDisplayChunk(displayQueue);
        displayQueue = rest;
        appendAssistantContent(chunk);
        displayTimer = window.setTimeout(drain, 22);
      };
      drain();
    });
    const flushDisplayQueue = () => {
      if (displayTimer !== null) {
        window.clearTimeout(displayTimer);
        displayTimer = null;
      }
      appendAssistantContent(displayQueue);
      displayQueue = "";
    };
    try {
      const usage = await streamChat({
        model,
        messages: nextMessages,
        apiKey,
        signal: controller.signal,
        onDelta(delta) {
          displayQueue += delta;
          scheduleDisplay();
        },
        thinking,
        onReasoningDelta: appendReasoningContent,
      });
      const durationMs = performance.now() - requestStartedAt;
      await drainDisplayQueue();
      setMessages((current) =>
        current.map((message) =>
          message.id === assistantID ? {
            ...message,
            status: "complete",
            metrics: {
              promptTokens: usage?.prompt_tokens,
              completionTokens: usage?.completion_tokens,
              totalTokens: usage?.total_tokens,
              durationMs,
              tokensPerSecond: usage?.completion_tokens
                ? usage.completion_tokens / Math.max(durationMs / 1000, 0.001)
                : undefined,
            },
          } : message,
        ),
      );
    } catch (reason) {
      flushDisplayQueue();
      if (controller.signal.aborted) {
        setMessages((current) =>
          current.map((message) =>
            message.id === assistantID ? { ...message, status: "complete" } : message,
          ),
        );
      } else {
        const detail = reason instanceof Error ? reason.message : "生成失败";
        setMessages((current) =>
          current.map((message) =>
            message.id === assistantID
              ? { ...message, content: message.content || detail, status: "error" }
              : message,
          ),
        );
      }
    } finally {
      abortRef.current = null;
    }
  }

  if (surface === "admin") {
    return <AdminWorkspace apiKey={apiKey} onApiKeyChange={saveApiKey} onBack={() => navigate("chat")} />;
  }

  return (
    <div className="app-shell">
      <button className="mobile-menu icon-button" onClick={() => setSidebarOpen(true)} aria-label="打开侧边栏">
        <Menu size={20} />
      </button>

      <aside className={`sidebar ${sidebarOpen ? "sidebar-open" : ""}`}>
        <div className="sidebar-header">
          <div className="brand-mark" aria-hidden="true">S</div>
          <span className="brand-name">Simple One <small>API</small></span>
          <button className="sidebar-close icon-button" onClick={() => setSidebarOpen(false)} aria-label="关闭侧边栏">
            <X size={18} />
          </button>
        </div>
        <div className="workspace-switch" aria-label="工作区切换">
          <button onClick={() => navigate("admin")}><Database size={16} />配置</button>
          <button className="active"><MessageSquare size={16} />Chat</button>
        </div>
        <button className="new-chat" onClick={newConversation}>
          <Plus size={17} />
          新对话
        </button>
        <div className="sidebar-section-label">最近</div>
        <div className="conversation-list">
          {conversations.map((conversation) => (
            <div className={`conversation-row ${activeConversationID === conversation.id ? "active" : ""}`} key={conversation.id}>
              <button className="conversation-item" onClick={() => selectConversation(conversation)} title={conversation.title}>
                <MessageSquare size={16} />
                <span>{conversation.title}</span>
              </button>
              <button className="conversation-delete icon-button" onClick={() => deleteConversation(conversation.id)} aria-label={`删除对话 ${conversation.title}`} title="删除对话"><Trash2 size={14} /></button>
            </div>
          ))}
          {conversations.length === 0 && <div className="conversation-empty">还没有保存的对话</div>}
        </div>
        <div className="sidebar-spacer" />
        <button className="sidebar-action" onClick={() => setTheme(theme === "light" ? "dark" : "light")}>
          {theme === "light" ? <Moon size={17} /> : <Sun size={17} />}
          {theme === "light" ? "深色模式" : "浅色模式"}
        </button>
        <button className="sidebar-action" onClick={() => setSettingsOpen(true)}>
          <Settings size={17} />
          连接设置
        </button>
      </aside>
      {sidebarOpen && <button className="sidebar-backdrop" onClick={() => setSidebarOpen(false)} aria-label="关闭侧边栏" />}

      <main className="main-panel">
        <header className="topbar">
          <div className="model-picker-wrap">
            <button className="model-picker" onClick={() => setModelMenuOpen(!modelMenuOpen)} disabled={loadingModels}>
              <span>{loadingModels ? "正在连接…" : model || "没有可用模型"}</span>
              <ChevronDown size={16} />
            </button>
            {modelMenuOpen && (
              <div className="model-menu">
                {models.map((item) => (
                  <button key={item} onClick={() => { setModel(item); setModelMenuOpen(false); }}>
                    <span>{item}</span>
                    {model === item && <Check size={16} />}
                  </button>
                ))}
                {!models.length && <div className="model-empty">请先配置并启用模型</div>}
              </div>
            )}
          </div>
          <div className={`connection-state ${error ? "connection-error" : ""}`}>
            <span className="status-dot" />
            {error ? "连接异常" : "已连接"}
          </div>
        </header>

        <section ref={chatScrollRef} className="chat-scroll" aria-label="对话内容" onScroll={updateChatScrollState}>
          {!messages.length ? (
            <div className="welcome">
              <div className="welcome-mark">S</div>
              <h1>今天想聊点什么？</h1>
              <p>通过你的 simple-one-api，在一个界面里使用已经配置好的模型。</p>
              {error && (
                <div className="inline-error" role="alert">
                  <span>{error}</span>
                  <div className="inline-error-actions">
                    <button onClick={() => void refreshModels()}><RefreshCw size={15} />重试</button>
                    {!models.length && <button onClick={() => navigate("admin")}><Database size={15} />去配置</button>}
                  </div>
                </div>
              )}
              <div className="prompt-grid">
                {welcomePrompts.map((prompt) => (
                  <button key={prompt} onClick={() => void submit(undefined, prompt)} disabled={!model}>
                    {prompt}
                  </button>
                ))}
              </div>
            </div>
          ) : (
            <div className="message-list">
              {messages.map((message) => (
                <article key={message.id} className={`message message-${message.role}`}>
                  <div className="message-avatar">{message.role === "user" ? "你" : "S"}</div>
                  <div className="message-body">
                    <div className="message-label">{message.role === "user" ? "你" : model}</div>
                    {message.role === "assistant" && message.reasoningContent && (
                      <details className="reasoning-panel" open={message.status === "streaming"}>
                        <summary><Brain size={14} />思考过程</summary>
                        <div>{message.reasoningContent}</div>
                      </details>
                    )}
                    <div className={`message-content ${message.status === "error" ? "message-error" : ""}`}>
                      {message.content ? message.role === "assistant" ? (
                        <Suspense fallback={<span className="message-plain">{message.content}</span>}>
                          <MarkdownMessage content={message.content} />
                        </Suspense>
                      ) : <span className="message-plain">{message.content}</span> : !message.reasoningContent && <span className="typing"><i /><i /><i /></span>}
                    </div>
                    {message.role === "assistant" && message.metrics && (
                      <div className="message-metrics" aria-label="响应统计">
                        <Activity size={13} />
                        {message.metrics.promptTokens != null && <span title="输入 Token">输入 {message.metrics.promptTokens}</span>}
                        {message.metrics.completionTokens != null && <span title="输出 Token">输出 {message.metrics.completionTokens}</span>}
                        {message.metrics.totalTokens != null && <span title="总 Token">合计 {message.metrics.totalTokens}</span>}
                        <span title="模型请求耗时">{formatDuration(message.metrics.durationMs)}</span>
                        {message.metrics.tokensPerSecond != null && <span title="平均输出速度">{message.metrics.tokensPerSecond.toFixed(1)} tok/s</span>}
                      </div>
                    )}
                    {message.role === "assistant" && message.content && message.status !== "streaming" && (
                      <div className="message-actions">
                        <button onClick={() => void navigator.clipboard.writeText(message.content)} aria-label="复制回答">
                          <Copy size={15} />
                        </button>
                      </div>
                    )}
                  </div>
                </article>
              ))}
              <div ref={endRef} />
            </div>
          )}
        </section>

        {!atChatBottom && messages.length > 0 && (
          <button className="scroll-to-bottom" onClick={scrollChatToBottom} aria-label="回到最新消息" title="回到最新消息">
            <ArrowDown size={18} />
          </button>
        )}

        <div className="composer-zone">
          <div className="composer-options">
            <button type="button" className={thinking ? "active" : ""} onClick={() => setThinking((current) => {
              localStorage.setItem("simple-one-thinking", String(!current));
              return !current;
            })} aria-pressed={thinking} title="为支持的模型启用思考模式">
              <Brain size={14} />思考
            </button>
          </div>
          <form className="composer" onSubmit={(event) => void submit(event)}>
            <textarea
              value={draft}
              onChange={(event) => setDraft(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter" && !event.shiftKey) {
                  event.preventDefault();
                  void submit();
                }
              }}
              placeholder={model ? `发送消息给 ${model}` : "请先配置一个可用模型"}
              rows={1}
              disabled={!model}
              aria-label="消息"
            />
            {streaming ? (
              <button type="button" className="send-button" onClick={() => abortRef.current?.abort()} aria-label="停止生成">
                <CircleStop size={19} />
              </button>
            ) : (
              <button type="submit" className="send-button" disabled={!draft.trim() || !model} aria-label="发送消息">
                <SendHorizontal size={19} />
              </button>
            )}
          </form>
          <div className="composer-note">模型可能会生成不准确的信息，请核对重要内容。</div>
        </div>
      </main>

      {settingsOpen && (
        <div className="dialog-backdrop" role="presentation" onMouseDown={() => setSettingsOpen(false)}>
          <section className="settings-dialog" role="dialog" aria-modal="true" aria-labelledby="settings-title" onMouseDown={(event) => event.stopPropagation()}>
            <div className="dialog-header">
              <div>
                <h2 id="settings-title">连接设置</h2>
                <p>Access Key 只保存在当前浏览器会话中。</p>
              </div>
              <button className="icon-button" onClick={() => setSettingsOpen(false)} aria-label="关闭设置"><X size={19} /></button>
            </div>
            <label className="field-label" htmlFor="api-key">Access Key</label>
            <input id="api-key" className="text-input" type="password" value={apiKey} onChange={(event) => saveApiKey(event.target.value)} autoComplete="off" placeholder="sk-…" />
            <div className="dialog-actions">
              <button className="secondary-button" onClick={() => void refreshModels()}><RefreshCw size={16} />测试连接</button>
              <button className="primary-button" onClick={() => { setSettingsOpen(false); void refreshModels(); }}>完成</button>
            </div>
          </section>
        </div>
      )}
    </div>
  );
}
