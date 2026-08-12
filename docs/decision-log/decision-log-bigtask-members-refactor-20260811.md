# Decision Log — Big Task "Anggota" (bukan Reviewer), PIC Scoped, Clone-Review by Orang, Board Description

**Tanggal:** 2026-08-11
**Konteks:** bigtask-members-refactor

## Konteks/Masalah

Konsep "Reviewer" per Big Task (fitur `big_task_reviewers`, decision-log-bigtask-reviewer-assignment-20260810) ternyata salah model menurut kebutuhan tim. Yang sebenarnya dibutuhkan: daftar **siapa saja yang terlibat/menangani** sebuah Big Task (tim-nya) — komposisi role bebas (mis. 2 dev + 1 qa + 1 spv, atau 4 dev semua). "Reviewer" cuma salah satu peran situasional, bukan identitas keanggotaan. Selain itu:
- Saat tambah Daily Task, picker PIC menampilkan SEMUA user assignable — mestinya dibatasi ke anggota Big Task itu.
- Clone-review memaksa pilih role statis "SPV"/"QA" lalu comot user pertama dengan role itu — mestinya assign **orang spesifik** yang jadi reviewer.
- Review Queue digantung ke `big_task_reviewers` + fallback spv — mestinya berdasarkan siapa yang di-assign sebagai reviewer (PIC task hasil clone-review).
- Form "+ Board Baru" punya field `tag` yang tidak efektif — mestinya `nama board` + `deskripsi board`.

## Keputusan

1. **`big_task_reviewers` → rename jadi `big_task_members`** (tabel + index). Maknanya: anggota tim yang terlibat di Big Task. Field response `BigTask.reviewer_user_ids` → `member_user_ids`; request create `reviewer_user_ids` → `member_user_ids`.

2. **Big Task WAJIB minimal 2 anggota.** `Create` menolak 400 kalau `< 2`. Anggota **bisa diedit** setelah dibuat lewat endpoint baru `PUT /big-tasks/{id}/members` body `{member_user_ids}` (replace-set, tetap divalidasi min-2). Ini sekaligus jalur merapikan Big Task lama yang anggotanya masih 0/1 (data lama TIDAK diutak-atik otomatis — di-grandfather, diperbaiki manual lewat edit).

3. **PIC Daily Task wajib salah satu anggota Big Task.** `Create` dan `CloneReview` memvalidasi `pic_user_id ∈ members`, else 400. Daily task lama dengan PIC di luar anggota di-grandfather (validasi cuma di titik create baru). Frontend membatasi picker PIC ke anggota.

4. **Clone-review = assign ORANG, bukan role.** Request `{role_tag}` → `{reviewer_user_id, start_date, end_date}`. Reviewer wajib anggota Big Task (400 kalau bukan). Judul task jadi `"[Review <display_name>] <judul asal>"` (nama spesifik, bukan role — lebih jelas). Menyimpan **`daily_tasks.review_of_daily_task_id`** (kolom baru, FK nullable ke daily_tasks) = daily task asal yang direview.

5. **Review Queue di-rework.** Tidak lagi pakai `big_task_reviewers`/fallback spv. Sekarang: item = daily task yang merupakan **task review** (`review_of_daily_task_id IS NOT NULL`), yang **PIC-nya = requesting user**, dan belum ada di `item_reviews`. Response nambah kolom **daily task asal** yang direview (`source_daily_task_title`) + tetap ada big task & board. Otorisasi `mark-reviewed`: requesting user harus PIC dari review task itu.

6. **Board: `tag` → `description`.** Migration drop kolom `tag`, tambah `description TEXT NOT NULL DEFAULT ''`. Form create board: nama + deskripsi.

## Alasan

- **Anggota, bukan reviewer:** merefleksikan realita — Big Task dikeroyok tim campuran; "reviewer" adalah peran per-task (via clone-review), bukan atribut keanggotaan. Satu tabel keanggotaan + reviewer di-derive dari clone-review lebih akurat & tidak ambigu.
- **Min 2 anggota:** Big Task adalah unit kerja tim; satu orang = Daily Task saja. Enforce di create, tapi editable supaya fleksibel & bisa memperbaiki data lama.
- **Clone-review by orang + simpan source:** "[Review <nama>]" lebih jelas siapa yang bertanggung jawab; menyimpan `review_of_daily_task_id` memungkinkan Review Queue menampilkan konteks "mereview daily task apa" (diminta eksplisit).
- **Review Queue by PIC review task:** reviewer sekarang = PIC task review, jadi queue tinggal "task review yang di-assign ke saya & belum ditandai". Lebih lurus daripada join ke tabel keanggotaan.
- **Grandfather data lama:** hindari merusak data existing user (aturan main: jangan utak-atik data yang bukan dibuat sesi ini). Perbaikan lewat UI edit anggota.

## Dampak/File Terpengaruh

- Migrations `0015_rename_big_task_reviewers_to_members`, `0016_add_review_of_daily_task`, `0017_board_description`.
- `backend/internal/bigtask/handler.go` — `MemberUserIDs`, min-2 di Create, `SetMembers` (PUT), query rename tabel. Test min-2.
- `backend/internal/board/handler.go` — `Description` (drop `Tag`).
- `backend/internal/dailytask/handler.go` — validasi PIC∈members (Create+CloneReview), CloneReview by `reviewer_user_id` + `review_of_daily_task_id` + judul nama.
- `backend/internal/reviewqueue/handler.go` — list by PIC review task + `source_daily_task_title`; otorisasi mark-reviewed by PIC.
- `backend/cmd/api/main.go` — route `PUT /big-tasks/{bigTaskID}/members`.
- Frontend: `BigTaskList.svelte` (Anggota picker min-2 + summary + edit), `DailyTaskPanel.svelte` (PIC scoped + clone-review pilih orang), board create form (deskripsi), `routes/review-queue/+page.svelte` + `reviewQueueStore.ts` (kolom source), `lib/types.ts`.
- `CLAUDE.md` — sinkron: konsep anggota, PIC scoped, clone-review, review queue, board description; tandai `decision-log-bigtask-reviewer-assignment-20260810` SEBAGIAN DIGANTIKAN.

## Catatan

- Decision log lama `decision-log-bigtask-reviewer-assignment-20260810.md` (konsep "reviewer" + fallback spv di Review Queue) **digantikan** oleh keputusan ini untuk bagian model keanggotaan & sumber Review Queue.
