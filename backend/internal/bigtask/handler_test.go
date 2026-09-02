package bigtask

import (
	"testing"
	"time"
)

func timePtr(t time.Time) *time.Time { return &t }

func TestComputeVerdict(t *testing.T) {
	now := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	deadline := now.AddDate(0, 0, 5)

	cases := []struct {
		name        string
		deadline    time.Time
		signedAt    *time.Time
		now         time.Time
		wantVerdict string
	}{
		// BRD RULE-04: on_progress netral berapa pun gap-nya selama belum lewat deadline.
		{"unsigned, deadline in future", now.AddDate(0, 0, 5), nil, now, "on_progress"},
		{"unsigned, deadline today", now, nil, now, "on_progress"},
		// BRD RULE-06: lose otomatis kalau deadline lewat tanpa sign-off.
		{"unsigned, deadline passed", now.AddDate(0, 0, -1), nil, now, "lose"},
		// BRD RULE-05: win hanya sah kalau sign-off terjadi sebelum/tepat deadline.
		{"signed sebelum deadline", deadline, timePtr(deadline.AddDate(0, 0, -1)), now, "win"},
		{"signed tepat deadline", deadline, timePtr(deadline), now, "win"},
		{"signed setelah deadline (telat)", deadline, timePtr(deadline.AddDate(0, 0, 1)), now, "lose"},
		// Regression decision-log-verdict-backfill-signoff-20260820.md: sign-off
		// SAH on-time di masa lalu HARUS TETAP "win" walau dibaca lama setelah
		// deadline lewat (now jauh di depan) -- computeVerdict dulu pakai `now`
		// (bukan `signedAt`) buat evaluasi win/lose, jadi verdict "berubah" jadi
		// lose seiring waktu berjalan walau keputusan menangnya udah final.
		{
			"signed on-time di masa lalu, dibaca lama setelah deadline lewat",
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			timePtr(time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)),
			time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC),
			"win",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			verdict, _ := computeVerdict(c.deadline, c.signedAt, c.now)
			if verdict != c.wantVerdict {
				t.Errorf("computeVerdict(%v, %v, %v) = %q, want %q", c.deadline, c.signedAt, c.now, verdict, c.wantVerdict)
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

	_, daysLeft := computeVerdict(deadline, nil, now)
	if daysLeft != 3 {
		t.Errorf("daysLeft = %d, want 3", daysLeft)
	}
}

func TestDedupeMembers(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"buang duplikat, jaga urutan", []string{"a", "b", "a", "c", "b"}, []string{"a", "b", "c"}},
		{"buang string kosong", []string{"a", "", "b", ""}, []string{"a", "b"}},
		{"nil -> slice kosong", nil, []string{}},
		{"semua unik tidak berubah", []string{"x", "y"}, []string{"x", "y"}},
	}
	for _, c := range cases {
		got := dedupeMembers(c.in)
		if len(got) != len(c.want) {
			t.Errorf("%s: len = %d, want %d (%v)", c.name, len(got), len(c.want), got)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%s: got %v, want %v", c.name, got, c.want)
				break
			}
		}
	}
}

func TestIsValidSeverity(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"critical", true},
		{"high", true},
		{"medium", true},
		{"low", true},
		{"", false},
		{"CRITICAL", false}, // case-sensitive -- konsisten dgn CHECK constraint DB
		{"urgent", false},
	}
	for _, c := range cases {
		if got := isValidSeverity(c.in); got != c.want {
			t.Errorf("isValidSeverity(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
