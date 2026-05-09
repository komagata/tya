package pkg

import (
	"fmt"
	"os"
)

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

func ReadManifest(path string) (*Manifest, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	tv, err := ParseToml(string(src))
	if err != nil {
		return nil, err
	}
	m := &Manifest{Path: path, Deps: map[string]Dependency{}, DevDeps: map[string]Dependency{}}
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
	if m.Name == "" {
		return nil, fmt.Errorf("tya.toml: missing name")
	}
	if m.Version.Raw == "" {
		return nil, fmt.Errorf("tya.toml: missing version")
	}
	return m, nil
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
	return os.WriteFile(m.Path, []byte(EmitToml(t)), 0644)
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
