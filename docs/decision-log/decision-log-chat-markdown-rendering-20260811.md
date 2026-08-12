# Decision Log — Render Markdown di Bubble Chat Change Request

**Tanggal:** 2026-08-11
**Konteks:** chat-markdown-rendering

## Konteks/Masalah

Balasan Claude di chat Change Request (Vision §6) berformat Markdown (heading, list, bold, code block, tabel). Sebelumnya bubble assistant menampilkannya sebagai teks mentah (`{m.text}` + `white-space: pre-wrap`) — jadi `**tebal**`, `# Judul`, ```` ```code``` ```` kelihatan simbol mentahnya, tidak rapi. User minta bubble assistant dirender layaknya viewer `.md`.

## Keputusan

Render **hanya bubble assistant** sebagai HTML hasil parse Markdown. Bubble **user tetap plain text** (`{m.text}`) — input user tidak boleh diinterpretasi sebagai markup/HTML (surface XSS + bikin bingung).

**Tambah 2 runtime dependency pertama di frontend** (`dependencies` sebelumnya kosong):
- **`marked`** (^18) — parse Markdown → HTML. Dipilih karena ringan, cepat, de-facto standar, sync.
- **`dompurify`** (^3) — sanitasi HTML hasil parse. WAJIB karena `marked` meneruskan raw HTML apa adanya (markdown mengizinkan inline HTML), dan konten assistant bisa memuat/echo HTML berbahaya (mis. user minta Claude mengulang `<img onerror=...>`). Konsisten dengan kultur anti-XSS repo ini (lihat `lib/comments.ts` yang escape dulu sebelum render).
- **`jsdom`** (^29, devDependency) — supaya `renderMarkdown` (yang butuh DOM buat DOMPurify) bisa di-unit-test di vitest (env `node` default tidak punya DOM). Test file pakai docblock `// @vitest-environment jsdom`.

**Implementasi:** `lib/markdown.ts` → `renderMarkdown(md)` = `marked.parse` (gfm + breaks) lalu `DOMPurify.sanitize`. Sanitasi hanya jalan di browser (`typeof window !== 'undefined'`) — di server/prerender tidak pernah ada pesan chat, jadi aman & tidak butuh jsdom di bundle produksi. Hook DOMPurify menambah `target="_blank" rel="noopener noreferrer"` ke semua `<a>` (biar klik link tidak menendang user keluar SPA). Komponen: bubble assistant pakai `{@html renderMarkdown(m.text)}` dalam wrapper `.cr-md` (punya style heading/list/code/pre/tabel), bubble user tetap `{m.text}`.

## Alasan

- **Kenapa lib, bukan hand-rolled:** markdown lengkap (heading, list bertingkat, code fence, tabel GFM, link) aman & benar itu non-trivial; menulis sendiri rawan bug & XSS. `marked`+`DOMPurify` matang & kecil. Ini pengecualian wajar dari sikap "hemat dependency" repo — beda dari kasus chart (di mana kebutuhannya sederhana & custom SVG cukup); di sini parsing markdown yang benar memang layak pakai lib.
- **Kenapa sanitasi wajib:** `marked` v5+ menghapus opsi `sanitize` bawaan; raw HTML diteruskan. Tanpa DOMPurify, `<script>`/`onerror` dari konten bisa jalan. Di-test eksplisit (script & handler di-strip).
- **Kenapa user tetap plain text:** hanya assistant yang menghasilkan markdown; merender input user sebagai HTML = surface XSS tanpa manfaat.

## Dampak/File Terpengaruh

- `frontend/package.json` — tambah `marked`, `dompurify` (deps) + `jsdom` (devDep).
- `frontend/src/lib/markdown.ts` (baru) — `renderMarkdown`, browser-safe, link `target=_blank`.
- `frontend/src/lib/markdown.test.ts` (baru, `@vitest-environment jsdom`) — bold/heading/list/code + strip `<script>`/`onerror` + link target.
- `frontend/src/lib/components/ChangeRequestChat.svelte` — bubble assistant `{@html renderMarkdown()}` + style `.cr-md`.
- `CLAUDE.md` — catatan render markdown + deps baru.

## Catatan lanjutan

- Kalau nanti ada tempat lain yang perlu render markdown (mis. deskripsi Big Task), pakai `lib/markdown.ts` yang sama, jangan bikin parser baru.
- `npm audit` melaporkan sejumlah vuln transitif (mayoritas dari `jsdom`, devDep) — tidak masuk bundle produksi; tidak diprioritaskan sekarang.
