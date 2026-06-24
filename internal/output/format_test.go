package output

import "testing"

func TestFormatBytes(t *testing.T) {
	cases := map[uint64]string{
		0:               "0 B",
		512:             "512 B",
		1024:            "1.0 KB",
		202006528:       "192.6 MB",
		1610612736:      "1.5 GB",
		109951162777600: "100.0 TB",
	}
	for n, want := range cases {
		if got := formatBytes(n); got != want {
			t.Errorf("formatBytes(%d) = %q, want %q", n, got, want)
		}
	}
}
