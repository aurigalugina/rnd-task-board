// Bentuk data mengikuti docs/05-api-contract.md — field turunan (actual_pct,
// expected_pct, verdict, completion_rate, is_weekend) selalu dari server,
// jangan pernah dihitung ulang di frontend.

export type Board = {
  id: string;
  name: string;
  tag: string;
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

export type BigTask = {
  id: string;
  board_id: string;
  name: string;
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
};

export type DayEntry = {
  id: string;
  entry_date: string;
  planned_text: string;
  is_done: boolean;
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
  days: DayEntry[];
};

export type AssignableUser = {
  id: string;
  display_name: string;
  initials: string;
  org_team: string;
  roles: string[];
};

export type ManagedUser = AssignableUser & {
  email: string;
};

export type Role = {
  code: string;
  label: string;
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
