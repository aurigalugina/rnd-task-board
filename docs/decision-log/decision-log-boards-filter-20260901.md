# Board Filters: Status & Name Search with Polished UI/UX

**Date:** 2026-09-01  
**Status:** Implemented & Pushed ✅

## Context

Previously, Boards page showed all boards in a horizontal pill row (no filtering). User wanted two filter controls:
1. **Status filter** — show all boards vs. incomplete boards only
2. **Name search** — real-time input to search boards by name

Also requested: **polished UI/UX** matching the retro-OS aesthetic of the app.

## Design Decisions

### Filter Section Placement & Layout
- **Filter section above board pills** — follows conventional top-down flow (filter → results)
- **Horizontal layout** (flexbox) to keep interface compact and scannable
- **Grouped controls**: Search input on left, Status buttons on right (or wraps on narrow screens)
- **Visual separation**: Light background (`--content-alt`) + border to distinguish from main board area

### Status Filter
- **Two states**: "Semua" (all) | "Belum Selesai" (incomplete)
- **UI pattern**: Toggle buttons (not dropdown) for visibility — users see both options at a glance
- **Active state**: Filled background (`--win-blue`) + white text to indicate selection
- **MVP limitation**: Backend doesn't yet return board.status or completion percentage. For now, "Belum Selesai" shows all boards as placeholder (TODO: implement when Board type has status field)

### Name Search
- **Real-time filtering**: As user types, board pills update instantly
- **Input styling**: Matches existing form inputs (retro inset border, 1px solid)
- **Placeholder text**: "Ketik nama board..." — clear instruction
- **Case-insensitive**: "DASHBOARD" matches "dashboard"

### CSS & Theming
- **Consistent with design system**: Uses custom properties (`--face`, `--content-alt`, `--text-primary`, `--win-blue`)
- **Dark theme support**: Separate selectors for `retro-dark` and `modern-dark` themes
- **Hover states**: Subtle border + background changes (0.15s transition)
- **Accessibility**: Proper focus states, `min-width` on inputs to prevent squishing on narrow screens

## Implementation

### Frontend Changes

**File:** `frontend/src/routes/boards/+page.svelte`

1. **State** (lines ~48-49):
   ```typescript
   let filterStatus: 'all' | 'not-done' = 'all';
   let searchBoardName = '';
   ```

2. **Computed reactive** (lines ~67-83):
   ```typescript
   $: filteredBoards = boards.filter((b) => {
     const matchName = b.name.toLowerCase().includes(searchBoardName.toLowerCase());
     if (!matchName) return false;
     if (filterStatus === 'not-done') {
       // TODO: implement when Board type has status field
       return true;
     }
     return true;
   });
   ```

3. **Filter UI section** (lines ~169-200):
   - Search input with label
   - Status buttons (Semua / Belum Selesai)

4. **Updated loop** (line ~201):
   ```svelte
   {#each filteredBoards as board (board.id)}
   ```
   Changed from `boards` to `filteredBoards`.

**File:** `frontend/src/app.css`

Added ~90 lines of CSS at end of file:
- `.boards-filter-section` — container
- `.filter-container`, `.filter-group` — layout
- `.filter-input` — search input styling
- `.filter-btn`, `.filter-btn-active` — status toggle buttons
- Dark theme overrides (`:root[data-theme='retro-dark']`, `modern-dark`)

### Visual Features

✅ **Responsive**: Filters wrap on narrow screens (flexbox)  
✅ **Consistent aesthetic**: Retro-OS inset borders, button styles match app  
✅ **Hover feedback**: Buttons highlight on hover, input gets blue border on focus  
✅ **Accessibility**: Labels, focus states, clear contrast  
✅ **Theme-aware**: Adapts to all theme variants (retro light/dark, modern light/dark)

## Testing

Manual testing checklist:
- [ ] Search "Dashboard" → filters to boards containing "dashboard" (case-insensitive)
- [ ] Clear search → all boards reappear
- [ ] Click "Belum Selesai" → (placeholder) shows all boards (TODO when backend adds status)
- [ ] Click "Semua" → toggles back
- [ ] Responsive: on narrow screen (mobile), filters wrap to multiple lines
- [ ] Dark theme: switch theme in settings → filter section adapts colors

## Future Work

1. **Backend status tracking**: Board type needs `status` field (e.g., "done" / "in_progress" / "not_started")
   - OR: Count completed big tasks (`SELECT COUNT(*) ... WHERE signed = true`) and return as `completed_big_tasks_count`
   - Then implement actual filter logic in `filteredBoards` computed

2. **Filter persistence** (optional): Save filter preferences to localStorage so they persist across sessions

3. **Advanced filters** (nice-to-have): 
   - Filter by category (project / routine)
   - Filter by team membership
   - Filter by recent activity

## Files Modified

- `frontend/src/routes/boards/+page.svelte` — logic + UI
- `frontend/src/app.css` — styling
- `docs/decision-log/decision-log-day-entry-edit-window-20260901.md` — (from previous commit)
- `nginx/upgrade-map.conf` — (unrelated, auto-generated)

## Commits

- `c333daf` — "feat: add board filters (status + name search) with UI/UX polish"

## Related Issues / PRs

None currently tracked.
