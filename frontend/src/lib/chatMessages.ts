// Parsing event mentah dari claude-chat-service (SDK message @anthropic-ai/
// claude-agent-sdk) jadi bentuk ternormalisasi buat render chat. Diekstrak
// sebagai fungsi murni supaya bisa ditest tanpa WebSocket / render komponen —
// pola sama seperti lib/dashboardStats.ts, lib/comments.ts.
// Referensi bentuk pesan: claude-chat-service/docs/api_contract.md.

export type ChatEvent =
  | { kind: 'assistant_text'; text: string }
  | { kind: 'tool_use'; name: string }
  | { kind: 'result'; success: boolean; subtype: string; sessionId?: string; costUsd?: number }
  | { kind: 'system_init'; sessionId?: string }
  | { kind: 'rate_limit' };

type ContentBlock = { type?: string; text?: string; name?: string };

function contentBlocks(msg: Record<string, unknown>): ContentBlock[] {
  const inner = msg.message as { content?: unknown } | undefined;
  if (!inner || !Array.isArray(inner.content)) return [];
  return inner.content as ContentBlock[];
}

// parseSdkMessage menerjemahkan satu SDKMessage jadi 0..n ChatEvent. Satu pesan
// assistant bisa berisi beberapa blok (teks + beberapa tool_use), makanya balikin
// array. Tipe yang tidak relevan buat UI (user echo, stream delta) -> [].
export function parseSdkMessage(raw: unknown): ChatEvent[] {
  if (!raw || typeof raw !== 'object') return [];
  const msg = raw as Record<string, unknown>;

  switch (msg.type) {
    case 'assistant': {
      const events: ChatEvent[] = [];
      for (const block of contentBlocks(msg)) {
        if (block.type === 'text' && block.text) {
          events.push({ kind: 'assistant_text', text: block.text });
        } else if (block.type === 'tool_use' && block.name) {
          events.push({ kind: 'tool_use', name: block.name });
        }
      }
      return events;
    }
    case 'result':
      return [
        {
          kind: 'result',
          subtype: typeof msg.subtype === 'string' ? msg.subtype : 'unknown',
          success: msg.subtype === 'success',
          sessionId: typeof msg.session_id === 'string' ? msg.session_id : undefined,
          costUsd: typeof msg.total_cost_usd === 'number' ? msg.total_cost_usd : undefined
        }
      ];
    case 'system':
      if (msg.subtype === 'init') {
        return [{ kind: 'system_init', sessionId: typeof msg.session_id === 'string' ? msg.session_id : undefined }];
      }
      return [];
    case 'rate_limit_event':
      return [{ kind: 'rate_limit' }];
    default:
      return [];
  }
}

// mergeAssistantText menggabung teks assistant beruntun jadi satu blok — turn
// bisa mengirim banyak event assistant_text kecil (streaming), kita mau
// menampilkannya sebagai satu bubble, bukan puluhan potongan.
export function mergeAssistantText(events: ChatEvent[]): string {
  return events
    .filter((e): e is Extract<ChatEvent, { kind: 'assistant_text' }> => e.kind === 'assistant_text')
    .map((e) => e.text)
    .join('');
}

// ChatMsg adalah bentuk pesan yang dirender di panel chat (hasil reduksi dari
// ChatEvent). Dipakai bareng komponen + store — makanya di sini, bukan inline.
export type ChatMsg =
  | { role: 'user'; text: string; images?: string[] } // images = data URL buat render
  | { role: 'assistant'; text: string }
  | { role: 'tool'; name: string }
  | { role: 'system'; text: string };

// Tipe gambar yang boleh dilampirkan (mengikuti yang didukung Claude vision).
export const SUPPORTED_IMAGE_TYPES = ['image/png', 'image/jpeg', 'image/gif', 'image/webp'];

export function isSupportedImageType(mime: string): boolean {
  return SUPPORTED_IMAGE_TYPES.includes(mime);
}

// appendChatEvent menerapkan satu ChatEvent ke daftar pesan secara IMMUTABLE.
// Teks assistant beruntun digabung ke bubble terakhir (streaming jadi satu
// balon, bukan puluhan potongan). Efek samping non-pesan (busy/cost/sessionId
// dari event result) ditangani caller, bukan di sini. Pure & testable.
export function appendChatEvent(messages: ChatMsg[], ev: ChatEvent): ChatMsg[] {
  switch (ev.kind) {
    case 'assistant_text': {
      const last = messages[messages.length - 1];
      if (last && last.role === 'assistant') {
        return [...messages.slice(0, -1), { role: 'assistant', text: last.text + ev.text }];
      }
      return [...messages, { role: 'assistant', text: ev.text }];
    }
    case 'tool_use':
      return [...messages, { role: 'tool', name: ev.name }];
    case 'rate_limit':
      return [...messages, { role: 'system', text: 'Menunggu rate limit Claude...' }];
    case 'result':
      return ev.success ? messages : [...messages, { role: 'system', text: `Turn berakhir: ${ev.subtype}` }];
    case 'system_init':
      return messages;
  }
}

// buildTranscript menyusun percakapan jadi teks markdown buat disimpan ke
// change_requests.raw_conversation. Baris sistem (status UI) dibuang.
export function buildTranscript(messages: ChatMsg[]): string {
  return messages
    .filter((m) => m.role !== 'system')
    .map((m) => {
      if (m.role === 'user') {
        const marker = m.images && m.images.length > 0 ? `\n\n_[${m.images.length} gambar dilampirkan]_` : '';
        return `## User\n\n${m.text}${marker}`;
      }
      if (m.role === 'assistant') return `## Assistant\n\n${m.text}`;
      return `_[tool: ${m.name}]_`;
    })
    .join('\n\n');
}
