# Decision Log — Change Request: Dokumen Terpisah dari Transcript & Markdown Rendering

**Tanggal:** 2026-08-12
**Konteks:** change-request-document

## Konteks/Masalah

Di halaman Change Request, satu-satunya cara melihat isi usulan yang tersimpan adalah tombol "Lihat percakapan", yang menampilkan `raw_conversation` (seluruh transcript chat) sebagai `<pre>` plain text — markdown yang sudah dihasilkan `buildTranscript()` (heading `## User`/`## Assistant`) jadi kebaca mentah dengan karakter `#`/`*` literal, bukan diformat. User juga minta: begitu klik "Susun change request" dan Claude kasih respons dokumen `change_request.md`-nya, itu semestinya tersimpan TERPISAH dari transcript percakapan — jadi di list bisa lihat dua hal: history chat, DAN dokumen change request itu sendiri secara terpisah.

Ketemu saat riset: skema `change_requests` sudah punya kolom `structured_doc_path` (dari migration `0006` awal) yang PERSIS ditujukan untuk kebutuhan ini — tapi tidak pernah diisi frontend, selalu `NULL`. Niat awalnya (dokumentasi `06-db-design.md` lama) adalah path relatif ke file `change_request.md` di disk, seperti pola modul `upload`.

## Keputusan

**Rename `structured_doc_path` → `document_md` (migration `0021`), isinya markdown INLINE, bukan path file.** Menyimpan sebagai path file butuh modul file-serving baru (mirip `upload`) buat manfaat yang tidak sepadan — isi dokumennya kecil (hasil satu respons Claude), konsisten lebih simpel disimpan sebagai TEXT langsung di kolom yang sama seperti `raw_conversation`, tanpa infrastruktur tambahan.

**Deteksi "yang mana dokumennya" via flag di store, bukan re-derive dari transcript.** `chatSessionStore.ts` tidak punya cara struktural menandai "pesan assistant ini adalah hasil compile" — ditambahkan module-level flag `awaitingCompileTurn`, diset `true` tepat sebelum `compile()` kirim `COMPILE_PROMPT` (cuma kalau `sendPrompt()` beneran sukses kirim, bukan asal-asal — lihat return value baru `sendPrompt(): boolean`), lalu dikonsumsi di `onWsMessage` pas event `result` (turn selesai) — pesan assistant TERAKHIR di titik itu disalin ke state baru `compiledDocument`. Aman dari race condition karena `sendPrompt()` sendiri sudah nge-guard `busy` (gak bisa kirim prompt baru sebelum turn sebelumnya kelar), jadi `result` PERTAMA setelah `compile()` dipanggil pasti punya balasan buat compile itu, bukan turn lain.

**Dokumen = respons TERAKHIR Claude abis "Susun change request" doang, tanpa tahap edit manual** (opsi lain yang dipertimbangkan: kasih textarea buat user edit draft sebelum simpan — ditolak, scope-nya dibikin minimal dulu, bisa ditambah nanti kalau kebutuhannya muncul).

**`raw_conversation` DAN `document_md` DUA-DUANYA dikirim tiap "Simpan sebagai change request"** — bukan `document_md` menggantikan `raw_conversation`. Transcript penuh tetap penting buat konteks/audit percakapan; dokumen cuma ringkasan terstruktur hasil compile. `document_md` null kalau user belum pernah klik "Susun change request" sebelum save (list-page cukup nampilin transcript-nya doang di kasus itu, tombol "Lihat dokumen" gak muncul).

**Rendering: markdown via `renderMarkdown()`, BUKAN `<pre>` lagi**, untuk KEDUANYA (transcript maupun dokumen) — dipakai class `.cr-md` yang tadinya scoped di `ChangeRequestChat.svelte` (buat bubble chat), DIPINDAH jadi global di `app.css` supaya bisa dipakai bareng di `routes/change-requests/+page.svelte` juga (Svelte scoped style tidak nembus lintas komponen, class yang sama di komponen berbeda dapat hash scoping berbeda).

**List page dapat dua toggle independen per item**: "Lihat percakapan" (selalu ada) dan "Lihat dokumen" (cuma render kalau `cr.document_md` truthy) — bukan satu toggle gabungan/tab, biar user bisa buka salah satu atau keduanya sekaligus tanpa saling menutup.

## Alasan

- **Inline TEXT bukan file path**: lebih simpel, konsisten pola `raw_conversation` yang sudah TEXT juga, tanpa perlu modul serving file baru untuk konten yang kecil.
- **Flag di store, bukan re-derive**: tidak ada tag struktural di `ChatMsg`/SDK event yang bisa dipakai buat identifikasi "ini pesan hasil compile" — nge-tag lewat momen pengiriman prompt (yang KITA kontrol) lebih robust ketimbang heuristik nebak dari isi pesan.
- **Tanpa tahap edit manual dulu**: scope minimal — kalau nanti user pengin edit sebelum simpan, itu perubahan terpisah (textarea + state tambahan), tidak perlu dibangun sekarang kalau belum ada kebutuhan konkret.
- **Dua-duanya disimpan (bukan salah satu)**: `raw_conversation` = bukti/konteks lengkap gimana usulan itu muncul (penting buat triase SPV/SA yang mau ngerti latar belakangnya), `document_md` = ringkasan siap-baca. Keduanya punya kegunaan beda, bukan pengganti satu sama lain.
- **`.cr-md` jadi global CSS**: DRY — kalau tetap scoped per-komponen, styling markdown harus di-duplikasi persis di setiap tempat yang butuh render markdown, gampang divergen kalau salah satu diubah lupa yang lain.

## Dampak/File Terpengaruh

- `backend/db/migrations/0021_change_requests_document_md.up/down.sql` (baru) — rename kolom.
- `backend/internal/changerequest/handler.go` — struct `ChangeRequest`/`createRequest` field `DocumentMD` (json `document_md`), query `selectCR`/INSERT ikut disesuaikan.
- `frontend/src/lib/types.ts` — `ChangeRequest.document_md: string | null` (rename dari `structured_doc_path`).
- `frontend/src/lib/stores/chatSessionStore.ts` — state baru `compiledDocument`, flag `awaitingCompileTurn`, `sendPrompt()` sekarang balikin `boolean`.
- `frontend/src/lib/components/ChangeRequestChat.svelte` — `saveChangeRequest()` kirim `document_md`; indikator kecil "Dokumen change request siap disimpan"; blok CSS `.cr-md` dihapus dari sini (pindah ke `app.css`).
- `frontend/src/app.css` — `.cr-md` (markdown viewer) jadi global; gotcha flexbox `min-width:0`.
- `frontend/src/routes/change-requests/+page.svelte` — dua toggle "Lihat percakapan"/"Lihat dokumen", render via `renderMarkdown()` + `.cr-md` (ganti `<pre class="cr-transcript">`).
