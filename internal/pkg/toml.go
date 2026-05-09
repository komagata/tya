package pkg

// Minimal TOML reader/writer for the subset Tya v0.26 needs:
// scalar key=value, [section], [[array.of.tables]], string/int/bool/array/
// inline-table values, # comments. No multi-line strings, no dotted keys
// outside section headers, no datetimes.

import (
	"fmt"
	"strconv"
	"strings"
)

type TomlValue struct {
	Kind  string // "string", "int", "bool", "array", "table"
	Str   string
	Int   int64
	Bool  bool
	Array []TomlValue
	Table map[string]TomlValue
	Order []string // insertion order for tables
}

func TomlString(s string) TomlValue { return TomlValue{Kind: "string", Str: s} }
func TomlInt(n int64) TomlValue     { return TomlValue{Kind: "int", Int: n} }
func TomlBool(b bool) TomlValue     { return TomlValue{Kind: "bool", Bool: b} }
func NewTomlTable() TomlValue       { return TomlValue{Kind: "table", Table: map[string]TomlValue{}} }
func NewTomlArray() TomlValue       { return TomlValue{Kind: "array"} }

func (t *TomlValue) SetField(k string, v TomlValue) {
	if t.Table == nil {
		t.Table = map[string]TomlValue{}
	}
	if _, ok := t.Table[k]; !ok {
		t.Order = append(t.Order, k)
	}
	t.Table[k] = v
}

// ParseToml parses a TOML document and returns the root table.
func ParseToml(src string) (TomlValue, error) {
	root := NewTomlTable()
	currentPath := []string{} // empty = root
	lines := strings.Split(src, "\n")
	for li, raw := range lines {
		line := strings.TrimSpace(stripTomlComment(raw))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[[") && strings.HasSuffix(line, "]]") {
			path := splitDottedKeys(strings.TrimSpace(line[2 : len(line)-2]))
			if err := appendTable(&root, path); err != nil {
				return root, fmt.Errorf("line %d: %v", li+1, err)
			}
			currentPath = pathForArrayLast(path)
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			path := splitDottedKeys(strings.TrimSpace(line[1 : len(line)-1]))
			if err := ensureTable(&root, path); err != nil {
				return root, fmt.Errorf("line %d: %v", li+1, err)
			}
			currentPath = path
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			return root, fmt.Errorf("line %d: expected '=' in %q", li+1, line)
		}
		key := strings.TrimSpace(line[:eq])
		val, err := parseTomlValue(strings.TrimSpace(line[eq+1:]))
		if err != nil {
			return root, fmt.Errorf("line %d: %v", li+1, err)
		}
		setAtPath(&root, currentPath, key, val)
	}
	return root, nil
}

func ensureTable(root *TomlValue, path []string) error {
	if len(path) == 0 {
		return nil
	}
	first := path[0]
	rest := path[1:]
	child, ok := root.Table[first]
	if !ok {
		child = NewTomlTable()
	} else if child.Kind != "table" {
		return fmt.Errorf("%s is not a table", first)
	}
	if err := ensureTable(&child, rest); err != nil {
		return err
	}
	root.SetField(first, child)
	return nil
}

// setAtPath writes key=val into the table at path within root, materializing
// intermediate tables as needed.
func setAtPath(root *TomlValue, path []string, key string, val TomlValue) {
	if len(path) == 0 {
		root.SetField(key, val)
		return
	}
	first := path[0]
	rest := path[1:]
	child, ok := root.Table[first]
	if !ok || child.Kind != "table" {
		// If the slot is an array of tables, write into the last element.
		if ok && child.Kind == "array" && len(child.Array) > 0 && child.Array[len(child.Array)-1].Kind == "table" {
			last := child.Array[len(child.Array)-1]
			setAtPath(&last, rest, key, val)
			child.Array[len(child.Array)-1] = last
			root.Table[first] = child
			return
		}
		child = NewTomlTable()
	}
	setAtPath(&child, rest, key, val)
	root.SetField(first, child)
}

// appendTable adds a new table to the array at path.
func appendTable(root *TomlValue, path []string) error {
	if len(path) == 0 {
		return fmt.Errorf("empty array-of-tables path")
	}
	first := path[0]
	rest := path[1:]
	if len(rest) == 0 {
		arr, ok := root.Table[first]
		if !ok {
			arr = NewTomlArray()
		}
		if arr.Kind != "array" {
			return fmt.Errorf("%s is not an array of tables", first)
		}
		arr.Array = append(arr.Array, NewTomlTable())
		root.SetField(first, arr)
		return nil
	}
	child, ok := root.Table[first]
	if !ok || child.Kind != "table" {
		child = NewTomlTable()
	}
	if err := appendTable(&child, rest); err != nil {
		return err
	}
	root.SetField(first, child)
	return nil
}

// pathForArrayLast returns path components such that subsequent setAtPath
// calls write into the last element of the array at path.
func pathForArrayLast(path []string) []string {
	return path
}

func splitDottedKeys(path string) []string {
	out := []string{}
	cur := strings.Builder{}
	inQuote := false
	for i := 0; i < len(path); i++ {
		c := path[i]
		if c == '"' {
			inQuote = !inQuote
			continue
		}
		if c == '.' && !inQuote {
			out = append(out, strings.TrimSpace(cur.String()))
			cur.Reset()
			continue
		}
		cur.WriteByte(c)
	}
	if cur.Len() > 0 {
		out = append(out, strings.TrimSpace(cur.String()))
	}
	return out
}

func stripTomlComment(s string) string {
	inStr := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '"' {
			j := i - 1
			esc := false
			for j >= 0 && s[j] == '\\' {
				esc = !esc
				j--
			}
			if !esc {
				inStr = !inStr
			}
		}
		if c == '#' && !inStr {
			return s[:i]
		}
	}
	return s
}

func parseTomlValue(s string) (TomlValue, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return TomlValue{}, fmt.Errorf("empty value")
	}
	if s == "true" {
		return TomlValue{Kind: "bool", Bool: true}, nil
	}
	if s == "false" {
		return TomlValue{Kind: "bool", Bool: false}, nil
	}
	if s[0] == '"' {
		return parseTomlString(s)
	}
	if s[0] == '[' {
		return parseTomlArray(s)
	}
	if s[0] == '{' {
		return parseTomlInlineTable(s)
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return TomlValue{Kind: "int", Int: n}, nil
	}
	return TomlValue{}, fmt.Errorf("unknown value %q", s)
}

func parseTomlString(s string) (TomlValue, error) {
	if len(s) < 2 || s[0] != '"' {
		return TomlValue{}, fmt.Errorf("expected string")
	}
	out := strings.Builder{}
	i := 1
	for i < len(s) && s[i] != '"' {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case '"':
				out.WriteByte('"')
			case '\\':
				out.WriteByte('\\')
			case 'n':
				out.WriteByte('\n')
			case 't':
				out.WriteByte('\t')
			case 'r':
				out.WriteByte('\r')
			default:
				return TomlValue{}, fmt.Errorf("unknown escape \\%c", s[i+1])
			}
			i += 2
			continue
		}
		out.WriteByte(s[i])
		i++
	}
	if i >= len(s) || s[i] != '"' {
		return TomlValue{}, fmt.Errorf("unterminated string")
	}
	rest := strings.TrimSpace(s[i+1:])
	if rest != "" {
		return TomlValue{}, fmt.Errorf("trailing characters after string: %q", rest)
	}
	return TomlValue{Kind: "string", Str: out.String()}, nil
}

func parseTomlArray(s string) (TomlValue, error) {
	if len(s) < 2 || s[0] != '[' || s[len(s)-1] != ']' {
		return TomlValue{}, fmt.Errorf("expected [...] array")
	}
	body := strings.TrimSpace(s[1 : len(s)-1])
	if body == "" {
		return TomlValue{Kind: "array"}, nil
	}
	parts := splitTomlList(body)
	arr := TomlValue{Kind: "array"}
	for _, p := range parts {
		v, err := parseTomlValue(strings.TrimSpace(p))
		if err != nil {
			return arr, err
		}
		arr.Array = append(arr.Array, v)
	}
	return arr, nil
}

func parseTomlInlineTable(s string) (TomlValue, error) {
	if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
		return TomlValue{}, fmt.Errorf("expected {...} inline table")
	}
	body := strings.TrimSpace(s[1 : len(s)-1])
	t := NewTomlTable()
	if body == "" {
		return t, nil
	}
	parts := splitTomlList(body)
	for _, p := range parts {
		eq := strings.IndexByte(p, '=')
		if eq < 0 {
			return t, fmt.Errorf("expected key=value in inline table")
		}
		k := strings.TrimSpace(p[:eq])
		v, err := parseTomlValue(strings.TrimSpace(p[eq+1:]))
		if err != nil {
			return t, err
		}
		t.SetField(k, v)
	}
	return t, nil
}

// splitTomlList splits comma-separated parts at the top brace level only.
func splitTomlList(s string) []string {
	out := []string{}
	depth := 0
	inStr := false
	cur := strings.Builder{}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '"' && (i == 0 || s[i-1] != '\\') {
			inStr = !inStr
		}
		if !inStr {
			if c == '[' || c == '{' {
				depth++
			}
			if c == ']' || c == '}' {
				depth--
			}
			if c == ',' && depth == 0 {
				out = append(out, cur.String())
				cur.Reset()
				continue
			}
		}
		cur.WriteByte(c)
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

func contains(s []string, x string) bool {
	for _, v := range s {
		if v == x {
			return true
		}
	}
	return false
}

// EmitToml writes a top-level table back as TOML text. Output preserves the
// insertion order recorded in the Order field.
func EmitToml(root TomlValue) string {
	var b strings.Builder
	emitTomlScalars(&b, root)
	emitTomlSubtables(&b, root, "")
	return b.String()
}

func emitTomlScalars(b *strings.Builder, t TomlValue) {
	for _, k := range t.Order {
		v := t.Table[k]
		if v.Kind == "table" || (v.Kind == "array" && allTables(v)) {
			continue
		}
		fmt.Fprintf(b, "%s = %s\n", k, emitTomlValue(v))
	}
}

func emitTomlSubtables(b *strings.Builder, t TomlValue, prefix string) {
	for _, k := range t.Order {
		v := t.Table[k]
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		if v.Kind == "table" {
			b.WriteString("\n[")
			b.WriteString(path)
			b.WriteString("]\n")
			emitTomlScalars(b, v)
			emitTomlSubtables(b, v, path)
		} else if v.Kind == "array" && allTables(v) {
			for _, item := range v.Array {
				b.WriteString("\n[[")
				b.WriteString(path)
				b.WriteString("]]\n")
				emitTomlScalars(b, item)
				emitTomlSubtables(b, item, path)
			}
		}
	}
}

func allTables(v TomlValue) bool {
	if v.Kind != "array" || len(v.Array) == 0 {
		return false
	}
	for _, item := range v.Array {
		if item.Kind != "table" {
			return false
		}
	}
	return true
}

func emitTomlValue(v TomlValue) string {
	switch v.Kind {
	case "string":
		return strconv.Quote(v.Str)
	case "int":
		return strconv.FormatInt(v.Int, 10)
	case "bool":
		if v.Bool {
			return "true"
		}
		return "false"
	case "array":
		parts := []string{}
		for _, it := range v.Array {
			parts = append(parts, emitTomlValue(it))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case "table":
		parts := []string{}
		for _, k := range v.Order {
			parts = append(parts, k+" = "+emitTomlValue(v.Table[k]))
		}
		return "{ " + strings.Join(parts, ", ") + " }"
	}
	return ""
}
