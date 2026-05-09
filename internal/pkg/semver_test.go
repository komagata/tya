package pkg

import "testing"

func TestParseVersion(t *testing.T) {
	cases := []struct {
		s    string
		want Version
		err  bool
	}{
		{"1.2.3", Version{Major: 1, Minor: 2, Patch: 3, Raw: "1.2.3"}, false},
		{"0.0.1", Version{Major: 0, Minor: 0, Patch: 1, Raw: "0.0.1"}, false},
		{"1.2", Version{}, true},
		{"abc", Version{}, true},
	}
	for _, c := range cases {
		got, err := ParseVersion(c.s)
		if (err != nil) != c.err {
			t.Errorf("%q: err=%v want err=%v", c.s, err, c.err)
		}
		if !c.err && (got.Major != c.want.Major || got.Minor != c.want.Minor || got.Patch != c.want.Patch) {
			t.Errorf("%q: got %+v want %+v", c.s, got, c.want)
		}
	}
}

func TestVersionCompare(t *testing.T) {
	mustV := func(s string) Version { v, _ := ParseVersion(s); return v }
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"2.0.0", "1.9.9", 1},
		{"1.2.0", "1.10.0", -1},
	}
	for _, c := range cases {
		got := mustV(c.a).Compare(mustV(c.b))
		if got != c.want {
			t.Errorf("%s vs %s: got %d want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestParseConstraint(t *testing.T) {
	mustV := func(s string) Version { v, _ := ParseVersion(s); return v }
	cases := []struct {
		s  string
		v  string
		ok bool
	}{
		{"^1.2.3", "1.2.3", true},
		{"^1.2.3", "1.5.0", true},
		{"^1.2.3", "2.0.0", false},
		{"^1.2.3", "1.2.2", false},
		{"~1.2.3", "1.2.3", true},
		{"~1.2.3", "1.2.99", true},
		{"~1.2.3", "1.3.0", false},
		{">= 1.0.0, < 2.0.0", "1.5.0", true},
		{">= 1.0.0, < 2.0.0", "2.0.0", false},
		{"1.2.3", "1.2.3", true},
		{"1.2.3", "1.2.4", false},
		{"^0.5.0", "0.5.99", true},
		{"^0.5.0", "0.6.0", false},
	}
	for _, c := range cases {
		con, err := ParseConstraint(c.s)
		if err != nil {
			t.Errorf("ParseConstraint(%q): %v", c.s, err)
			continue
		}
		got := con.Satisfies(mustV(c.v))
		if got != c.ok {
			t.Errorf("%q satisfies %s: got %v want %v", c.s, c.v, got, c.ok)
		}
	}
}
