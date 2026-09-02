// Bentuk data mengikuti docs/05-api-contract.md — field turunan (actual_pct,
// expected_pct, verdict, completion_rate, is_weekend) selalu dari server,
// jangan pernah dihitung ulang di frontend.

export type BoardCategory = 'project' | 'routine';

export type Board = {
  id: string;
  name: string;
  description: string;
  // Nullable -- board lama "belum dikategorikan" sampai di-edit super_user.
  // team_ids: assignment many-to-many ke referensi_tim, lihat
  // decision-log-boards-dashboard-enhancements-20260820.md.
  category: BoardCategory | null;
  team_ids: string[];
};

// GET /boards/archive (super_user only) — lihat decision-log-board-archive-20260812.md.
export type ArchivedBoard = {
  id: string;
  name: string;
  description: string;
  archived_at: string;
  archived_by_name: string;
};

export type BoardSummary = {
  board_id: string;
  total_big_tasks: number;
  not_started: number;
  running: number;
  done: number;
  on_hold: number;
  won: number;
  lost: number;
  completion_rate: number;
  project_status: 'in_progress' | 'done';
};

export type Verdict = 'on_progress' | 'win' | 'lose';

export type Severity = 'critical' | 'high' | 'medium' | 'low';

export type BigTask = {
  id: string;
  board_id: string;
  name: string;
  description: string;
  severity: Severity;
  start_date: string;
  deadline: string;
  default_pic_user_id: string | null;
  on_hold: boolean;
  actual_pct: number;
  expected_pct: number;
  days_left: number;
  verdict: Verdict;
  signed: boolean;
  signed_by: string | null;
  signed_at: string | null;
  // Terisi (super_user id) kalau signed_at di-backdate manual, bukan sign-off
  // real-time -- lihat decision-log-verdict-backfill-signoff-20260820.md.
  signed_at_backdated_by: string | null;
  updated_by: string | null;
  member_user_ids: string[];
};

export type DayEntry = {
  id: string;
  entry_date: string;
  planned_text: string;
  realisasi_text: string;
  progress_pct: number;
  blocker_text: string;
  is_weekend: boolean;
};

export type DailyTask = {
  id: string;
  big_task_id: string;
  title: string;
  pic_user_id: string;
  start_date: string;
  end_date: string;
  actual_pct: number;
  // true kalau ini Daily Task khusus "Baseline Awal" (persentase progress
  // awal migrasi data, diisi lewat PATCH /big-tasks/{id} baseline_pct) --
  // lihat decision-log-bigtask-baseline-progress-20260824.md.
  is_baseline: boolean;
  days: DayEntry[];
};

// Board Backlog: item planning mentah per board sebelum ditentukan Big
// Task/PIC/tanggal-nya. Reusable (bisa "dipromosikan" jadi Daily Task
// berkali-kali). Lihat decision-log-board-backlog-20260902.md.
export type BacklogItem = {
  id: string;
  board_id: string;
  title: string;
  description: string;
  created_by: string;
  created_by_name: string;
  created_at: string;
  promoted_count: number;
};

export type AssignableUser = {
  id: string;
  display_name: string;
  initials: string;
  org_team: string;
  roles: string[];
};

export type AccessLevel = 'super_user' | 'regular_user';

export type ManagedUser = AssignableUser & {
  email: string;
  access_level: AccessLevel;
  hr_user_id: number | null;
  task_scope_visibility?: 'self' | 'team'; // 2026-08-31
  can_manage_backlog?: boolean; // 2026-09-02, lihat decision-log-board-backlog-20260902.md
};

export type Role = {
  code: string;
  label: string;
};

export type ReferensiTim = {
  id: string;
  name: string;
};

// Data pegawai sistem HR asli (referensi_user_hr) -- dipakai buat mapping di
// Manajemen User, MENGGANTIKAN placeholder CRC32 di push HR. Lihat
// docs/decision-log/decision-log-hr-mapping-super-user-20260810.md.
export type ReferensiUserHR = {
  hr_user_id: number;
  email: string;
  nip: string;
  nama_lengkap: string;
};

export type TeamStatusRow = {
  user_id: string;
  display_name: string;
  initials: string;
  total_daily_tasks: number;
  pushed_daily_tasks: number;
  all_pushed: boolean;
};

export type UserProgressSummary = {
  user_id: string;
  belum: number;
  on_progress: number;
  selesai: number;
  completion_rate: number;
};

export type InAppAlert = {
  type: 'sign_off_ready' | 'verdict_lose' | 'deadline_soon';
  big_task_id: string;
  big_task_name: string;
  board_name: string;
  days_left: number;
  actual_pct: number;
  expected_pct: number;
};

export type NotificationSettings = {
  telegram_chat_id: string;
  telegram_thread_id: string;
  deadline_threshold_days: number;
  cooldown_hours: number;
  notify_sign_off_ready: boolean;
  notify_verdict_lose: boolean;
  notify_deadline_soon: boolean;
};

export type CheatSheetItem = {
  id: string;
  board_id: string;
  type: 'file' | 'url' | 'note';
  title: string;
  value: string;
  author_id: string;
  created_at: string;
};

export type Comment = {
  id: string;
  big_task_id: string;
  daily_task_id: string | null;
  author_id: string;
  body: string;
  created_at: string;
};

export type CRStatus = 'pending' | 'approved' | 'rejected' | 'scheduled';

export type ChangeRequest = {
  id: string;
  submitted_by: string;
  submitted_by_name: string;
  raw_conversation: string;
  document_md: string | null;
  status: CRStatus;
  reviewed_by: string | null;
  reviewed_by_name: string | null;
  reviewed_at: string | null;
  created_at: string;
};

// GET /team-today?date=YYYY-MM-DD -- "apa yang dikerjakan tim hari ini",
// POV per orang. Lihat decision-log-team-today-menu-20260901.md.
export type TeamTodayEntry = {
  day_entry_id: string;
  board_id: string;
  board_name: string;
  big_task_id: string;
  big_task_name: string;
  daily_task_id: string;
  daily_task_title: string;
  planned_text: string;
  realisasi_text: string;
  progress_pct: number;
  blocker_text: string;
};

export type TeamTodayUser = {
  user_id: string;
  display_name: string;
  initials: string;
  org_team: string;
  entries: TeamTodayEntry[];
};
