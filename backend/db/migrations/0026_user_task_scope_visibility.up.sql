-- Migration 0026: Add task_scope_visibility to users
--
-- Adds task_scope_visibility column to control whether a regular user sees only
-- their own assigned tasks (self) or all team tasks (team, default).
-- Super user is unaffected by this flag — always sees everything based on org_team + permissions.

ALTER TABLE users ADD COLUMN task_scope_visibility TEXT NOT NULL DEFAULT 'team';
ALTER TABLE users ADD CONSTRAINT check_task_scope_visibility 
  CHECK (task_scope_visibility IN ('self', 'team'));

COMMENT ON COLUMN users.task_scope_visibility IS 
'Scope of task visibility for this user: "team" = sees all tasks in their org_team (default), "self" = sees only tasks they are assigned to (based on big_task_members). Super users always see all. Set by admin.';
