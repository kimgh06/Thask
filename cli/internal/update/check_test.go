package update

import "testing"

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"0.5.4", "0.5.3", true},
		{"0.5.4", "0.5.4", false},
		{"0.5.3", "0.5.4", false},
		{"v0.5.4", "v0.5.3", true},
		{"1.0.0", "0.9.9", true},
		{"0.10.0", "0.9.9", true},
		{"", "0.5.3", false},
		{"0.5.4", "dev", false},
		{"0.5.4", "", false},
	}
	for _, c := range cases {
		got := isNewer(c.latest, c.current)
		if got != c.want {
			t.Errorf("isNewer(%q, %q) = %v; want %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestParseSemver(t *testing.T) {
	cases := []struct {
		v    string
		want int64
	}{
		{"0.5.3", 5_000_003},
		{"0.5.4", 5_000_004},
		{"1.0.0", 1_000_000_000_000},
		{"0.10.0", 10_000_000},
		{"0.5.1000", 5_001_000},          // patch ≥ 1000 still parses correctly
		{"1.2.3-beta", 1_000_002_000_003}, // pre-release suffix stripped
		{"1.2.3+build", 1_000_002_000_003},
	}
	for _, c := range cases {
		got := parseSemver(c.v)
		if got != c.want {
			t.Errorf("parseSemver(%q) = %d; want %d", c.v, got, c.want)
		}
	}
}

func TestIsNewerEdgeCases(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"0.5.1000", "0.6.0", false}, // 0.6.0 wins over 0.5.<huge>
		{"0.6.0", "0.5.1000", true},
		{"1.2.4-beta", "1.2.3", true}, // pre-release of next patch still newer
		{"1.2.3", "1.2.3-beta", false},
	}
	for _, c := range cases {
		got := isNewer(c.latest, c.current)
		if got != c.want {
			t.Errorf("isNewer(%q, %q) = %v; want %v", c.latest, c.current, got, c.want)
		}
	}
}
