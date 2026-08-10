package bigtask

import (
	"testing"
	"time"
)

func TestComputeVerdict(t *testing.T) {
	now := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name        string
		deadline    time.Time
		signed      bool
		wantVerdict string
	}{
		// BRD RULE-04: on_progress netral berapa pun gap-nya selama belum lewat deadline.
		{"unsigned, deadline in future", now.AddDate(0, 0, 5), false, "on_progress"},
		{"unsigned, deadline today", now, false, "on_progress"},
		// BRD RULE-06: lose otomatis kalau deadline lewat tanpa sign-off.
		{"unsigned, deadline passed", now.AddDate(0, 0, -1), false, "lose"},
		// BRD RULE-05: win hanya sah kalau sign-off terjadi sebelum/tepat deadline.
		{"signed, deadline in future", now.AddDate(0, 0, 5), true, "win"},
		{"signed, deadline today", now, true, "win"},
		{"signed, deadline already passed", now.AddDate(0, 0, -1), true, "lose"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			verdict, _ := computeVerdict(c.deadline, c.signed, now)
			if verdict != c.wantVerdict {
				t.Errorf("computeVerdict(%v, %v, now) = %q, want %q", c.deadline, c.signed, verdict, c.wantVerdict)
			}
		})
	}
}

func TestComputeExpectedPct(t *testing.T) {
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	deadline := start.AddDate(0, 0, 10)

	cases := []struct {
		name string
		now  time.Time
		want int
	}{
		{"belum mulai", start.AddDate(0, 0, -2), 0},
		{"tepat mulai", start, 0},
		{"separuh jalan", start.AddDate(0, 0, 5), 50},
		{"tepat deadline", deadline, 100},
		{"lewat deadline diklem ke 100", deadline.AddDate(0, 0, 5), 100},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := computeExpectedPct(start, deadline, c.now)
			if got != c.want {
				t.Errorf("computeExpectedPct(...) = %d, want %d", got, c.want)
			}
		})
	}
}

func TestComputeExpectedPctMinimumOneDayDuration(t *testing.T) {
	// start_date == deadline (durasi < 1 hari) tidak boleh divide-by-zero.
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	got := computeExpectedPct(start, start, start)
	if got != 0 {
		t.Errorf("computeExpectedPct dengan totalDays=0 = %d, want 0 (bukan panic)", got)
	}
}

func TestComputeVerdictDaysLeft(t *testing.T) {
	now := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	deadline := now.AddDate(0, 0, 3)

	_, daysLeft := computeVerdict(deadline, false, now)
	if daysLeft != 3 {
		t.Errorf("daysLeft = %d, want 3", daysLeft)
	}
}
