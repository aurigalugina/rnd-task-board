// Store sesi chat Change Request (Vision §6) — state SENGAJA di level modul
// (bukan di dalam komponen) supaya sesi TETAP HIDUP saat user pindah menu tanpa
// menutupnya. Komponen ChangeRequestChat cuma "view" yang subscribe ke sini;
// sesi baru benar-benar ditutup lewat closeSession() eksplisit (atau resetAll()
// saat logout). WebSocket + referensi sesi disimpan di scope modul ini juga,
// jadi tetap menerima event walau tidak ada komponen yang ter-mount.
import { writable } from 'svelte/store';
import { chatApi, openChatSocket, type ChatSession } from '$lib/chatClient';
import { parseSdkMessage, appendChatEvent, type ChatMsg } from '$lib/chatMessages';

export type ChatStep = 'idle' | 'setup' | 'chatting';

const COMPILE_PROMPT =
  'Tolong susun percakapan di atas menjadi dokumen change_request.md terstruktur ' +
  'dalam bahasa Indonesia, dengan bagian: Judul, Latar Belakang/Masalah, Usulan Perubahan, ' +
  'Dampak & Area Terpengaruh, dan Prioritas. Keluarkan sebagai markdown saja, tanpa basa-basi.';

// Prompt init OTOMATIS yang dikirim diam-diam saat sesi mulai (tidak tampil
// sebagai bubble user). Minta Claude baca CLAUDE.md dulu supaya paham konteks
// produk/arsitektur, lalu menyapa user & menanyakan usulannya.
const INTRO_PROMPT =
  'Kamu adalah asisten internal untuk fitur "Change Request" di portal R&D Ops. ' +
  'Tugasmu membantu anggota tim merumuskan usulan perubahan atau penambahan fitur. ' +
  'Langkah pertama SEBELUM apa pun: baca file CLAUDE.md di root direktori kerja ini ' +
  '(boleh juga lihat sekilas struktur folder / dokumen di docs/ kalau perlu) untuk memahami ' +
  'konteks produk, arsitektur, dan aturan proyek. ' +
  'Setelah paham, JANGAN menampilkan ringkasan panjang atau isi CLAUDE.md. ' +
  'Cukup sapa user dengan ramah dan singkat dalam Bahasa Indonesia, sebutkan bahwa kamu sudah ' +
  'memahami konteks proyeknya, lalu tanyakan perubahan atau fitur apa yang ingin mereka usulkan/diskusikan.';

export type ChatState = {
  step: ChatStep;
  cwd: string;
  configuredCwd: boolean;
  session: ChatSession | null;
  messages: ChatMsg[];
  busy: boolean; // turn sedang berjalan
  starting: boolean;
  costUsd: number;
  claudeSessionId: string | null;
  error: string | null;
  // Dokumen change_request.md hasil "Susun change request" -- diisi otomatis
  // begitu turn balasan compile() itu selesai (lihat awaitingCompileTurn di
  // bawah), TERPISAH dari messages/transcript. Disimpan ke document_md saat
  // "Simpan sebagai change request" (raw_conversation tetap seluruh chat).
  compiledDocument: string | null;
};

const initialState: ChatState = {
  step: 'idle',
  cwd: '',
  configuredCwd: false,
  session: null,
  messages: [],
  busy: false,
  starting: false,
  costUsd: 0,
  claudeSessionId: null,
  error: null,
  compiledDocument: null
};

const store = writable<ChatState>({ ...initialState });
export const chatSession = { subscribe: store.subscribe };

// Referensi non-reaktif di scope modul (tidak masuk store value).
let ws: WebSocket | null = null;
let intentionalClose = false;
let configLoaded = false;
// Ditandai true tepat sebelum compile() kirim COMPILE_PROMPT, dikonsumsi pas
// event 'result' turn itu selesai (lihat onWsMessage) -- aman dari race
// karena sendPrompt() sendiri sudah nge-guard "busy" (gak bisa kirim prompt
// baru sebelum turn sebelumnya kelar), jadi 'result' BERIKUTNYA abis compile()
// dipanggil PASTI punya balasan buat compile ini, bukan turn lain.
let awaitingCompileTurn = false;

function patch(p: Partial<ChatState>) {
  store.update((s) => ({ ...s, ...p }));
}

function snapshot(): ChatState {
  let s!: ChatState;
  store.subscribe((v) => (s = v))();
  return s;
}

// beginSetup dipanggil dari tombol "Ajukan usulan". Kalau sudah ada sesi hidup,
// tidak melakukan apa-apa (biar view langsung nampilin sesi yang berjalan).
export async function beginSetup() {
  const s = snapshot();
  if (s.step === 'chatting') return;
  patch({ step: 'setup', error: null });
  if (!configLoaded) {
    try {
      const cfg = await chatApi.config();
      configLoaded = true;
      patch({ cwd: cfg.default_cwd || '', configuredCwd: !!cfg.default_cwd });
    } catch (e) {
      patch({ error: (e as Error).message });
      return;
    }
  }
  // Kalau direktori sudah dari konfigurasi backend, langsung mulai sesi + init
  // (skip layar "Mulai percakapan"). Kalau belum di-set, tetap di layar setup
  // supaya user pilih direktori manual dulu.
  if (snapshot().configuredCwd) await startSession();
}

export function setCwd(cwd: string) {
  patch({ cwd });
}

export async function startSession() {
  const s = snapshot();
  if (!s.cwd) {
    patch({ error: 'Pilih direktori project dulu.' });
    return;
  }
  patch({ starting: true, error: null });
  try {
    const session = await chatApi.createSession({ cwd: s.cwd, permissionMode: 'plan' });
    intentionalClose = false;
    ws = openChatSocket(session.id);
    ws.addEventListener('message', onWsMessage);
    ws.addEventListener('close', onWsClose);
    ws.addEventListener('error', () => patch({ error: 'Koneksi WebSocket bermasalah.', busy: false }));
    ws.addEventListener('open', () => {
      // Init otomatis: kirim INTRO_PROMPT diam-diam (bukan bubble user) supaya
      // Claude baca CLAUDE.md dulu lalu menyapa user. Echo 'user' dari SDK
      // diabaikan parseSdkMessage, jadi prompt ini tidak tampil sebagai bubble.
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'prompt', text: INTRO_PROMPT }));
      }
    });
    patch({
      session,
      step: 'chatting',
      starting: false,
      busy: true, // langsung "mengetik" — init membaca CLAUDE.md lalu menyapa
      messages: [{ role: 'system', text: 'Menyiapkan konteks proyek dari CLAUDE.md…' }]
    });
  } catch (e) {
    patch({ error: (e as Error).message, starting: false });
    teardownSocket();
  }
}

function onWsMessage(event: MessageEvent) {
  let data: { type?: string; message?: unknown };
  try {
    data = JSON.parse(event.data);
  } catch {
    return;
  }
  if (data.type === 'error') {
    patch({ error: String((data as { message?: string }).message ?? 'error'), busy: false });
    return;
  }
  if (data.type !== 'sdk_message') return;

  for (const ev of parseSdkMessage(data.message)) {
    store.update((s) => {
      const messages = appendChatEvent(s.messages, ev);
      const next: Partial<ChatState> = { messages };
      if (ev.kind === 'result') {
        next.busy = false;
        if (typeof ev.costUsd === 'number') next.costUsd = ev.costUsd;
        if (ev.sessionId) next.claudeSessionId = ev.sessionId;
        if (awaitingCompileTurn) {
          awaitingCompileTurn = false;
          const lastAssistant = [...messages].reverse().find((m) => m.role === 'assistant');
          if (lastAssistant?.text) next.compiledDocument = lastAssistant.text;
        }
      }
      return { ...s, ...next };
    });
  }
}

function onWsClose() {
  if (intentionalClose) return;
  // Koneksi putus bukan karena kita (mis. service restart) — tandai, tapi
  // JANGAN hapus sesi/history biar user tetap bisa lihat & simpan transcript.
  patch({ busy: false, error: 'Koneksi chat terputus (service mati/restart?). History masih tersimpan.' });
}

export type PromptImage = { media_type: string; data: string }; // data = base64 mentah

// Balikin true kalau prompt beneran terkirim (dipakai compile() buat mastiin
// awaitingCompileTurn cuma diset kalau kirimnya sukses -- lihat komentar di
// deklarasi awaitingCompileTurn).
export function sendPrompt(text: string, images: PromptImage[] = []): boolean {
  const t = text.trim();
  const s = snapshot();
  // Boleh kirim kalau ada teks ATAU minimal satu gambar (screenshot tanpa teks).
  if (!ws || ws.readyState !== WebSocket.OPEN || s.busy || (!t && images.length === 0)) return false;
  const displayUrls = images.map((i) => `data:${i.media_type};base64,${i.data}`);
  patch({
    messages: [...s.messages, { role: 'user', text: t, images: displayUrls.length ? displayUrls : undefined }],
    busy: true
  });
  ws.send(JSON.stringify({ type: 'prompt', text: t, images: images.length ? images : undefined }));
  return true;
}

export function compile() {
  if (sendPrompt(COMPILE_PROMPT)) awaitingCompileTurn = true;
}

export function interrupt() {
  if (ws && ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ type: 'interrupt' }));
}

function teardownSocket() {
  if (ws) {
    intentionalClose = true;
    ws.removeEventListener('message', onWsMessage);
    ws.removeEventListener('close', onWsClose);
    ws.close();
    ws = null;
  }
}

// closeSession = penutupan EKSPLISIT oleh user ("Akhiri sesi"). Baru di sinilah
// WS ditutup + sesi di service di-DELETE + state di-reset ke idle.
export async function closeSession() {
  const s = snapshot();
  teardownSocket();
  if (s.session) chatApi.closeSession(s.session.id).catch(() => {});
  store.set({ ...initialState });
  // configLoaded HARUS ikut di-reset di sini (bukan cuma resetAll/logout) --
  // tanpa ini, beginSetup() abis "Akhiri sesi" skip fetch config lagi (karena
  // configLoaded masih true dari sesi sebelumnya) dan pakai cwd/configuredCwd
  // hasil reset (kosong/false) -- muncul salah kaprah "CHAT_DEFAULT_CWD belum
  // di-set di backend" padahal backend-nya beres, cuma cache frontend basi.
  configLoaded = false;
}

// cancelSetup: batal dari layar setup (belum ada sesi). Tidak menyentuh WS.
export function cancelSetup() {
  patch({ step: 'idle', error: null });
}

// resetAll dipanggil saat logout — pastikan sesi tidak menggantung. configLoaded
// sudah ikut di-reset di closeSession() sendiri, gak perlu diulang di sini.
export function resetAll() {
  closeSession();
}
