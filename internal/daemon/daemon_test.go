package daemon

import "testing"

func TestNormalizeURL(t *testing.T) {
	base := "http://127.0.0.1:1234/"

	cases := []struct {
		in   string
		want string
	}{
		{"", base},
		{"/", "http://127.0.0.1:1234/"},
		{"/yolo", "http://127.0.0.1:1234/yolo"},
		{"yolo", "http://127.0.0.1:1234/yolo"},
		{"http://example.com/x", "http://example.com/x"},
		{"https://example.com/x", "https://example.com/x"},
	}

	for _, tc := range cases {
		if got := normalizeURL(base, tc.in); got != tc.want {
			t.Fatalf("normalizeURL(%q,%q)=%q want %q", base, tc.in, got, tc.want)
		}
	}
}
