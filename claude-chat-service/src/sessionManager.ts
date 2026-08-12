import { randomUUID } from "node:crypto";
import { query, type Options, type PermissionMode, type Query, type SDKMessage, type SDKUserMessage } from "@anthropic-ai/claude-agent-sdk";
import { AsyncQueue } from "./lib/asyncQueue.js";
import type { ChatImage, CreateSessionRequest, SessionSummary } from "./types.js";

export type PumpError = { type: "pump_error"; message: string };
type Listener = (message: SDKMessage | PumpError) => void;

class ChatSession {
  readonly id = randomUUID();
  readonly cwd: string;
  model?: string;
  permissionMode: PermissionMode;
  claudeSessionId?: string;
  totalCostUsd = 0;
  status: "active" | "closed" = "active";
  readonly createdAt = new Date().toISOString();

  private readonly inputQueue = new AsyncQueue<SDKUserMessage>();
  private readonly listeners = new Set<Listener>();
  private readonly abortController = new AbortController();
  private readonly handle: Query;

  constructor(request: CreateSessionRequest) {
    this.cwd = request.cwd;
    this.model = request.model;
    this.permissionMode = request.permissionMode ?? "default";

    const options: Options = {
      cwd: request.cwd,
      model: request.model,
      permissionMode: this.permissionMode,
      allowDangerouslySkipPermissions: this.permissionMode === "bypassPermissions",
      includePartialMessages: true,
      persistSession: true,
      abortController: this.abortController,
      resume: request.resume,
    };

    this.handle = query({ prompt: this.inputQueue, options });
    void this.pump();
  }

  private async pump(): Promise<void> {
    try {
      for await (const message of this.handle) {
        this.observe(message);
        for (const listener of this.listeners) listener(message);
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      const errored: PumpError = { type: "pump_error", message };
      for (const listener of this.listeners) listener(errored);
    } finally {
      this.status = "closed";
    }
  }

  /** Update state turunan (claude_session_id, akumulasi cost) dari message yang lewat. */
  private observe(message: SDKMessage): void {
    if ("session_id" in message && typeof message.session_id === "string") {
      this.claudeSessionId = message.session_id;
    }
    if (message.type === "result" && "total_cost_usd" in message) {
      this.totalCostUsd = message.total_cost_usd;
    }
  }

  sendPrompt(text: string, images?: ChatImage[]): void {
    // Tanpa gambar: kirim string biasa (backward-compatible). Dengan gambar:
    // susun content block array (Anthropic format) — teks dulu, lalu tiap
    // image sebagai block base64. Claude multimodal, jadi screenshot kebaca.
    const content =
      images && images.length > 0
        ? [
            ...(text ? [{ type: "text" as const, text }] : []),
            ...images.map((img) => ({
              type: "image" as const,
              source: { type: "base64" as const, media_type: img.media_type, data: img.data },
            })),
          ]
        : text;

    this.inputQueue.push({
      type: "user",
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      message: { role: "user", content: content as any },
      parent_tool_use_id: null,
    });
  }

  async setPermissionMode(mode: PermissionMode): Promise<void> {
    await this.handle.setPermissionMode(mode);
    this.permissionMode = mode;
  }

  async setModel(model?: string): Promise<void> {
    await this.handle.setModel(model);
    this.model = model;
  }

  async interrupt(): Promise<void> {
    await this.handle.interrupt();
  }

  subscribe(listener: Listener): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  close(): void {
    this.inputQueue.close();
    this.abortController.abort();
    this.status = "closed";
  }

  toSummary(): SessionSummary {
    return {
      id: this.id,
      cwd: this.cwd,
      model: this.model,
      permissionMode: this.permissionMode,
      claudeSessionId: this.claudeSessionId,
      totalCostUsd: this.totalCostUsd,
      status: this.status,
      createdAt: this.createdAt,
    };
  }
}

const sessions = new Map<string, ChatSession>();

export function createSession(request: CreateSessionRequest): ChatSession {
  const session = new ChatSession(request);
  sessions.set(session.id, session);
  return session;
}

export function getSession(id: string): ChatSession | undefined {
  return sessions.get(id);
}

export function listSessions(): ChatSession[] {
  return [...sessions.values()];
}

export function closeSession(id: string): boolean {
  const session = sessions.get(id);
  if (!session) return false;
  session.close();
  sessions.delete(id);
  return true;
}

export type { ChatSession };
