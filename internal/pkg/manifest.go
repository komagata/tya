package pkg

import (
	"fmt"
	"os"
	"path/filepath"
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
	Deps        map[string]Dependency // name -> dependency entry
	DevDeps     map[string]Dependency
	DepOrder    []string // insertion order for Deps
	DevOrder    []string
	Tasks       map[string]Task // task name -> task definition
	TaskOrder   []string        // insertion order for Tasks

	Path string // path to the manifest file (for relative path deps)
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
	Name   string
	Kind   string   // "string" | "array" | "parallel"
	String string   // populated when Kind == "string"
	Array  []string // populated when Kind == "array" or "parallel"
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
	m := &Manifest{Path: path, Deps: map[string]Dependency{}, DevDeps: map[string]Dependency{}, Tasks: map[string]Task{}}
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
	if m.Name == "" {
		return nil, fmt.Errorf("tya.toml: missing name")
	}
	if m.Version.Raw == "" {
		return nil, fmt.Errorf("tya.toml: missing version")
	}
	return m, nil
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
		return t, nil
	default:
		return t, fmt.Errorf("expected string, array of strings, or table form, got %s", v.Kind)
	}
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
	return os.WriteFile(m.Path, []byte(EmitToml(t)), 0644)
}

func taskToToml(t Task) TomlValue {
	switch t.Kind {
	case "array":
		arr := NewTomlArray()
		for _, cmd := range t.Array {
			arr.Array = append(arr.Array, TomlString(cmd))
		}
		return arr
	case "parallel":
		tbl := NewTomlTable()
		arr := NewTomlArray()
		for _, cmd := range t.Array {
			arr.Array = append(arr.Array, TomlString(cmd))
		}
		tbl.SetField("cmds", arr)
		tbl.SetField("parallel", TomlBool(true))
		return tbl
	}
	return TomlString(t.String)
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
