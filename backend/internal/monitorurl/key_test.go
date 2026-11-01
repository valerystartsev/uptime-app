package monitorurl

import "testing"

func TestKey(t *testing.T) {
	for _, tc := range []struct {
		first, second string
		equal         bool
	}{
		{"https://EXAMPLE.com:443", "https://example.com/", true},
		{"http://EXAMPLE.com:80/a", "http://example.com/a", true},
		{"http://[::1]:80", "http://[::1]/", true},
		{"https://example.com/a", "https://example.com/b", false},
		{"https://example.com/?a=1", "https://example.com/?a=2", false},
		{"https://example.com/", "http://example.com/", false},
		{"https://example.com:8443/", "https://example.com/", false},
	} {
		if got := Key(tc.first) == Key(tc.second); got != tc.equal {
			t.Errorf("Key(%q) == Key(%q) = %v, want %v", tc.first, tc.second, got, tc.equal)
		}
	}
}
