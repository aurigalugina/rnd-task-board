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
| org_team | text | Afiliasi tim/organisasi, mis. `"R&D"` atau `"QA"` — dipakai untuk penanda visual "di luar tim inti" |
| theme_preference | text | Default `'retro-light'` |
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
| tag | text | Label deskriptif, mis. `"CBS Konvensional"` |
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
| is_done | boolean | Default false |
| blocker_text | text | Nullable |
| updated_at | timestamptz | |

Constraint: unique `(daily_task_id, entry_date)`. Indikator akhir pekan (`is_weekend`) **tidak disimpan** — dihitung dari `entry_date` di layer aplikasi/frontend.

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

### 3.12 `change_requests` (disiapkan, belum dipakai UI pada Fase 1)
| Kolom | Tipe | Keterangan |
|---|---|---|
| id | uuid | PK |
| submitted_by | uuid | FK → users.id |
| raw_conversation | text | Percakapan mentah sebelum disusun terstruktur |
| structured_doc_path | text | Nullable; path relatif ke `change_request.md` hasil penyusunan |
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
```

`idx_day_entries_entry_date` penting untuk kalkulasi rollup mingguan (FR-WKL-02/03) yang memfilter berdasarkan rentang tanggal lintas banyak Daily Task.

## 5. Contoh Query Kunci (Ilustratif)

### 5.1 `actual_pct` Daily Task
```sql
SELECT
  daily_task_id,
  ROUND(100.0 * COUNT(*) FILTER (WHERE is_done) / COUNT(*), 0) AS actual_pct
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
  SELECT daily_task_id, 100.0 * COUNT(*) FILTER (WHERE is_done) / COUNT(*) AS actual_pct
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
8. `0008_create_item_reviews.sql` — `item_reviews` (direncanakan, belum dibuat — lihat `docs/decision-log/decision-log-review-queue-schema-20260808.md`; dieksekusi saat Fase 7 roadmap pengembangan dikerjakan).
