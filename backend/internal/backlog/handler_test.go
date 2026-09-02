package backlog

import "testing"

func TestCanManageBacklog(t *testing.T) {
	cases := []struct {
		name          string
		isSuperUser   bool
		userCanManage bool
		want          bool
	}{
		{"super_user selalu boleh, terlepas flag", true, false, true},
		{"super_user dengan flag juga boleh", true, true, true},
		{"regular user dengan flag true -> boleh", false, true, true},
		{"regular user tanpa flag -> ditolak", false, false, false},
	}
	for _, c := range cases {
		if got := canManageBacklog(c.isSuperUser, c.userCanManage); got != c.want {
			t.Errorf("%s: canManageBacklog(%v, %v) = %v, want %v",
				c.name, c.isSuperUser, c.userCanManage, got, c.want)
		}
	}
}
