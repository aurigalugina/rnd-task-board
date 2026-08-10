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
