package dailytask

import (
	"testing"
	"time"
)

func TestIsWeekend(t *testing.T) {
	cases := []struct {
		name string
		date string
		want bool
	}{
		{"Saturday", "2026-08-08", true},
		{"Sunday", "2026-08-09", true},
		{"Monday", "2026-08-10", false},
		{"Friday", "2026-08-07", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, err := time.Parse("2006-01-02", c.date)
			if err != nil {
				t.Fatalf("failed to parse fixture date: %v", err)
			}
			if got := isWeekend(d); got != c.want {
				t.Errorf("isWeekend(%s) = %v, want %v", c.date, got, c.want)
			}
		})
	}
}

func TestParseDateRange(t *testing.T) {
	t.Run("valid inclusive range", func(t *testing.T) {
		start, end, err := parseDateRange("2026-08-05", "2026-08-07")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if start.Format("2006-01-02") != "2026-08-05" || end.Format("2006-01-02") != "2026-08-07" {
			t.Errorf("got start=%v end=%v", start, end)
		}
	})

	t.Run("start equals end is valid (single day)", func(t *testing.T) {
		_, _, err := parseDateRange("2026-08-05", "2026-08-05")
		if err != nil {
			t.Errorf("expected no error for equal start/end, got %v", err)
		}
	})

	t.Run("end before start is rejected", func(t *testing.T) {
		_, _, err := parseDateRange("2026-08-07", "2026-08-05")
		if err == nil {
			t.Fatal("expected error when end_date is before start_date, got nil")
		}
	})

	t.Run("malformed start_date is rejected", func(t *testing.T) {
		_, _, err := parseDateRange("not-a-date", "2026-08-05")
		if err == nil {
			t.Fatal("expected error for malformed start_date, got nil")
		}
	})

	t.Run("malformed end_date is rejected", func(t *testing.T) {
		_, _, err := parseDateRange("2026-08-05", "not-a-date")
		if err == nil {
			t.Fatal("expected error for malformed end_date, got nil")
		}
	})
}

func TestCanEditDayEntry(t *testing.T) {
	cases := []struct {
		name        string
		userID      string
		picUserID   string
		isSuperUser bool
		userScope   string
		want        bool
	}{
		{"pemilik entry selalu boleh, scope self", "u1", "u1", false, "self", true},
		{"pemilik entry selalu boleh, scope team", "u1", "u1", false, "team", true},
		{"pemilik entry selalu boleh, super_user", "u1", "u1", true, "self", true},
		{"bukan pemilik, scope self -- DITOLAK", "u2", "u1", false, "self", false},
		{"bukan pemilik, scope team -- boleh (akses lihat semua orang)", "u2", "u1", false, "team", true},
		{"bukan pemilik, super_user -- boleh", "u2", "u1", true, "self", true},
		{"bukan pemilik, super_user DAN scope team -- boleh", "u2", "u1", true, "team", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := canEditDayEntry(c.userID, c.picUserID, c.isSuperUser, c.userScope)
			if got != c.want {
				t.Errorf("canEditDayEntry(%q, %q, %v, %q) = %v, want %v",
					c.userID, c.picUserID, c.isSuperUser, c.userScope, got, c.want)
			}
		})
	}
}
