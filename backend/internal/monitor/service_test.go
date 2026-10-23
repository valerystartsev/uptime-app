package monitor

import "testing"

func TestValidURL(t *testing.T) {
	for _, tc := range []struct {
		value string
		valid bool
	}{
		{"https://example.com/path?q=1", true},
		{"http://localhost:8080", true},
		{"ftp://example.com", false},
		{"https://", false},
		{"https://user:pass@example.com", false},
		{"https://example.com/#fragment", false},
		{"", false},
	} {
		if got := validURL(tc.value); got != tc.valid {
			t.Errorf("validURL(%q) = %v, want %v", tc.value, got, tc.valid)
		}
	}
}

func TestValidInterval(t *testing.T) {
	for _, tc := range []struct {
		value int64
		unit  string
		valid bool
	}{
		{7, "seconds", true},
		{10, "minutes", true},
		{2, "hours", true},
		{maxSafeIntervalValue, "seconds", true},
		{0, "seconds", false},
		{-1, "minutes", false},
		{maxSafeIntervalValue + 1, "hours", false},
		{1, "days", false},
	} {
		if got := validInterval(tc.value, tc.unit); got != tc.valid {
			t.Errorf("validInterval(%d, %q) = %v, want %v", tc.value, tc.unit, got, tc.valid)
		}
	}
}
