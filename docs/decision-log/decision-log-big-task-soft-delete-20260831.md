# Decision Log: Big Task Soft Delete

**Date:** 2026-08-31  
**Context:** User action / data safety / cascading deletion  
**Status:** Final

---

## Problem

Need ability for super_user to delete Big Task records from portal, with safety guardrails:
- Prevent accidental data loss
- Maintain referential integrity (daily_tasks, comments, day_entries, etc remain in DB)
- Hide deleted Big Task + child records from application queries/UI
- Leave hard-delete option open for system administrators only (via raw SQL)

## Decision

**Soft Delete Pattern** — Mark Big Task as deleted via `deleted_at` timestamp + `deleted_by` audit trail, instead of hard DELETE from database.

### Implementation Details

**1. Schema (Migration 0025)**
- Add `big_tasks.deleted_at TIMESTAMPTZ NULL` — NULL = not deleted, non-NULL = deletion timestamp
- Add `big_tasks.deleted_by UUID NULL REFERENCES users(id)` — audit trail: who deleted it

**2. Backend Handler**
- DELETE /big-tasks/{id} endpoint (super_user only, checked in-handler via `auth.IsSuperUser()`)
- Soft delete: `UPDATE big_tasks SET deleted_at = now(), deleted_by = $userID, updated_at = now() WHERE id = $1`
- 204 No Content response on success
- Returns 403 Forbidden if non-super_user attempts delete

**3. Query Layer**
- All queries that fetch big_tasks now filter `bt.deleted_at IS NULL`
  - `ListByBoard()` — exclude deleted
  - `loadBigTask()` — exclude deleted (used by SignOff, UndoSignOff, Update, etc.)
- Child records (daily_tasks, comments, day_entries, etc.) remain in DB, just invisible because parent Big Task is filtered

**4. Frontend**
- Delete button in Edit Big Task modal (super_user only, conditional rendering)
- Red text styling (visual danger indicator)
- Confirmation dialog before delete: "Hapus Big Task \"[name]\"? Data tidak bisa dikembalikan."
- On success: close modal, refresh board list
- On error: display error message in modal

**5. Route** (main.go)
- `protected.Delete("/big-tasks/{bigTaskID}", bigTaskHandler.Delete)`
- In `protected` group (not behind RequireRole, since super_user is access_level not role)
- Placed logically next to other big_task mutations (PATCH Update)

---

## Why This Approach

| Option | Pros | Cons |
|--------|------|------|
| **Hard DELETE** | Clean schema, no extra columns | Irreversible, high risk of accidents, cascade cleanup needed |
| **Soft DELETE (chosen)** | Safe, auditable, referential integrity preserved, data recovery possible | Requires `deleted_at IS NULL` in every query, schema clutter |
| **Archive flag** | Similar safety, simpler query | Semantically "archive" ≠ "delete", confusing for users |

Soft delete is industry standard for financial/audit-trail systems (our SRS emphasizes audit trails).

---

## Impact / Files Changed

**Database:**
- `backend/db/migrations/0025_big_task_soft_delete.up.sql` — add columns
- `backend/db/migrations/0025_big_task_soft_delete.down.sql` — rollback

**Backend:**
- `backend/internal/bigtask/handler.go`:
  - New `Delete()` method (soft delete, super_user check)
  - Updated `ListByBoard()` query filter
  - Updated `loadBigTask()` query filter
- `backend/cmd/api/main.go`:
  - New route: `protected.Delete("/big-tasks/{bigTaskID}", bigTaskHandler.Delete)`

**Frontend:**
- `frontend/src/lib/components/BigTaskList.svelte`:
  - New `deleteBigTask()` function
  - Import `Trash2` icon
  - Conditional delete button in edit modal (super_user only)
  - Call `api.del("/big-tasks/{id}")`

---

## Testing

- ✅ Go unit tests pass (existing compute/verify logic unaffected)
- ✅ Frontend svelte-check: 0 errors, 0 warnings
- ✅ Soft delete: 204 No Content on success
- ✅ Permission check: 403 Forbidden if non-super_user
- Manual browser test: delete button appears only for super_user, shows confirmation, closes modal on success

---

## Future Work

- Consider adding undelete endpoint (restore `deleted_at = NULL`) if business asks
- Monitor query performance (might want index on `(board_id, deleted_at)` for large datasets)
- Audit report: query deleted Big Tasks via `deleted_at IS NOT NULL` + `deleted_by` user info

---

## Notes

- Data is **not truly gone** — super_user can restore via `UPDATE big_tasks SET deleted_at = NULL WHERE id = $1` if needed
- Child records (daily_tasks, comments, day_entries, weekly_push_log, etc.) **remain in DB** — they just become invisible because parent Big Task is filtered
- This pattern matches existing "archive" behavior for boards (decision-log-board-archive-20260812.md)
