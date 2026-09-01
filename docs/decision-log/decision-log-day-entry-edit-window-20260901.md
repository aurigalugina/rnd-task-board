# Allow Day Entry Edits Within 3 Days Past (Previously All Past Locked)

**Date:** 2026-09-01  
**Status:** Implemented & Tested ✅

## Context

Current behavior: day entries that are in the past (entry_date < today) are completely locked as read-only in the modal. User feedback: this is too restrictive — should allow editing recent entries (e.g., yesterday's or 2 days ago's entry if you realized you mistyped something).

## Decision

**Entries can be edited if they are within 3 days back from today.** Entries 3+ days old are locked as read-only. Entries in the future remain editable.

- **Days 0 to 3 back** (today, yesterday, 2 days ago, 3 days ago): ✏️ **Editable**
- **Day 4+ back**: 🔒 **Read-only**
- **Future days**: ✏️ **Editable**

## Rationale

- **Real-world use case:** PIC often logs entries same-day or next day, but may review/fix 1-2 days back without formal re-submission. 3-day window balances **correctability** vs **audit trail immutability** (older records should not be casually edited).
- **No backend change needed:** Backend (`PATCH /day-entries/{id}`) has no validation on age — all the protection is in UI. Could add backend validation for security, but frontend alone is sufficient for this non-sensitive operation.
- **Consistent with retro-OS design:** Users understand "disabling input" as read-only, no extra modal variant needed.

## Implementation

### Frontend Changes

**File:** `frontend/src/lib/components/DailyTaskPanel.svelte`

Replaced function `isPastDate()` (simple `entryDate < today`) with `isDayOlderThan3Days()`:

```typescript
function isDayOlderThan3Days(entryDate: string): boolean {
  const entryDateObj = new Date(`${entryDate}T00:00:00Z`);
  const todayObj = new Date(`${today}T00:00:00Z`);
  const diffMs = todayObj.getTime() - entryDateObj.getTime();
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
  return diffDays > 3;
}
```

Updated call site (line ~313):
```
{@const past = isDayOlderThan3Days(day.entry_date)}
```

**File:** `frontend/src/lib/components/DayEntryEditModal.svelte`

No change needed — already respects the `isPast` prop passed in from parent, disabling inputs when true. Works transparently with new logic.

### Testing

Created `frontend/src/lib/editable-days.spec.ts` with 7 unit tests (all ✅ passing):

- Future dates → editable
- Today → editable
- 1 day ago → editable
- 3 days ago → editable
- 4 days ago → locked (first failing day)
- 7 days ago → locked
- 30 days ago → locked

Ran via `npm run test` — all pass.

## Impact

- **Boards page, day entries section:** Users can now correct/update entries up to 3 days back. Entries older than that remain audited and immutable from UI.
- **No API changes:** Backend endpoint `PATCH /day-entries/{id}` still accepts updates for any date (security could be tightened later if needed).
- **Backward compatible:** No schema changes, no migration, zero breaking changes.

## Future Considerations

- If audit log becomes critical (e.g., "who edited this entry when"), add a `changed_by` / `changed_at` column to `day_entries` and log all PATCH requests.
- Could make the **3-day window configurable** via a settings field if different teams want different windows (e.g., data-entry team wants 7 days, compliance wants 1 day). Left as hard-coded 3 for now per user request.
- Backend could validate age and reject PATCH on entries >3 days old as a security layer (prevents accidental API misuse), though frontend alone is fine for MVP.
