package weeklyplan

import "testing"

func TestParseWeekStart(t *testing.T) {
	t.Run("valid Monday returns 6-day-later Sunday as end", func(t *testing.T) {
		start, end, err := parseWeekStart("2026-08-03")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if start != "2026-08-03" {
			t.Errorf("start = %q, want 2026-08-03", start)
		}
		if end != "2026-08-09" {
			t.Errorf("end = %q, want 2026-08-09 (start + 6 days)", end)
		}
	})

	t.Run("crosses month boundary correctly", func(t *testing.T) {
		_, end, err := parseWeekStart("2026-08-31")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if end != "2026-09-06" {
			t.Errorf("end = %q, want 2026-09-06", end)
		}
	})

	t.Run("missing/empty week_start is rejected", func(t *testing.T) {
		_, _, err := parseWeekStart("")
		if err == nil {
			t.Fatal("expected error for empty week_start, got nil")
		}
	})

	t.Run("malformed week_start is rejected", func(t *testing.T) {
		_, _, err := parseWeekStart("2026/08/03")
		if err == nil {
			t.Fatal("expected error for malformed week_start, got nil")
		}
	})
}

func TestPlaceholderHRUserID(t *testing.T) {
	t.Run("deterministic for the same UUID", func(t *testing.T) {
		id1 := placeholderHRUserID("33bee6fa-1768-4d15-9ffc-a1c558d9cd5a")
		id2 := placeholderHRUserID("33bee6fa-1768-4d15-9ffc-a1c558d9cd5a")
		if id1 != id2 {
			t.Errorf("placeholderHRUserID not deterministic: %d != %d", id1, id2)
		}
	})

	t.Run("always positive, non-zero (my_agenda.user_id is NOT NULL int)", func(t *testing.T) {
		uuids := []string{
			"33bee6fa-1768-4d15-9ffc-a1c558d9cd5a",
			"f45dadf6-65c2-4bce-9fe1-2ed5bd04c1b6",
			"", // edge case: empty string shouldn't crash or produce 0
		}
		for _, u := range uuids {
			id := placeholderHRUserID(u)
			if id <= 0 {
				t.Errorf("placeholderHRUserID(%q) = %d, want > 0", u, id)
			}
		}
	})

	t.Run("different UUIDs usually produce different ids", func(t *testing.T) {
		id1 := placeholderHRUserID("33bee6fa-1768-4d15-9ffc-a1c558d9cd5a")
		id2 := placeholderHRUserID("f45dadf6-65c2-4bce-9fe1-2ed5bd04c1b6")
		if id1 == id2 {
			t.Errorf("expected different ids for different UUIDs, both got %d", id1)
		}
	})
}

func TestResolveTargetUserID(t *testing.T) {
	t.Run("no as_user_id -- always resolves to requester, regardless of access level", func(t *testing.T) {
		id, ok := resolveTargetUserID("requester-id", "", false)
		if !ok || id != "requester-id" {
			t.Errorf("got (%q, %v), want (requester-id, true)", id, ok)
		}

		id, ok = resolveTargetUserID("requester-id", "", true)
		if !ok || id != "requester-id" {
			t.Errorf("got (%q, %v), want (requester-id, true)", id, ok)
		}
	})

	t.Run("as_user_id honored for super_user", func(t *testing.T) {
		id, ok := resolveTargetUserID("requester-id", "target-id", true)
		if !ok || id != "target-id" {
			t.Errorf("got (%q, %v), want (target-id, true)", id, ok)
		}
	})

	t.Run("as_user_id rejected (not silently ignored) for regular_user", func(t *testing.T) {
		id, ok := resolveTargetUserID("requester-id", "target-id", false)
		if ok || id != "" {
			t.Errorf("got (%q, %v), want (\"\", false) -- regular_user must not be able to view/push as someone else", id, ok)
		}
	})
}
