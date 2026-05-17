package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// FindManifest walks startDir and its parents (up to 16 levels) looking
// for a tya.toml. It returns the project root directory and the full
// manifest path. When no manifest is found, it returns an error whose
// message matches the historical phrasing used by `tya install` etc.
func FindManifest(startDir string) (rootDir, manifestPath string, err error) {
	dir := startDir
	for i := 0; i < 16; i++ {
		candidate := filepath.Join(dir, ManifestName)
		if _, statErr := os.Stat(candidate); statErr == nil {
			return dir, candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", "", fmt.Errorf("no tya.toml found in current directory or any parent")
}

// Manifest is the parsed tya.toml.
type Manifest struct {
	Name        string
	Version     Version
	Description string
	Authors     []string
	License     string
	Native      Native
	Deps        map[string]Dependency // name -> dependency entry
	DevDeps     map[string]Dependency
	DepOrder    []string // insertion order for Deps
	DevOrder    []string
	Tasks       map[string]Task // task name -> task definition
	TaskOrder   []string        // insertion order for Tasks
	Tools       map[string]string
	ToolOrder   []string

	Path string // path to the manifest file (for relative path deps)
}

type Native struct {
	Sources     []string
	Headers     []string
	IncludeDirs []string
	PkgConfig   []string
	CFlags      []string
	LDFlags     []string
	Functions   map[string]NativeFunction
	FuncOrder   []string
}

type NativeFunction struct {
	Name   string
	Symbol string
	Arity  int
}

type Dependency struct {
	Name       string
	Constraint Constraint // empty when source != "registry"
	Source     string     // "version", "git", "path"
	Git        string     // URL
	Tag        string
	Branch     string
	Rev        string
	PathRef    string // for path source
}

// Task is a single entry under [tasks] in tya.toml. A task is one of:
//
//	Kind == "string"   single shell command (`name = "..."`)
//	Kind == "array"    sequence of commands, sequential, stop on first failure
//	Kind == "parallel" table form: `[tasks.name] cmds = [...]; parallel = true`
//	                   runs every entry concurrently, waits for all, returns
//	                   the first non-zero exit code
type Task struct {
	Name      string
	Kind      string            // "string" | "array" | "parallel"
	String    string            // populated when Kind == "string"
	Array     []string          // populated when Kind == "array" or "parallel"
	DependsOn []string          // dependency task names, in manifest order
	Env       map[string]string // per-task environment overrides
	Watch     []string          // optional watch globs
	Ignore    []string          // optional additional ignore globs
}

func ReadManifest(path string) (*Manifest, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	tv, err := ParseToml(string(src))
	if err != nil {
		return nil, err
	}
	m := &Manifest{Path: path, Deps: map[string]Dependency{}, DevDeps: map[string]Dependency{}, Tasks: map[string]Task{}, Tools: map[string]string{}}
	if v, ok := tv.Table["name"]; ok && v.Kind == "string" {
		m.Name = v.Str
	}
	if v, ok := tv.Table["version"]; ok && v.Kind == "string" {
		ver, err := ParseVersion(v.Str)
		if err != nil {
			return nil, err
		}
		m.Version = ver
	}
	if v, ok := tv.Table["description"]; ok && v.Kind == "string" {
		m.Description = v.Str
	}
	if v, ok := tv.Table["license"]; ok && v.Kind == "string" {
		m.License = v.Str
	}
	if v, ok := tv.Table["authors"]; ok && v.Kind == "array" {
		for _, a := range v.Array {
			if a.Kind == "string" {
				m.Authors = append(m.Authors, a.Str)
			}
		}
	}
	if deps, ok := tv.Table["dependencies"]; ok && deps.Kind == "table" {
		for _, name := range deps.Order {
			d, err := readDep(name, deps.Table[name])
			if err != nil {
				return nil, fmt.Errorf("dependencies.%s: %v", name, err)
			}
			m.Deps[name] = d
			m.DepOrder = append(m.DepOrder, name)
		}
	}
	if dev, ok := tv.Table["dev-dependencies"]; ok && dev.Kind == "table" {
		for _, name := range dev.Order {
			d, err := readDep(name, dev.Table[name])
			if err != nil {
				return nil, fmt.Errorf("dev-dependencies.%s: %v", name, err)
			}
			m.DevDeps[name] = d
			m.DevOrder = append(m.DevOrder, name)
		}
	}
	if tasks, ok := tv.Table["tasks"]; ok && tasks.Kind == "table" {
		for _, name := range tasks.Order {
			t, err := readTask(name, tasks.Table[name])
			if err != nil {
				return nil, fmt.Errorf("tasks.%s: %v", name, err)
			}
			m.Tasks[name] = t
			m.TaskOrder = append(m.TaskOrder, name)
		}
	}
	if tools, ok := tv.Table["tools"]; ok && tools.Kind == "table" {
		for _, name := range tools.Order {
			v := tools.Table[name]
			if v.Kind != "string" {
				return nil, fmt.Errorf("tools.%s: expected string", name)
			}
			m.Tools[name] = v.Str
			m.ToolOrder = append(m.ToolOrder, name)
		}
	}
	if native, ok := tv.Table["native"]; ok && native.Kind == "table" {
		n, err := readNative(native)
		if err != nil {
			return nil, fmt.Errorf("native: %v", err)
		}
		m.Native = n
	}
	if m.Name == "" {
		return nil, fmt.Errorf("tya.toml: missing name")
	}
	if m.Version.Raw == "" {
		return nil, fmt.Errorf("tya.toml: missing version")
	}
	return m, nil
}

func readNative(v TomlValue) (Native, error) {
	n := Native{Functions: map[string]NativeFunction{}}
	var err error
	if n.Sources, err = readStringArrayField(v, "sources"); err != nil {
		return n, err
	}
	if n.Headers, err = readStringArrayField(v, "headers"); err != nil {
		return n, err
	}
	if n.IncludeDirs, err = readStringArrayField(v, "include_dirs"); err != nil {
		return n, err
	}
	if n.PkgConfig, err = readStringArrayField(v, "pkg_config"); err != nil {
		return n, err
	}
	if n.CFlags, err = readStringArrayField(v, "cflags"); err != nil {
		return n, err
	}
	if n.LDFlags, err = readStringArrayField(v, "ldflags"); err != nil {
		return n, err
	}
	if funcs, ok := v.Table["functions"]; ok {
		if funcs.Kind != "table" {
			return n, fmt.Errorf("functions: expected table")
		}
		for _, name := range funcs.Order {
			fv := funcs.Table[name]
			if fv.Kind != "table" {
				return n, fmt.Errorf("functions.%s: expected table", name)
			}
			sym, ok := fv.Table["symbol"]
			if !ok || sym.Kind != "string" || sym.Str == "" {
				return n, fmt.Errorf("functions.%s.symbol: expected string", name)
			}
			arity, ok := fv.Table["arity"]
			if !ok || arity.Kind != "int" {
				return n, fmt.Errorf("functions.%s.arity: expected int", name)
			}
			if arity.Int < 0 || arity.Int > 4 {
				return n, fmt.Errorf("functions.%s.arity: expected 0..4", name)
			}
			n.Functions[name] = NativeFunction{Name: name, Symbol: sym.Str, Arity: int(arity.Int)}
			n.FuncOrder = append(n.FuncOrder, name)
		}
	}
	return n, nil
}

func readStringArrayField(v TomlValue, name string) ([]string, error) {
	field, ok := v.Table[name]
	if !ok {
		return nil, nil
	}
	if field.Kind != "array" {
		return nil, fmt.Errorf("%s: expected array of strings", name)
	}
	out := []string{}
	for i, item := range field.Array {
		if item.Kind != "string" {
			return nil, fmt.Errorf("%s[%d]: expected string", name, i)
		}
		out = append(out, item.Str)
	}
	return out, nil
}

func readTask(name string, v TomlValue) (Task, error) {
	t := Task{Name: name}
	switch v.Kind {
	case "string":
		t.Kind = "string"
		t.String = v.Str
		return t, nil
	case "array":
		t.Kind = "array"
		for i, item := range v.Array {
			if item.Kind != "string" {
				return t, fmt.Errorf("array element #%d: expected string, got %s", i+1, item.Kind)
			}
			t.Array = append(t.Array, item.Str)
		}
		return t, nil
	case "table":
		// table form: must carry `cmds = [...]`; `parallel = true`
		// flips Kind to "parallel". Without `parallel = true`, the
		// table form behaves like an array form.
		cmds, ok := v.Table["cmds"]
		if !ok || cmds.Kind != "array" {
			return t, fmt.Errorf("expected `cmds = [...]` in table form")
		}
		for i, item := range cmds.Array {
			if item.Kind != "string" {
				return t, fmt.Errorf("cmds element #%d: expected string, got %s", i+1, item.Kind)
			}
			t.Array = append(t.Array, item.Str)
		}
		t.Kind = "array"
		if p, ok := v.Table["parallel"]; ok && p.Kind == "bool" && p.Bool {
			t.Kind = "parallel"
		}
		var err error
		if t.DependsOn, err = readStringArrayTaskField(v, "depends_on"); err != nil {
			return t, err
		}
		if t.Watch, err = readStringArrayTaskField(v, "watch"); err != nil {
			return t, err
		}
		if t.Ignore, err = readStringArrayTaskField(v, "ignore"); err != nil {
			return t, err
		}
		if t.Env, err = readTaskEnv(v); err != nil {
			return t, fmt.Errorf("[TYA-E0906] %v", err)
		}
		return t, nil
	default:
		return t, fmt.Errorf("expected string, array of strings, or table form, got %s", v.Kind)
	}
}

func readStringArrayTaskField(v TomlValue, name string) ([]string, error) {
	field, ok := v.Table[name]
	if !ok {
		return nil, nil
	}
	if field.Kind != "array" {
		return nil, fmt.Errorf("%s: expected array of strings", name)
	}
	out := []string{}
	for i, item := range field.Array {
		if item.Kind != "string" {
			return nil, fmt.Errorf("%s[%d]: expected string, got %s", name, i, item.Kind)
		}
		out = append(out, item.Str)
	}
	return out, nil
}

func readTaskEnv(v TomlValue) (map[string]string, error) {
	field, ok := v.Table["env"]
	if !ok {
		return nil, nil
	}
	if field.Kind != "table" {
		return nil, fmt.Errorf("env: expected table")
	}
	out := map[string]string{}
	for _, key := range field.Order {
		value := field.Table[key]
		if value.Kind != "string" {
			return nil, fmt.Errorf("env.%s: expected string", key)
		}
		out[key] = value.Str
	}
	return out, nil
}

func readDep(name string, v TomlValue) (Dependency, error) {
	d := Dependency{Name: name}
	if v.Kind == "string" {
		c, err := ParseConstraint(v.Str)
		if err != nil {
			return d, err
		}
		d.Constraint = c
		d.Source = "version"
		return d, nil
	}
	if v.Kind != "table" {
		return d, fmt.Errorf("expected string or table")
	}
	if g, ok := v.Table["git"]; ok && g.Kind == "string" {
		d.Source = "git"
		d.Git = g.Str
		if tag, ok := v.Table["tag"]; ok && tag.Kind == "string" {
			d.Tag = tag.Str
		}
		if br, ok := v.Table["branch"]; ok && br.Kind == "string" {
			d.Branch = br.Str
		}
		if rv, ok := v.Table["rev"]; ok && rv.Kind == "string" {
			d.Rev = rv.Str
		}
		if d.Tag == "" && d.Branch == "" && d.Rev == "" {
			return d, fmt.Errorf("git dependency requires one of tag/branch/rev")
		}
	} else if p, ok := v.Table["path"]; ok && p.Kind == "string" {
		d.Source = "path"
		d.PathRef = p.Str
	} else if vr, ok := v.Table["version"]; ok && vr.Kind == "string" {
		c, err := ParseConstraint(vr.Str)
		if err != nil {
			return d, err
		}
		d.Constraint = c
		d.Source = "version"
	} else {
		return d, fmt.Errorf("unknown dependency form")
	}
	if vr, ok := v.Table["version"]; ok && vr.Kind == "string" && d.Source != "version" {
		c, err := ParseConstraint(vr.Str)
		if err != nil {
			return d, err
		}
		d.Constraint = c
	}
	return d, nil
}

func WriteManifest(m *Manifest) error {
	t := NewTomlTable()
	t.SetField("name", TomlString(m.Name))
	t.SetField("version", TomlString(m.Version.String()))
	if m.Description != "" {
		t.SetField("description", TomlString(m.Description))
	}
	if m.License != "" {
		t.SetField("license", TomlString(m.License))
	}
	if len(m.Authors) > 0 {
		arr := NewTomlArray()
		for _, a := range m.Authors {
			arr.Array = append(arr.Array, TomlString(a))
		}
		t.SetField("authors", arr)
	}
	if len(m.DepOrder) > 0 {
		dep := NewTomlTable()
		for _, k := range m.DepOrder {
			dep.SetField(k, depToToml(m.Deps[k]))
		}
		t.SetField("dependencies", dep)
	}
	if len(m.DevOrder) > 0 {
		dep := NewTomlTable()
		for _, k := range m.DevOrder {
			dep.SetField(k, depToToml(m.DevDeps[k]))
		}
		t.SetField("dev-dependencies", dep)
	}
	if len(m.TaskOrder) > 0 {
		tasks := NewTomlTable()
		for _, k := range m.TaskOrder {
			tasks.SetField(k, taskToToml(m.Tasks[k]))
		}
		t.SetField("tasks", tasks)
	}
	if len(m.ToolOrder) > 0 {
		tools := NewTomlTable()
		for _, k := range m.ToolOrder {
			tools.SetField(k, TomlString(m.Tools[k]))
		}
		t.SetField("tools", tools)
	}
	if nativeHasFields(m.Native) {
		t.SetField("native", nativeToToml(m.Native))
	}
	return os.WriteFile(m.Path, []byte(EmitToml(t)), 0644)
}

func nativeHasFields(n Native) bool {
	return len(n.Sources) > 0 || len(n.Headers) > 0 || len(n.IncludeDirs) > 0 ||
		len(n.PkgConfig) > 0 || len(n.CFlags) > 0 || len(n.LDFlags) > 0 || len(n.FuncOrder) > 0
}

func nativeToToml(n Native) TomlValue {
	t := NewTomlTable()
	setStringArray := func(name string, values []string) {
		if len(values) == 0 {
			return
		}
		arr := NewTomlArray()
		for _, value := range values {
			arr.Array = append(arr.Array, TomlString(value))
		}
		t.SetField(name, arr)
	}
	setStringArray("sources", n.Sources)
	setStringArray("headers", n.Headers)
	setStringArray("include_dirs", n.IncludeDirs)
	setStringArray("pkg_config", n.PkgConfig)
	setStringArray("cflags", n.CFlags)
	setStringArray("ldflags", n.LDFlags)
	if len(n.FuncOrder) > 0 {
		funcs := NewTomlTable()
		for _, name := range n.FuncOrder {
			fn := n.Functions[name]
			ft := NewTomlTable()
			ft.SetField("symbol", TomlString(fn.Symbol))
			ft.SetField("arity", TomlInt(int64(fn.Arity)))
			funcs.SetField(name, ft)
		}
		t.SetField("functions", funcs)
	}
	return t
}

func taskToToml(t Task) TomlValue {
	if len(t.DependsOn) > 0 || len(t.Env) > 0 || len(t.Watch) > 0 || len(t.Ignore) > 0 || t.Kind == "parallel" {
		tbl := NewTomlTable()
		arr := NewTomlArray()
		for _, cmd := range t.Array {
			arr.Array = append(arr.Array, TomlString(cmd))
		}
		if t.Kind == "string" {
			arr.Array = append(arr.Array, TomlString(t.String))
		}
		tbl.SetField("cmds", arr)
		if t.Kind == "parallel" {
			tbl.SetField("parallel", TomlBool(true))
		}
		if len(t.DependsOn) > 0 {
			tbl.SetField("depends_on", stringsToTomlArray(t.DependsOn))
		}
		if len(t.Env) > 0 {
			env := NewTomlTable()
			keys := make([]string, 0, len(t.Env))
			for key := range t.Env {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				env.SetField(key, TomlString(t.Env[key]))
			}
			tbl.SetField("env", env)
		}
		if len(t.Watch) > 0 {
			tbl.SetField("watch", stringsToTomlArray(t.Watch))
		}
		if len(t.Ignore) > 0 {
			tbl.SetField("ignore", stringsToTomlArray(t.Ignore))
		}
		return tbl
	}
	if t.Kind == "array" {
		arr := NewTomlArray()
		for _, cmd := range t.Array {
			arr.Array = append(arr.Array, TomlString(cmd))
		}
		return arr
	}
	return TomlString(t.String)
}

func stringsToTomlArray(values []string) TomlValue {
	arr := NewTomlArray()
	for _, value := range values {
		arr.Array = append(arr.Array, TomlString(value))
	}
	return arr
}

func depToToml(d Dependency) TomlValue {
	switch d.Source {
	case "version":
		return TomlString(d.Constraint.Raw)
	case "git":
		t := NewTomlTable()
		t.SetField("git", TomlString(d.Git))
		if d.Tag != "" {
			t.SetField("tag", TomlString(d.Tag))
		}
		if d.Branch != "" {
			t.SetField("branch", TomlString(d.Branch))
		}
		if d.Rev != "" {
			t.SetField("rev", TomlString(d.Rev))
		}
		if d.Constraint.Raw != "" {
			t.SetField("version", TomlString(d.Constraint.Raw))
		}
		return t
	case "path":
		t := NewTomlTable()
		t.SetField("path", TomlString(d.PathRef))
		return t
	}
	return NewTomlTable()
}
