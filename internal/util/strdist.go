// Package util collects small helpers shared across the
// compiler. v0.54 introduces a Levenshtein-based did-you-mean
// suggester used by parser / runner / checker diagnostics.
package util

import "sort"

// Levenshtein returns the edit distance between a and b in O(len(a)*len(b))
// time. The implementation is byte-oriented; tya identifiers are ASCII
// so this is exact, and string literals get a conservative result.
func Levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := 0; j <= len(b); j++ {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			ins := cur[j-1] + 1
			del := prev[j] + 1
			sub := prev[j-1] + cost
			cur[j] = min3(ins, del, sub)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

func min3(a, b, c int) int {
	if a <= b && a <= c {
		return a
	}
	if b <= c {
		return b
	}
	return c
}

// Suggest returns up to max candidates from `pool` whose Levenshtein
// distance to `name` is no greater than maxDist. Candidates are sorted
// by distance (ascending) then by their position in pool (stable).
// `name` is excluded from the output even if it appears verbatim in
// pool — the caller usually feeds candidates that include the
// queried name's own scope.
func Suggest(name string, pool []string, maxDist, max int) []string {
	if name == "" || maxDist < 1 || max < 1 {
		return nil
	}
	type cand struct {
		s    string
		d    int
		rank int
	}
	out := []cand{}
	for i, p := range pool {
		if p == name {
			continue
		}
		d := Levenshtein(name, p)
		if d > maxDist {
			continue
		}
		out = append(out, cand{s: p, d: d, rank: i})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].d != out[j].d {
			return out[i].d < out[j].d
		}
		return out[i].rank < out[j].rank
	})
	if len(out) > max {
		out = out[:max]
	}
	res := make([]string, 0, len(out))
	for _, c := range out {
		res = append(res, c.s)
	}
	return res
}
