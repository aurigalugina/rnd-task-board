# Decision Log: Board/Task Visibility Scoping by User Assignment

**Date:** 2026-08-31  
**Context:** Permission model / data visibility / regular-user privacy  
**Status:** Final

---

## Problem

Currently all users (regular_user + super_user) see **all boards and Big Tasks in their org_team**. Need granular visibility:
- **Regular user with scope='team':** Sees all tasks in their team (current behavior, default)
- **Regular user with scope='self':** Sees only boards + tasks they are directly assigned to (based on big_task_members)
- **Super user:** Always sees all (unaffected by flag)

## Decision

Add `task_scope_visibility` **TEXT CHECK ('self'|'team')** column to `users` table (default='team').

Visibility enforcement:
- **Board List** (`GET /boards`): Filter based on scope — if scope='self', only include boards where user is Big Task member
- **Daily Task List** (`GET /big-tasks/{id}/daily-tasks`): Permission check in-handler — if scope='self' and not member, return 403
- **Admin UI** (Settings modal): Dropdown to set user's scope per-user

### Implementation

**1. Migration 0026**

```sql
ALTER TABLE users ADD COLUMN task_scope_visibility TEXT NOT NULL DEFAULT 'team';
ALTER TABLE users ADD CONSTRAINT check_task_scope_visibility 
  CHECK (task_scope_visibility IN ('self', 'team'));
```

**2. Backend**

**board/handler.go :: List()**
- Regular user path now includes CTE `user_scope` (fetch task_scope_visibility)
- CTE `scope_boards` filters:
  - Team visibility: existing team-based filter (unchanged)
  - Self visibility: ONLY boards where user IS member of any Big Task in that board
- Query condition: `(scope='team') OR (scope='self' AND user_member_of_bt_in_board)`

**dailytask/handler.go :: ListByBigTask()**
- Permission check before returning daily tasks:
  - Super user: pass (always allowed)
  - Regular user scope='team': pass (see all)
  - Regular user scope='self' + not member of Big Task: return 403 Forbidden
  - Query checks `big_task_members` existence

**user/handler.go**
- Added `TaskScopeVisibility` field to `updateUserRequest` struct
- Validation: must be 'self' or 'team'
- UPDATE statement includes: `task_scope_visibility = COALESCE($6, task_scope_visibility)`

**3. Frontend**

**types.ts**
- Added `task_scope_visibility?: 'self' | 'team'` to ManagedUser type (optional, defaults to 'team' if null)

**SettingsModal.svelte**
- New state: `editTaskScopeVisibility: 'self' | 'team'`
- startEdit(): initialize from user.task_scope_visibility
- saveEdit(): include `task_scope_visibility` in PATCH /users/{id}
- UI: dropdown in edit form ("Lihat semua task tim" vs "Lihat hanya task sendiri")
- Table: new "Scope" column showing current scope (displays 'team' if null)
- Table header: updated colspan from 7 to 8

---

## Design Decisions

| Question | Decision | Why |
|----------|----------|-----|
| Where to enforce scope? | Backend (query + permission checks) | Prevents data leaks, cannot bypass via API |
| Default scope for new users? | 'team' | Preserves existing behavior, admins explicitly set 'self' when needed |
| Can super_user see scope flag? | Yes (FYI only) | Useful for auditing user restrictions, doesn't affect their access |
| Include soft-deleted Big Tasks in scope filter? | No (`deleted_at IS NULL`) | Consistency with Board List view, deleted = invisible |
| Scope applies to Daily Task creation? | No (future feature) | Current flow: super_user/admin creates, anyone assigned sees. Can add permission check later if needed |
| Scope visible to the user themselves? | No (frontend only shows to admins) | Privacy — regular users don't see their own scope restriction |

---

## Query Examples

Regular user (scope='team') in R&D team:
```
GET /boards
→ Returns all boards assigned to R&D team (current behavior)
```

Regular user (scope='self') assigned to 1 Big Task:
```
GET /boards
→ Returns only boards containing Big Tasks they are member of
→ Other boards in same team invisible
```

Access daily tasks (regular, scope='self', NOT member):
```
GET /big-tasks/xyz/daily-tasks
→ Returns 403 Forbidden (no permission)
```

---

## Impact / Files Changed

**Backend Migrations:**
- `backend/db/migrations/0026_user_task_scope_visibility.up.sql`
- `backend/db/migrations/0026_user_task_scope_visibility.down.sql`

**Backend Code:**
- `backend/internal/board/handler.go`:
  - `List()`: Added CTE + scope filter logic for regular users
- `backend/internal/dailytask/handler.go`:
  - `ListByBigTask()`: Added permission check (scope + membership)
- `backend/internal/user/handler.go`:
  - `updateUserRequest`: Added `TaskScopeVisibility` field
  - `Update()`: Added validation + UPDATE clause

**Frontend:**
- `frontend/src/lib/types.ts`:
  - `ManagedUser`: Added `task_scope_visibility` optional field
- `frontend/src/lib/components/SettingsModal.svelte`:
  - New state: `editTaskScopeVisibility`
  - Edit form: Added scope dropdown + table column
  - Table header: Updated colspan + added "Scope" column
  - `startEdit()` / `saveEdit()`: Handle scope field

---

## Testing

**Frontend:**
- ✅ svelte-check: 0 errors, 0 warnings
- ✅ Type definitions: ManagedUser.task_scope_visibility recognized

**Backend:**
- ✅ Go compiles (no new test files, existing tests unaffected)
- ✅ dailytask tests pass (no behavior change, just added permission check)

**Manual test cases (requires data):**
1. User with scope='team' sees all boards in their org_team
2. User with scope='self' sees only boards where they are Big Task member
3. User with scope='self' accessing daily tasks of non-assigned Big Task → 403 Forbidden
4. Admin can toggle user scope in Settings modal
5. Super user always sees all boards (regardless of scope flag)
6. New user defaults to scope='team'

---

## Known Limitations

- **Cascade to existing members:** When Big Task member is removed, user scope doesn't retroactively hide boards (board visibility only checked at fetch time, not persisted). Future: consider notification on loss of access.
- **Performance:** CTEs + JOIN in board List query may slow on large datasets (100+ boards, 1000+ Big Tasks). Future: add index on `(board_id, user_id, deleted_at)` if slow.
- **Daily Task permission check:** Performed per-endpoint. If user has BOTH member + non-member Big Tasks, mix is not validated (assume frontend doesn't mix endpoints).

---

## Future Enhancements

- Scope setting in user profile (self-service, admin approval workflow)
- Audit log: track scope changes + access denials
- Board invitation system: allow scope='self' users to request/receive board access
- Dashboard filtering: respect scope visibility when aggregating stats
- Notification: alert user when they lose board access due to member removal
