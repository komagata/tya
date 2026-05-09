// Package pkg implements the v0.26 package manager: tya.toml manifest,
// tya.lock lockfile, version resolution, and source fetchers.
package pkg

import (
	"fmt"
	"strconv"
	"strings"
)

// Version is a SemVer 2.0 version (no pre-release/build metadata in v0.26).
type Version struct {
	Major, Minor, Patch int
	Raw                 string
}

func ParseVersion(s string) (Version, error) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid version %q: need major.minor.patch", s)
	}
	v := Version{Raw: s}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return Version{}, fmt.Errorf("invalid version %q", s)
		}
		switch i {
		case 0:
			v.Major = n
		case 1:
			v.Minor = n
		case 2:
			v.Patch = n
		}
	}
	return v, nil
}

func (v Version) String() string {
	if v.Raw != "" {
		return v.Raw
	}
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compare returns -1, 0, +1.
func (v Version) Compare(o Version) int {
	if v.Major != o.Major {
		if v.Major < o.Major {
			return -1
		}
		return 1
	}
	if v.Minor != o.Minor {
		if v.Minor < o.Minor {
			return -1
		}
		return 1
	}
	if v.Patch != o.Patch {
		if v.Patch < o.Patch {
			return -1
		}
		return 1
	}
	return 0
}

// Constraint is a conjunction of version range terms.
type Constraint struct {
	Terms []Term
	Raw   string
}

type Term struct {
	Op      string // ">=", ">", "<=", "<", "="
	Version Version
}

func ParseConstraint(s string) (Constraint, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Constraint{}, fmt.Errorf("empty constraint")
	}
	c := Constraint{Raw: s}
	if strings.HasPrefix(s, "^") {
		v, err := ParseVersion(s[1:])
		if err != nil {
			return c, err
		}
		c.Terms = append(c.Terms, Term{Op: ">=", Version: v})
		var upper Version
		if v.Major > 0 {
			upper = Version{Major: v.Major + 1}
		} else if v.Minor > 0 {
			upper = Version{Major: 0, Minor: v.Minor + 1}
		} else {
			upper = Version{Major: 0, Minor: 0, Patch: v.Patch + 1}
		}
		c.Terms = append(c.Terms, Term{Op: "<", Version: upper})
		return c, nil
	}
	if strings.HasPrefix(s, "~") {
		v, err := ParseVersion(s[1:])
		if err != nil {
			return c, err
		}
		c.Terms = append(c.Terms, Term{Op: ">=", Version: v})
		c.Terms = append(c.Terms, Term{Op: "<", Version: Version{Major: v.Major, Minor: v.Minor + 1}})
		return c, nil
	}
	parts := strings.Split(s, ",")
	for _, p := range parts {
		t, err := parseTerm(strings.TrimSpace(p))
		if err != nil {
			return c, err
		}
		c.Terms = append(c.Terms, t)
	}
	return c, nil
}

func parseTerm(s string) (Term, error) {
	for _, op := range []string{">=", "<=", ">", "<", "="} {
		if strings.HasPrefix(s, op) {
			v, err := ParseVersion(strings.TrimSpace(s[len(op):]))
			if err != nil {
				return Term{}, err
			}
			return Term{Op: op, Version: v}, nil
		}
	}
	v, err := ParseVersion(s)
	if err != nil {
		return Term{}, err
	}
	return Term{Op: "=", Version: v}, nil
}

func (c Constraint) Satisfies(v Version) bool {
	for _, t := range c.Terms {
		cmp := v.Compare(t.Version)
		switch t.Op {
		case "=":
			if cmp != 0 {
				return false
			}
		case ">=":
			if cmp < 0 {
				return false
			}
		case ">":
			if cmp <= 0 {
				return false
			}
		case "<=":
			if cmp > 0 {
				return false
			}
		case "<":
			if cmp >= 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func (c Constraint) String() string {
	if c.Raw != "" {
		return c.Raw
	}
	parts := []string{}
	for _, t := range c.Terms {
		parts = append(parts, t.Op+t.Version.String())
	}
	return strings.Join(parts, ", ")
}
