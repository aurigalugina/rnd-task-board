# Decision Log: Board Sorting by Progress & Deadline

**Date:** 2026-08-31  
**Context:** Dashboard UX / query flexibility  
**Status:** Final

---

## Problem

Dashboard displays boards (projects) in fixed order (created_at). Users need ability to sort projects by:
1. **Progress** — average completion percentage across all Big Tasks in board
2. **Due Date** — earliest deadline from Big Tasks in board

With optional **sort order** (asc = low-to-high, desc = high-to-low).

## Decision

Add **query parameters** to `GET /boards` endpoint:
- `?sort_by=progress|duedate` (optional, default = created_at)
- `?sort_order=asc|desc` (optional, default = asc when sort_by is set)

### Implementation

**1. Backend (board/handler.go)**

- Accept `sort_by` & `sort_order` query params, validate values
- Add stats subquery (left join to big_tasks):
  - `avg_progress` — ROUND(AVG(progress_pct)) across all Big Tasks + day_entries
  - `earliest_deadline` — MIN(deadline) from Big Tasks (where deleted_at IS NULL)
- Build dynamic ORDER BY clause using fmt.Sprintf:
  - Default: `b.created_at ASC`
  - Progress: `COALESCE(stats.avg_progress, 0) {asc|desc}, b.created_at ASC`
  - Duedate: `COALESCE(stats.earliest_deadline, CURRENT_DATE) {asc|desc}, b.created_at ASC`
- Both super_user and regular_user paths updated identically

**2. Frontend (+page.svelte)**

- Add state: `sortBy` (progress|duedate|''), `sortOrder` (asc|desc)
- Add sort dropdowns in dash-filters section:
  - "Urutkan default" (sortBy='')
  - "Progress"
  - "Deadline"
  - When sortBy is set, show second dropdown for direction
- Pass sort params to `GET /boards` query string in `loadDashboard()`
- UI labels: "Rendah → Tinggi" (asc), "Tinggi → Rendah" (desc)

**3. Imports**

- `backend/internal/board/handler.go` — add `import "fmt"` for dynamic queries

---

## Design Decisions

| Question | Decision | Why |
|----------|----------|-----|
| Compute progress on-the-fly or cache? | On-the-fly in query | Simpler, always fresh, no cache invalidation needed |
| Include deleted Big Tasks in sort metrics? | No (filter `deleted_at IS NULL`) | Consistency with List view, deleted = invisible |
| Secondary sort key? | `b.created_at ASC` | Deterministic order when primary sort key ties |
| Default sort order when sort_by unset? | created_at ASC | Familiar FIFO behavior, no surprises |
| Frontend sorts OR backend sorts? | Backend (query-level) | More efficient, regular users on large boards don't fetch all boards |

---

## Query Example

Sorting projects by progress (highest first):
```
GET /boards?sort_by=progress&sort_order=desc
```

Sorting by nearest deadline:
```
GET /boards?sort_by=duedate&sort_order=asc
```

---

## Impact / Files Changed

**Backend:**
- `backend/internal/board/handler.go`:
  - Import `fmt`
  - Updated `List()` method: extract sort params, build dynamic ORDER BY, add stats subquery
  - Both super_user and regular_user query paths use same logic

**Frontend:**
- `frontend/src/routes/+page.svelte`:
  - New state: `sortBy`, `sortOrder`
  - Updated `loadDashboard()`: pass sort params to query string
  - New UI: two dropdowns for sort preference in dash-filters section

---

## Testing

- ✅ Frontend svelte-check: 0 errors, 0 warnings
- ✅ Go compiles (no new test files, but existing bigtask/board tests unaffected)
- Manual test cases:
  - Sort by progress (asc) — shows boards with lowest project progress first
  - Sort by progress (desc) — shows highest progress first
  - Sort by duedate (asc) — nearest deadline first
  - Sort by duedate (desc) — furthest deadline first
  - Default (no params) — sorted by created_at
  - Boards with no Big Tasks — avg_progress=0, earliest_deadline=CURRENT_DATE, sort deterministically

---

## Known Limitations

- **Ties:** When multiple boards have same progress/deadline, secondary sort by created_at ensures stable order
- **Performance:** Stats subquery joins 3 tables (big_tasks, daily_tasks, day_entries) — may slow down on very large datasets (future: add index on `(board_id, deleted_at)` if needed)
- **Soft-deleted Big Tasks:** Excluded from sort metrics (consistent with dashboard view)
- **On-hold Big Tasks:** Included in averages (treated same as running tasks) — could be revisited if UX asks for separate metric

---

## Future Enhancements

- Add "% On-Hold" sorting (boards most paused)
- Persistent sort preference (store in localStorage or user settings)
- Multi-column sort (primary + secondary sort key picker)
- Saved sort views for super_user (board grouping presets)
