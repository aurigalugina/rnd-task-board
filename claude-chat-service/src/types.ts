import type { PermissionMode } from "@anthropic-ai/claude-agent-sdk";

export type CreateSessionRequest = {
  cwd: string;
  model?: string;
  permissionMode?: PermissionMode;
  /** Resume sesi Claude Code sebelumnya (claude_session_id, bukan session_id internal kita). */
  resume?: string;
};

export type SessionSummary = {
  id: string;
  cwd: string;
  model?: string;
  permissionMode: PermissionMode;
  claudeSessionId?: string;
  totalCostUsd: number;
  status: "active" | "closed";
  createdAt: string;
};

/** Gambar terlampir pada prompt (mis. screenshot). `data` = base64 MENTAH
 *  tanpa prefix `data:...;base64,`. */
export type ChatImage = { media_type: string; data: string };

export type ClientToServerMessage =
  | { type: "prompt"; text: string; images?: ChatImage[] }
  | { type: "set_mode"; mode: PermissionMode }
  | { type: "set_model"; model?: string }
  | { type: "interrupt" };

export type ServerToClientMessage =
  | { type: "sdk_message"; message: unknown }
  | { type: "session_updated"; session: SessionSummary }
  | { type: "error"; message: string };
