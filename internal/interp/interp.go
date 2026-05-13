package interp

// FindExprEnd returns the byte offset of the closing brace for an
// interpolation expression that starts at open. Braces inside nested string
// literals are ignored, and braces outside strings are balanced.
func FindExprEnd(s string, open int) int {
	if open < 0 || open >= len(s) || s[open] != '{' {
		return -1
	}
	depth := 1
	for i := open + 1; i < len(s); i++ {
		switch s[i] {
		case '"':
			if n := skipString(s, i); n >= 0 {
				i = n
				continue
			}
			return -1
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func LooksLikeDictExprStart(s string, open int) bool {
	if open+2 >= len(s) || s[open] != '{' || s[open+1] != '{' {
		return false
	}
	if s[open+2] == '{' {
		return true
	}
	if s[open+2] != '"' {
		return false
	}
	end := skipString(s, open+2)
	return end >= 0 && end+1 < len(s) && s[end+1] == ':'
}

func skipString(s string, start int) int {
	if start+2 < len(s) && s[start:start+3] == `"""` {
		for i := start + 3; i+2 < len(s); i++ {
			if s[i:i+3] == `"""` {
				return i + 2
			}
		}
		return -1
	}
	escaped := false
	for i := start + 1; i < len(s); i++ {
		if escaped {
			escaped = false
			continue
		}
		if s[i] == '\\' {
			escaped = true
			continue
		}
		if s[i] == '"' {
			return i
		}
	}
	return -1
}
