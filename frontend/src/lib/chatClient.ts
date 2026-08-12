// Klien untuk claude-chat-service via backend proxy (/api/v1/chat/*). REST lewat
// `api` biasa (bawa Bearer header); WebSocket butuh URL khusus dengan
// ?access_token= karena browser WS tak bisa set header Authorization. Lihat
// docs/decision-log/decision-log-change-request-integration-20260811.md.
import { api, getAccessToken } from './api/client';

export type PermissionMode = 'default' | 'acceptEdits' | 'bypassPermissions' | 'plan' | 'dontAsk' | 'auto';

export type ChatSession = {
  id: string;
  cwd: string;
  model?: string;
  permissionMode: PermissionMode;
  claudeSessionId?: string;
  totalCostUsd: number;
  status: 'active' | 'closed';
  createdAt: string;
};

export type BrowseResult = {
  path: string;
  parent: string;
  directories: { name: string; path: string }[];
};

export const chatApi = {
  config: () => api.get<{ default_cwd: string }>('/chat/local/config'),
  browse: (path?: string) =>
    api.get<BrowseResult>(`/chat/fs/browse${path ? `?path=${encodeURIComponent(path)}` : ''}`),
  createSession: (body: { cwd: string; permissionMode?: PermissionMode; model?: string; resume?: string }) =>
    api.post<ChatSession>('/chat/sessions', body),
  closeSession: (id: string) => api.del(`/chat/sessions/${id}`)
};

// buildChatWsUrl menyusun URL WebSocket ke chat proxy. Diekstrak & diekspor
// supaya bisa ditest (bukan inline di komponen). `origin` = window.location.origin
// (mis. "http://localhost:5173") -> "ws://..."; https -> wss.
export function buildChatWsUrl(sessionId: string, origin: string, token: string | null): string {
  const base = origin.replace(/^http/, 'ws');
  const q = token ? `?access_token=${encodeURIComponent(token)}` : '';
  return `${base}/api/v1/chat/ws/sessions/${sessionId}${q}`;
}

// openChatSocket membuka WebSocket ke session, memakai token in-memory saat ini.
export function openChatSocket(sessionId: string): WebSocket {
  const url = buildChatWsUrl(sessionId, window.location.origin, getAccessToken());
  return new WebSocket(url);
}
