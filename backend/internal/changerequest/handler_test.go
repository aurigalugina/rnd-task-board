package changerequest

import "testing"

func TestIsValidStatusTransition(t *testing.T) {
	cases := []struct {
		name string
		from string
		to   string
		want bool
	}{
		{"pending -> approved", "pending", "approved", true},
		{"pending -> rejected", "pending", "rejected", true},
		{"pending -> scheduled", "pending", "scheduled", true},
		{"approved -> scheduled", "approved", "scheduled", true},
		{"batal triase: approved -> pending", "approved", "pending", true},
		{"target bukan enum ditolak", "pending", "done", false},
		{"target kosong ditolak", "pending", "", false},
		{"from tak dikenal ditolak", "garbage", "approved", false},
		{"from kosong (baris baru) tetap boleh ke enum valid", "", "approved", true},
	}
	for _, c := range cases {
		if got := isValidStatusTransition(c.from, c.to); got != c.want {
			t.Errorf("%s: isValidStatusTransition(%q,%q) = %v, want %v", c.name, c.from, c.to, got, c.want)
		}
	}
}
