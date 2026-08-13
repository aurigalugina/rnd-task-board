-- Kolom `structured_doc_path` sudah ada dari migration 0006 tapi TIDAK PERNAH
-- dipakai (selalu NULL) -- niat awalnya path file di disk, tapi lebih simpel
-- simpan langsung isi markdown-nya inline (konsisten sama raw_conversation,
-- gak perlu modul file serving baru). Rename, bukan kolom baru terpisah.
-- Isinya = respons terakhir Claude setelah user klik "Susun change request"
-- (dokumen change_request.md), terpisah dari raw_conversation (transcript
-- percakapan penuh). Lihat decision-log-change-request-document-20260812.md.
ALTER TABLE change_requests RENAME COLUMN structured_doc_path TO document_md;
