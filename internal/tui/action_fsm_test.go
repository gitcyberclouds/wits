package tui

import "testing"

func TestActionMenuSelect(t *testing.T) {
	tests := []struct {
		key     string
		pending actionKind
		close   bool
	}{
		{"k", actionKill, true}, {"K", actionKill, true},
		{"t", actionTerm, true}, {"T", actionTerm, true},
		{"p", actionPause, true}, {"P", actionPause, true},
		{"r", actionResume, true}, {"R", actionResume, true},
		{"n", actionRenice, true}, {"N", actionRenice, true},
		{"esc", actionNone, true}, {"q", actionNone, true}, {"Q", actionNone, true},
		{"z", actionNone, false}, {"enter", actionNone, false},
	}
	for _, tt := range tests {
		p, c := actionMenuSelect(tt.key)
		if p != tt.pending || c != tt.close {
			t.Errorf("actionMenuSelect(%q) = (%v, %v), want (%v, %v)", tt.key, p, c, tt.pending, tt.close)
		}
	}
}

func TestConfirmKey(t *testing.T) {
	tests := []struct {
		key  string
		want confirmDecision
	}{
		{"y", confirmExecute}, {"Y", confirmExecute},
		{"n", confirmCancel}, {"N", confirmCancel}, {"esc", confirmCancel},
		{"z", confirmIgnore}, {"", confirmIgnore},
	}
	for _, tt := range tests {
		if got := confirmKey(tt.key); got != tt.want {
			t.Errorf("confirmKey(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}

func TestValidateNiceValue(t *testing.T) {
	tests := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{"0", 0, false},
		{"19", 19, false},
		{"-20", -20, false},
		{"20", 0, true},  // above range
		{"-21", 0, true}, // below range
		{"abc", 0, true}, // not a number
		{"  5  ", 5, false},
	}
	for _, tt := range tests {
		got, err := validateNiceValue(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("validateNiceValue(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if err == nil && got != tt.want {
			t.Errorf("validateNiceValue(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
