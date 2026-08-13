# Desain Basis Data — R&D Ops

**Status:** Draft v1
**Referensi:** `03-srs.md`, `04-architecture.md`
**Engine:** PostgreSQL 16+

---

## 1. Prinsip Desain

1. Nilai turunan yang bergantung pada waktu berjalan (`expected_pct`, `verdict`, `completion_rate`) **tidak disimpan** sebagai kolom fisik — dihitung di layer aplikasi saat query. Hanya nilai sumber (tanggal, status boolean) yang disimpan.
2. Role bersifat many-to-many (lihat `04-architecture.md` §6 dan BRD RULE-11) — tidak ada kolom `role` tunggal di tabel `users`.
3. Seluruh tabel menggunakan `id UUID` sebagai primary key (bukan auto-increment integer), agar aman digenerate di sisi aplikasi sebelum insert dan tidak membocorkan volume data.
4. Seluruh tabel menyertakan `created_at` dan `updated_at` (timestamptz).

## 2. Entity Relationship (Ringkas)

```
users ─┬─< user_roles >─┬─ roles
       │
       ├─< daily_tasks.pic_user_id
       ├─< comments.author_id
       ├─< cheat_sheet_items.author_id
       ├─< big_task_signoffs.signed_by
       ├─< weekly_push_log.pushed_by
       └─< item_reviews.reviewed_by

boards ─< big_tasks ─┬─< daily_tasks ─< day_entries
                     ├─< big_task_signoffs (0..1 aktif)
                     └─< comments (scope: big_task atau daily_task)

boards ─< cheat_sheet_items

big_tasks ─< weekly_push_log
```

## 3. Skema Tabel

### 3.1 `users`
| Kolom | Tipe | Keterangan |
|---|---|---|
| id | uuid | PK |
| display_name | text | Nama tampilan |
| initials | varchar(2) | Inisial avatar |
| email | text | Unique, dipakai untuk login |
| password_hash | text | bcrypt/argon2 |
| org_team | text | Afiliasi tim/organisasi, mis. `"R&D"` atau `"QA"` — dipakai untuk penanda visual "di luar tim inti". TETAP kolom text apa adanya (bukan FK), divalidasi di layer aplikasi harus ada di `referensi_tim.name` (§3.14) — lihat `docs/decision-log/decision-log-hr-mapping-super-user-20260810.md` |
| theme_preference | text | Default `'retro-light'` |
| access_level | text | Ditambah migration `0014`. `CHECK IN ('super_user','regular_user')`, default `'regular_user'`. **SENGAJA kolom tunggal** (bukan lewat `roles`/`user_roles` many-to-many seperti biasa) — super_user/regular_user saling eksklusif per user, beda konsep dari `roles` (spv/dev/qa dst) yang bisa dirangkap. Keputusan sadar user, lihat decision log |
| hr_user_id | integer, nullable | Ditambah migration `0014`. `UNIQUE REFERENCES referensi_user_hr(hr_user_id)` (§3.15) — mapping ke pegawai sistem HR asli, dipakai `weeklyplan.pushToMyAgenda` menggantikan placeholder CRC32 begitu di-set |
| created_at | timestamptz | |
| updated_at | timestamptz | |

### 3.2 `roles`
| Kolom | Tipe | Keterangan |
|---|---|---|
| id | uuid | PK |
| code | text | Unique, mis. `spv`, `sa`, `dev`, `qa`, `admin` |
| label | text | Label tampilan, mis. `"System Analyst"` |

### 3.3 `user_roles`
| Kolom | Tipe | Keterangan |
|---|---|---|
| user_id | uuid | FK → users.id |
| role_id | uuid | FK → roles.id |
| | | PK komposit (user_id, role_id) |

### 3.4 `boards`
| Kolom | Tipe | Keterangan |
|---|---|---|
| id | uuid | PK |
| name | text | |
| description | text | Nama board + deskripsi (gantiin `tag` lama, migration `0017`) |
| archived_at | timestamptz | NULL. Keberadaan = board diarsipkan (existence-pattern, migration `0020`). Board archived hilang dari `GET /boards`, TETAP muncul di Weekly Plan/Review Queue — lihat `decision-log-board-archive-20260812.md` |
| archived_by | uuid | NULL. FK → users.id, audit trail siapa yang mengarsipkan |
| created_at | timestamptz | |
| updated_at | timestamptz | |

### 3.5 `big_tasks`
| Kolom | Tipe | Keterangan |
|---|---|---|
| id | uuid | PK |
| board_id | uuid | FK → boards.id |
| name | text | |
| start_date | date | |
| deadline | date | |
| default_pic_user_id | uuid | FK → users.id, nullable |
| on_hold | boolean | Default false |
| created_at | timestamptz | |
| updated_at | timestamptz | |

Catatan: `actual_pct` dan `expected_pct` **tidak** menjadi kolom di tabel ini (lihat Prinsip Desain #1). Dihitung dari agregasi `daily_tasks`/`day_entries` dan dari `start_date`/`deadline` vs waktu berjalan.

### 3.6 `big_task_signoffs`
| Kolom | Tipe | Keterangan |
|---|---|---|
| id | uuid | PK |
| big_task_id | uuid | FK → big_tasks.id, unique (satu Big Task hanya punya satu sign-off aktif) |
| signed_by | uuid | FK → users.id |
| signed_at | timestamptz | |

Keberadaan baris pada tabel ini = Big Task berstatus signed. Undo sign-off = hapus baris. Verdict `win`/`lose` ditentukan aplikasi dari kombinasi keberadaan baris ini, `signed_at` vs `deadline`, dan status `actual_pct`.

### 3.7 `daily_tasks`
| Kolom | Tipe | Keterangan |
|---|---|---|
| id | uuid | PK |
| big_task_id | uuid | FK → big_tasks.id |
| title | text | |
| pic_user_id | uuid | FK → users.id |
| start_date | date | |
| end_date | date | |
| created_at | timestamptz | |
| updated_at | timestamptz | |

### 3.8 `day_entries`
| Kolom | Tipe | Keterangan |
|---|---|---|
| id | uuid | PK |
| daily_task_id | uuid | FK → daily_tasks.id |
| entry_date | date | |
| planned_text | text | Nullable/kosong di awal |
| progress_pct | smallint | `CHECK (0-100)`, default 0. Menggantikan `is_done` boolean (migration `0011`) — `0`=Belum, `1-99`=On Progress, `100`=Selesai, turunan murni di UI, bukan 3 kolom terpisah. Lihat `docs/decision-log/decision-log-day-entry-progress-pct-20260810.md` |
| blocker_text | text | Nullable |
| created_at | timestamptz | Ditambah migration `0010` — dasar `ORDER BY entry_date, created_at` (bukan `updated_at`, biar urutan gak geser tiap baris diedit) |
| updated_at | timestamptz | |

**TIDAK ADA LAGI constraint unique `(daily_task_id, entry_date)`** (dicabut migration `0010_day_entries_allow_multiple_per_date` — lihat `docs/decision-log/decision-log-day-entry-add-delete-20260810.md`). Satu Daily Task boleh punya lebih dari satu baris di tanggal yang sama (breakdown lebih dari satu task per hari), dan boleh nol (semua baris di tanggal itu dihapus, mis. weekend yang PIC-nya gak mau lembur) — `actual_pct` (§5.1) otomatis menyesuaikan karena dihitung dari SEMUA baris yang ADA saat dibaca, bukan disimpan terpisah. Indikator akhir pekan (`is_weekend`) **tidak disimpan** — dihitung dari `entry_date` di layer aplikasi/frontend.

### 3.9 `comments`
| Kolom | Tipe | Keterangan |
|---|---|---|
| id | uuid | PK |
| big_task_id | uuid | FK → big_tasks.id |
| daily_task_id | uuid | FK → daily_tasks.id, **nullable** (null = scope umum/level Big Task) |
| author_id | uuid | FK → users.id |
| body | text | Boleh berisi token `@nama` mentah; parsing mention dilakukan di frontend saat render |
| created_at | timestamptz | |

### 3.10 `cheat_sheet_items`
| Kolom | Tipe | Keterangan |
|---|---|---|
| id | uuid | PK |
| board_id | uuid | FK → boards.id |
| type | text | Enum aplikatif: `file` \| `url` \| `note` |
| title | text | |
| value | text | Isi sesuai tipe: nama berkas (referensi storage), URL, atau teks catatan |
| author_id | uuid | FK → users.id |
| created_at | timestamptz | |

### 3.11 `weekly_push_log`
| Kolom | Tipe | Keterangan |
|---|---|---|
| id | uuid | PK |
| big_task_id | uuid | FK → big_tasks.id |
| week_start | date | Senin sebagai awal minggu |
| callback_id | text | Unique, digenerate sekali saat push pertama |
| pushed_by | uuid | FK → users.id |
| pushed_at | timestamptz | Diperbarui setiap push berikutnya (upsert) |
| last_payload_actual_pct | numeric(5,2) | Snapshot nilai saat push terakhir, untuk keperluan audit |
| last_payload_expected_pct | numeric(5,2) | Snapshot nilai saat push terakhir |

Constraint: unique `(big_task_id, week_start)` — inilah kunci upsert yang dirujuk pada FR-WKL-05.

### 3.12 `change_requests`
| Kolom | Tipe | Keterangan |
|---|---|---|
| id | uuid | PK |
| submitted_by | uuid | FK → users.id |
| raw_conversation | text | Transcript percakapan penuh (markdown, dari `buildTranscript()`) |
| document_md | text | Nullable; markdown INLINE dokumen `change_request.md` hasil "Susun change request" (rename dari `structured_doc_path` migration `0021` — niat awal path file di disk tidak jadi dipakai, lihat decision-log-change-request-document-20260812.md) |
| status | text | Enum aplikatif: `pending` \| `approved` \| `rejected` \| `scheduled` |
| reviewed_by | uuid | FK → users.id, nullable |
| reviewed_at | timestamptz | Nullable |
| created_at | timestamptz | |

### 3.13 `item_reviews`

Menyimpan status "sudah ditinjau" untuk kebutuhan Review Queue (FR-NTF-01/02/03). Ditambahkan belakangan — lihat `docs/decision-log/decision-log-review-queue-schema-20260808.md` untuk alasan lengkap. Pola sama seperti `big_task_signoffs`: keberadaan baris = sudah ditinjau.

| Kolom | Tipe | Keterangan |
|---|---|---|
| id | uuid | PK |
| item_type | text | Polymorphic, mis. `daily_task` — nilai lain menyusul kalau scope review meluas |
| item_id | uuid | Polymorphic, tidak berupa FK (tidak menunjuk satu tabel spesifik); validasi keberadaan dilakukan di layer aplikasi |
| reviewed_by | uuid | FK → users.id |
| reviewed_at | timestamptz | |

Constraint: unique `(item_type, item_id)` — `mark-reviewed` bersifat idempoten (upsert `reviewed_by`/`reviewed_at`, bukan insert baris baru).

### 3.14 `big_task_reviewers`

Reviewer yang di-assign eksplisit ke satu Big Task (orang spesifik, bisa lebih dari satu) — dasar filter otorisasi Review Queue per user (§9 API contract). Ditambahkan Fase pasca-8 — lihat `docs/decision-log/decision-log-bigtask-reviewer-assignment-20260810.md`. Menggantikan sebagian `decision-log-review-queue-scope-20260809.md` (yang tadinya hardcode role spv).

| Kolom | Tipe | Keterangan |
|---|---|---|
| big_task_id | uuid | FK → big_tasks.id, ON DELETE CASCADE |
| user_id | uuid | FK → users.id, ON DELETE CASCADE |

PK komposit `(big_task_id, user_id)` — satu baris per pasangan, insert ulang otomatis idempoten lewat `ON CONFLICT DO NOTHING` kalau dibutuhkan (saat ini di-insert sekali saat create Big Task, belum ada endpoint update terpisah). Big Task TANPA baris di tabel ini = "belum di-assign reviewer" → Review Queue fallback ke role `spv` (lihat `reviewqueue.List`).

### 3.15 `referensi_tim`

Daftar nama tim/org — sumber dropdown "Tim/Org" di form user (`users.org_team`), gantiin free-text. Ditambahkan migration `0012`, lihat `docs/decision-log/decision-log-hr-mapping-super-user-20260810.md`.

| Kolom | Tipe | Keterangan |
|---|---|---|
| id | uuid | PK |
| name | text | Unique, mis. `"R&D"` |
| created_at | timestamptz | |

Bukan CRUD penuh — cuma `GET`/`POST` (tambah nama tim baru), belum ada update/delete (belum ada kebutuhannya).

### 3.16 `referensi_user_hr`

Data pegawai sistem HR (MyAgenda) asli — dasar mapping `users.hr_user_id`. Di-seed migration `0013` PERSIS dari export yang diberikan (~78 baris) — data ini MILIK sistem HR eksternal, TIDAK ADA CRUD UI (update resmi lewat migration baru kalau daftar pegawai HR berubah). Lihat decision log di atas.

| Kolom | Tipe | Keterangan |
|---|---|---|
| hr_user_id | integer | PK, SAMA PERSIS dengan `user_id` di export HR (bukan auto-increment kita) |
| email | text | |
| nip | text, nullable | Beberapa baris export aslinya `"-"`, di-seed sebagai `NULL` |
| nama_lengkap | text | |

## 4. Indeks yang Direkomendasikan

```sql
CREATE INDEX idx_big_tasks_board_id ON big_tasks(board_id);
CREATE INDEX idx_daily_tasks_big_task_id ON daily_tasks(big_task_id);
CREATE INDEX idx_day_entries_daily_task_id ON day_entries(daily_task_id);
CREATE INDEX idx_day_entries_entry_date ON day_entries(entry_date);
CREATE INDEX idx_comments_big_task_id ON comments(big_task_id);
CREATE INDEX idx_comments_daily_task_id ON comments(daily_task_id);
CREATE INDEX idx_cheat_sheet_items_board_id ON cheat_sheet_items(board_id);
CREATE INDEX idx_weekly_push_log_lookup ON weekly_push_log(big_task_id, week_start);
CREATE INDEX idx_item_reviews_lookup ON item_reviews(item_type, item_id);
CREATE INDEX idx_big_task_reviewers_user ON big_task_reviewers(user_id);
```

`idx_day_entries_entry_date` penting untuk kalkulasi rollup mingguan (FR-WKL-02/03) yang memfilter berdasarkan rentang tanggal lintas banyak Daily Task.

## 5. Contoh Query Kunci (Ilustratif)

### 5.1 `actual_pct` Daily Task
Rata-rata `progress_pct` (0-100) semua `day_entries`-nya — BUKAN lagi persentase hari yang "full selesai" (lihat `decision-log-day-entry-progress-pct-20260810.md`, `progress_pct` menggantikan `is_done` boolean per migration `0011`).
```sql
SELECT
  daily_task_id,
  ROUND(AVG(progress_pct), 0) AS actual_pct
FROM day_entries
WHERE daily_task_id = $1
GROUP BY daily_task_id;
```

### 5.2 `actual_pct` Big Task (rata-rata Daily Task di bawahnya)
```sql
SELECT
  dt.big_task_id,
  ROUND(AVG(sub.actual_pct), 0) AS actual_pct
FROM daily_tasks dt
JOIN (
  SELECT daily_task_id, AVG(progress_pct) AS actual_pct
  FROM day_entries GROUP BY daily_task_id
) sub ON sub.daily_task_id = dt.id
WHERE dt.big_task_id = $1
GROUP BY dt.big_task_id;
```

### 5.3 Rollup mingguan per Big Task (dasar FR-WKL-02/03)
```sql
SELECT
  dt.big_task_id,
  COUNT(*) FILTER (WHERE de.entry_date BETWEEN $1 AND $2) AS total_week_days,
  COUNT(*) FILTER (WHERE de.entry_date BETWEEN $1 AND $2 AND de.is_done) AS done_week_days,
  COUNT(*) FILTER (WHERE de.entry_date BETWEEN $1 AND $2 AND de.entry_date <= CURRENT_DATE) AS elapsed_week_days
FROM daily_tasks dt
JOIN day_entries de ON de.daily_task_id = dt.id
GROUP BY dt.big_task_id;
```

## 6. Migration Awal (Urutan)

1. `0001_create_users_roles.sql` — `users`, `roles`, `user_roles`, seed 5 role dasar (`spv`, `sa`, `dev`, `qa`, `admin`).
2. `0002_create_boards_bigtasks.sql` — `boards`, `big_tasks`, `big_task_signoffs`.
3. `0003_create_daily_tasks.sql` — `daily_tasks`, `day_entries`.
4. `0004_create_collaboration.sql` — `comments`, `cheat_sheet_items`.
5. `0005_create_weekly_push_log.sql` — `weekly_push_log`.
6. `0006_create_change_requests.sql` — `change_requests` (skema disiapkan, tidak dipakai UI Fase 1).
7. `0007_indexes.sql` — seluruh indeks pada §4.
8. `0008_create_item_reviews.sql` — `item_reviews` (dieksekusi Fase 7, lihat `docs/decision-log/decision-log-review-queue-schema-20260808.md`).
9. `0009_create_big_task_reviewers.sql` — `big_task_reviewers` (§3.14), lihat `docs/decision-log/decision-log-bigtask-reviewer-assignment-20260810.md`.
10. `0010_day_entries_allow_multiple_per_date.sql` — cabut unique `(daily_task_id, entry_date)` di `day_entries`, tambah kolom `created_at`, lihat `docs/decision-log/decision-log-day-entry-add-delete-20260810.md`.
11. `0011_day_entries_progress_pct.sql` — ganti `day_entries.is_done` (boolean) jadi `progress_pct` (smallint 0-100), lihat `docs/decision-log/decision-log-day-entry-progress-pct-20260810.md`.
12. `0012_create_referensi_tim.sql` — `referensi_tim` (§3.15), seed `'R&D'`.
13. `0013_create_referensi_user_hr.sql` — `referensi_user_hr` (§3.16), seed ~78 baris dari export HR.
14. `0014_add_users_hr_access_level.sql` — tambah `users.access_level` dan `users.hr_user_id`, lihat `docs/decision-log/decision-log-hr-mapping-super-user-20260810.md`.
