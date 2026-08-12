import { describe, expect, it } from 'vitest';
import {
  parseSdkMessage,
  mergeAssistantText,
  appendChatEvent,
  buildTranscript,
  isSupportedImageType,
  type ChatEvent,
  type ChatMsg
} from './chatMessages';

describe('parseSdkMessage', () => {
  it('mengekstrak teks dari blok assistant', () => {
    const events = parseSdkMessage({
      type: 'assistant',
      message: { role: 'assistant', content: [{ type: 'text', text: 'Halo' }] }
    });
    expect(events).toEqual([{ kind: 'assistant_text', text: 'Halo' }]);
  });

  it('mengekstrak teks DAN tool_use dari satu pesan assistant, urut', () => {
    const events = parseSdkMessage({
      type: 'assistant',
      message: {
        role: 'assistant',
        content: [
          { type: 'text', text: 'Aku cek dulu' },
          { type: 'tool_use', name: 'Read', input: {} }
        ]
      }
    });
    expect(events).toEqual([
      { kind: 'assistant_text', text: 'Aku cek dulu' },
      { kind: 'tool_use', name: 'Read' }
    ]);
  });

  it('mengabaikan blok teks kosong', () => {
    const events = parseSdkMessage({
      type: 'assistant',
      message: { role: 'assistant', content: [{ type: 'text', text: '' }] }
    });
    expect(events).toEqual([]);
  });

  it('menerjemahkan result sukses (menyimpan session_id + cost)', () => {
    const events = parseSdkMessage({
      type: 'result',
      subtype: 'success',
      session_id: 'sess-123',
      total_cost_usd: 0.42
    });
    expect(events).toEqual([
      { kind: 'result', subtype: 'success', success: true, sessionId: 'sess-123', costUsd: 0.42 }
    ]);
  });

  it('result non-success ditandai success:false', () => {
    const [e] = parseSdkMessage({ type: 'result', subtype: 'interrupted', session_id: 's' });
    expect(e).toMatchObject({ kind: 'result', success: false, subtype: 'interrupted' });
  });

  it('system init menyimpan session_id, system lain diabaikan', () => {
    expect(parseSdkMessage({ type: 'system', subtype: 'init', session_id: 'abc' })).toEqual([
      { kind: 'system_init', sessionId: 'abc' }
    ]);
    expect(parseSdkMessage({ type: 'system', subtype: 'other' })).toEqual([]);
  });

  it('rate_limit_event -> rate_limit', () => {
    expect(parseSdkMessage({ type: 'rate_limit_event' })).toEqual([{ kind: 'rate_limit' }]);
  });

  it('tipe user (echo/tool_result) dan stream delta diabaikan', () => {
    expect(parseSdkMessage({ type: 'user', message: { role: 'user', content: [] } })).toEqual([]);
    expect(parseSdkMessage({ type: 'stream_event' })).toEqual([]);
  });

  it('input bukan objek diabaikan dengan aman', () => {
    expect(parseSdkMessage(null)).toEqual([]);
    expect(parseSdkMessage('halo')).toEqual([]);
    expect(parseSdkMessage(undefined)).toEqual([]);
  });
});

describe('mergeAssistantText', () => {
  it('menggabung hanya event assistant_text beruntun', () => {
    const events: ChatEvent[] = [
      { kind: 'assistant_text', text: 'Struktur ' },
      { kind: 'tool_use', name: 'Read' },
      { kind: 'assistant_text', text: 'folder backend...' }
    ];
    expect(mergeAssistantText(events)).toBe('Struktur folder backend...');
  });

  it('kosong kalau tidak ada teks assistant', () => {
    expect(mergeAssistantText([{ kind: 'rate_limit' }])).toBe('');
  });
});

describe('appendChatEvent', () => {
  it('assistant_text beruntun digabung ke satu bubble (immutable)', () => {
    let msgs: ChatMsg[] = [];
    msgs = appendChatEvent(msgs, { kind: 'assistant_text', text: 'Struktur ' });
    msgs = appendChatEvent(msgs, { kind: 'assistant_text', text: 'folder...' });
    expect(msgs).toEqual([{ role: 'assistant', text: 'Struktur folder...' }]);
  });

  it('tool_use memutus bubble — teks setelahnya jadi bubble baru', () => {
    let msgs: ChatMsg[] = [{ role: 'assistant', text: 'cek dulu' }];
    msgs = appendChatEvent(msgs, { kind: 'tool_use', name: 'Read' });
    msgs = appendChatEvent(msgs, { kind: 'assistant_text', text: 'hasilnya' });
    expect(msgs).toEqual([
      { role: 'assistant', text: 'cek dulu' },
      { role: 'tool', name: 'Read' },
      { role: 'assistant', text: 'hasilnya' }
    ]);
  });

  it('result success tidak menambah pesan; result gagal menambah baris sistem', () => {
    const base: ChatMsg[] = [{ role: 'user', text: 'hi' }];
    expect(appendChatEvent(base, { kind: 'result', success: true, subtype: 'success' })).toEqual(base);
    expect(appendChatEvent(base, { kind: 'result', success: false, subtype: 'interrupted' })).toEqual([
      { role: 'user', text: 'hi' },
      { role: 'system', text: 'Turn berakhir: interrupted' }
    ]);
  });

  it('rate_limit menambah baris sistem; system_init tidak mengubah pesan', () => {
    expect(appendChatEvent([], { kind: 'rate_limit' })).toEqual([
      { role: 'system', text: 'Menunggu rate limit Claude...' }
    ]);
    const base: ChatMsg[] = [{ role: 'user', text: 'x' }];
    expect(appendChatEvent(base, { kind: 'system_init', sessionId: 'a' })).toEqual(base);
  });

  it('tidak memutasi array input', () => {
    const base: ChatMsg[] = [{ role: 'assistant', text: 'a' }];
    const out = appendChatEvent(base, { kind: 'assistant_text', text: 'b' });
    expect(base).toEqual([{ role: 'assistant', text: 'a' }]);
    expect(out).toEqual([{ role: 'assistant', text: 'ab' }]);
  });
});

describe('buildTranscript', () => {
  it('menyusun markdown user/assistant, membuang baris sistem, menandai tool', () => {
    const msgs: ChatMsg[] = [
      { role: 'system', text: 'sesi dimulai' },
      { role: 'user', text: 'usulan X' },
      { role: 'tool', name: 'Grep' },
      { role: 'assistant', text: 'draft CR' }
    ];
    expect(buildTranscript(msgs)).toBe('## User\n\nusulan X\n\n_[tool: Grep]_\n\n## Assistant\n\ndraft CR');
  });

  it('kosong kalau cuma baris sistem', () => {
    expect(buildTranscript([{ role: 'system', text: 'x' }])).toBe('');
  });

  it('menandai jumlah gambar yang dilampirkan di pesan user', () => {
    const msgs: ChatMsg[] = [
      { role: 'user', text: 'lihat ini', images: ['data:image/png;base64,AAA', 'data:image/png;base64,BBB'] }
    ];
    expect(buildTranscript(msgs)).toBe('## User\n\nlihat ini\n\n_[2 gambar dilampirkan]_');
  });

  it('user tanpa images tidak dapat marker', () => {
    expect(buildTranscript([{ role: 'user', text: 'halo' }])).toBe('## User\n\nhalo');
  });
});

describe('isSupportedImageType', () => {
  it('menerima png/jpeg/gif/webp', () => {
    for (const t of ['image/png', 'image/jpeg', 'image/gif', 'image/webp']) {
      expect(isSupportedImageType(t)).toBe(true);
    }
  });
  it('menolak tipe lain', () => {
    expect(isSupportedImageType('image/svg+xml')).toBe(false);
    expect(isSupportedImageType('application/pdf')).toBe(false);
    expect(isSupportedImageType('')).toBe(false);
  });
});
