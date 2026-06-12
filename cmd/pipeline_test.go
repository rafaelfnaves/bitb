package cmd

import "testing"

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		secs int
		want string
	}{
		{0, "0s"},
		{45, "45s"},
		{60, "1m0s"},
		{90, "1m30s"},
		{330, "5m30s"},
		{3600, "1h0m"},
		{3661, "1h1m"},
		{7200, "2h0m"},
	}
	for _, tc := range cases {
		if got := formatDuration(tc.secs); got != tc.want {
			t.Errorf("formatDuration(%d) = %q, want %q", tc.secs, got, tc.want)
		}
	}
}
