package notification

import "testing"

func allEnabled() UserSettings {
	return UserSettings{
		DeadlineThresholdDays: 3,
		NotifySignOffReady:    true,
		NotifyVerdictLose:     true,
		NotifyDeadlineSoon:    true,
	}
}

func TestClassifyAlert(t *testing.T) {
	tests := []struct {
		name        string
		actualPct   int
		daysLeft    int
		settings    UserSettings
		wantType    string
		wantAlerted bool
	}{
		{"sign off ready takes priority over lose", 100, -5, allEnabled(), SignOffReady, true},
		{"sign off ready takes priority over deadline soon", 100, 1, allEnabled(), SignOffReady, true},
		{"verdict lose when deadline passed and not done", 60, -1, allEnabled(), VerdictLose, true},
		{"deadline soon within threshold", 60, 2, allEnabled(), DeadlineSoon, true},
		{"deadline soon exactly at threshold boundary", 60, 3, allEnabled(), DeadlineSoon, true},
		{"no alert when far from deadline and not done", 60, 10, allEnabled(), "", false},
		{"sign off ready suppressed when disabled, falls through to no alert (still on progress)", 100, 10,
			UserSettings{DeadlineThresholdDays: 3, NotifyVerdictLose: true, NotifyDeadlineSoon: true}, "", false},
		{"verdict lose suppressed when disabled", 60, -1,
			UserSettings{DeadlineThresholdDays: 3, NotifySignOffReady: true, NotifyDeadlineSoon: true}, "", false},
		{"deadline soon suppressed when disabled", 60, 1,
			UserSettings{DeadlineThresholdDays: 3, NotifySignOffReady: true, NotifyVerdictLose: true}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotOK := classifyAlert(tt.actualPct, tt.daysLeft, tt.settings)
			if gotOK != tt.wantAlerted || gotType != tt.wantType {
				t.Errorf("classifyAlert(%d, %d) = (%q, %v), want (%q, %v)",
					tt.actualPct, tt.daysLeft, gotType, gotOK, tt.wantType, tt.wantAlerted)
			}
		})
	}
}
