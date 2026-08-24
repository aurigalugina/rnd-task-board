-- Baseline Awal Big Task (input persentase awal saat migrasi data ke
-- staging/production) -- lihat
-- decision-log-bigtask-baseline-progress-20260824.md.
-- Diwujudkan sebagai satu daily_tasks row khusus (bukan kolom override
-- terpisah) supaya tetap mengikuti prinsip computed-field (AVG progress_pct)
-- yang sudah dipakai di semua level agregasi.
ALTER TABLE daily_tasks ADD COLUMN is_baseline BOOLEAN NOT NULL DEFAULT false;

-- Maksimal satu baseline Daily Task per Big Task -- constraint ini yang bikin
-- edit ulang baseline jadi UPDATE (bukan INSERT baris baru numpuk).
CREATE UNIQUE INDEX idx_daily_tasks_one_baseline_per_bigtask
  ON daily_tasks (big_task_id) WHERE is_baseline;
