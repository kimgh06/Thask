package telemetry

import "testing"

func TestRedactPatterns(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"argv token space", "thask --token thsk_secret auth", "thask --token thsk_*** auth"},
		{"argv token equals", "--token=thsk_secret", "--token=thsk_***"},
		{"url credential", "https://kim:hunter2@example.com/api", "https://***:***@example.com/api"},
		{"bearer header", "Authorization: Bearer thsk_actualkey123", "Authorization: Bearer thsk_***"},
		{"naked thsk token", "found key thsk_abcdEF123", "found key thsk_***"},
		{"jwt", "header eyJhbGciOiJIUzI1NiJ9.payload here", "header eyJ*** here"},
		{"plain user content untouched", "Refactor auth flow #1234", "Refactor auth flow #1234"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Redact(c.in)
			if got != c.want {
				t.Errorf("Redact(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestRedactHeaderMasksCookie(t *testing.T) {
	in := "Cookie: session=abc123\nSet-Cookie: refresh=xyz\nOther: keep"
	got := RedactHeader(in)
	if got != "Cookie: ***\nSet-Cookie: ***\nOther: keep" {
		t.Errorf("RedactHeader did not mask cookies, got: %q", got)
	}
}

func TestBucketBytes(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{0, "<1k"},
		{1023, "<1k"},
		{1024, "<10k"},
		{9 * 1024, "<10k"},
		{10 * 1024, "<100k"},
		{99 * 1024, "<100k"},
		{100 * 1024, ">=100k"},
		{10 * 1024 * 1024, ">=100k"},
	}
	for _, c := range cases {
		if got := BucketBytes(c.n); got != c.want {
			t.Errorf("BucketBytes(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}
