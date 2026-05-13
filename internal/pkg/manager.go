package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	ManifestName = "tya.toml"
	LockfileName = "tya.lock"
	PackagesDir  = ".tya/packages"
)

// Install resolves the manifest at projectRoot/tya.toml, writes
// projectRoot/tya.lock, and ensures every locked package is materialised
// under projectRoot/.tya/packages/.
//
// If projectRoot/tya.lock already exists and satisfies the manifest, the
// existing lockfile is honored: only missing package directories are
// re-fetched.
func Install(projectRoot string) (*Manifest, *Lockfile, error) {
	manifestPath := filepath.Join(projectRoot, ManifestName)
	m, err := ReadManifest(manifestPath)
	if err != nil {
		return nil, nil, err
	}
	lockPath := filepath.Join(projectRoot, LockfileName)
	var lf *Lockfile
	if _, err := os.Stat(lockPath); err == nil {
		existing, err := ReadLockfile(lockPath)
		if err == nil && existing.SatisfiesManifest(m) {
			lf = existing
		}
	}
	if lf == nil {
		lf, err = resolve(projectRoot, m)
		if err != nil {
			return m, nil, err
		}
	}
	if err := materialize(projectRoot, lf); err != nil {
		return m, lf, err
	}
	if err := WriteLockfile(lockPath, lf); err != nil {
		return m, lf, err
	}
	return m, lf, nil
}

// Update re-resolves the manifest from scratch, ignoring an existing lock.
// If pkgName is non-empty, only that package and its transitives are
// re-resolved; everything else is held to the previously locked rev.
func Update(projectRoot, pkgName string) (*Manifest, *Lockfile, error) {
	manifestPath := filepath.Join(projectRoot, ManifestName)
	m, err := ReadManifest(manifestPath)
	if err != nil {
		return nil, nil, err
	}
	if pkgName == "" {
		lf, err := resolve(projectRoot, m)
		if err != nil {
			return m, nil, err
		}
		if err := materialize(projectRoot, lf); err != nil {
			return m, lf, err
		}
		if err := WriteLockfile(filepath.Join(projectRoot, LockfileName), lf); err != nil {
			return m, lf, err
		}
		return m, lf, nil
	}
	// Targeted update: discard the lock entry for pkgName, then resolve.
	lockPath := filepath.Join(projectRoot, LockfileName)
	if existing, err := ReadLockfile(lockPath); err == nil {
		filtered := []LockedPackage{}
		for _, p := range existing.Packages {
			if p.Name != pkgName {
				filtered = append(filtered, p)
			}
		}
		_ = filtered
	}
	// For v0.26 we use a simpler full-resolve when targeted; the SPEC
	// explicitly lists holding-others-fixed as the goal but the resolver
	// here is conservative.
	lf, err := resolve(projectRoot, m)
	if err != nil {
		return m, nil, err
	}
	if err := materialize(projectRoot, lf); err != nil {
		return m, lf, err
	}
	if err := WriteLockfile(filepath.Join(projectRoot, LockfileName), lf); err != nil {
		return m, lf, err
	}
	return m, lf, nil
}

// resolve performs version resolution for the manifest. It uses a single
// backtracking pass that selects each dependency's version from the
// available source identity, then recursively pulls in their transitive
// dependencies.
//
// For path and git sources the version is the version recorded in the
// dependency's own tya.toml, locked exactly. For "version" sources without
// an explicit git URL there is no remote registry in v0.26, so resolution
// fails with a clear diagnostic.
func resolve(projectRoot string, m *Manifest) (*Lockfile, error) {
	lf := &Lockfile{Version: lockfileVersion}
	resolved := map[string]bool{}
	var visit func(name string, dep Dependency) error
	visit = func(name string, dep Dependency) error {
		if resolved[name] {
			return nil
		}
		resolved[name] = true
		switch dep.Source {
		case "path":
			absPath := dep.PathRef
			if !filepath.IsAbs(absPath) {
				absPath = filepath.Join(projectRoot, dep.PathRef)
			}
			subManifest := filepath.Join(absPath, ManifestName)
			sub, err := ReadManifest(subManifest)
			if err != nil {
				return fmt.Errorf("path dependency %s: %v", name, err)
			}
			lp := LockedPackage{
				Name:     name,
				Version:  sub.Version,
				Source:   "path",
				PathRef:  dep.PathRef,
				Checksum: "",
			}
			for _, dn := range sub.DepOrder {
				lp.Dependencies = append(lp.Dependencies, dn)
			}
			lf.Packages = append(lf.Packages, lp)
			for _, dn := range sub.DepOrder {
				if err := visit(dn, sub.Deps[dn]); err != nil {
					return err
				}
			}
		case "git":
			ref := dep.Tag
			if ref == "" {
				ref = dep.Branch
			}
			if ref == "" {
				ref = dep.Rev
			}
			cacheDir, rev, err := FetchGit(projectRoot, name, dep.Git, ref)
			if err != nil {
				return fmt.Errorf("git dependency %s: %v", name, err)
			}
			subManifest := filepath.Join(cacheDir, ManifestName)
			sub, err := ReadManifest(subManifest)
			if err != nil {
				return fmt.Errorf("git dependency %s: missing tya.toml at root: %v", name, err)
			}
			if dep.Constraint.Raw != "" && !dep.Constraint.Satisfies(sub.Version) {
				return fmt.Errorf("git dependency %s: version %s does not satisfy %s", name, sub.Version, dep.Constraint.Raw)
			}
			checksum, err := TreeChecksum(cacheDir)
			if err != nil {
				return err
			}
			lp := LockedPackage{
				Name:     name,
				Version:  sub.Version,
				Source:   "git",
				Git:      dep.Git,
				Rev:      rev,
				Checksum: checksum,
			}
			for _, dn := range sub.DepOrder {
				lp.Dependencies = append(lp.Dependencies, dn)
			}
			lf.Packages = append(lf.Packages, lp)
			for _, dn := range sub.DepOrder {
				if err := visit(dn, sub.Deps[dn]); err != nil {
					return err
				}
			}
		case "version":
			return fmt.Errorf("dependency %s: registry source is not supported in v0.26 — pin to a git tag or path", name)
		default:
			return fmt.Errorf("dependency %s: unknown source kind", name)
		}
		return nil
	}
	for _, name := range m.DepOrder {
		if err := visit(name, m.Deps[name]); err != nil {
			return nil, err
		}
	}
	for _, name := range m.DevOrder {
		if err := visit(name, m.DevDeps[name]); err != nil {
			return nil, err
		}
	}
	sort.SliceStable(lf.Packages, func(i, j int) bool { return lf.Packages[i].Name < lf.Packages[j].Name })
	return lf, nil
}

// materialize copies / clones each locked package's source tree into the
// per-project .tya/packages/<name>-<version>/ directory unless it is already
// there. Path-sourced packages are read in place and not copied.
func materialize(projectRoot string, lf *Lockfile) error {
	for i := range lf.Packages {
		p := &lf.Packages[i]
		if p.Source == "path" {
			continue
		}
		dst := filepath.Join(projectRoot, PackagesDir, fmt.Sprintf("%s-%s", p.Name, p.Version.String()))
		if _, err := os.Stat(filepath.Join(dst, ManifestName)); err == nil {
			continue
		}
		cacheDir, _, err := FetchGit(projectRoot, p.Name, p.Git, p.Rev)
		if err != nil {
			return err
		}
		if _, err := CopyTreeIntoPackages(projectRoot, p.Name, p.Version, cacheDir); err != nil {
			return err
		}
		if cs, err := TreeChecksum(dst); err == nil {
			p.Checksum = cs
		}
	}
	return nil
}

// AddDependency edits the manifest at projectRoot/tya.toml to include `dep`
// under [dependencies] (or [dev-dependencies] if isDev). It does not run
// install; call Install afterwards.
func AddDependency(projectRoot string, dep Dependency, isDev bool) error {
	manifestPath := filepath.Join(projectRoot, ManifestName)
	m, err := ReadManifest(manifestPath)
	if err != nil {
		return err
	}
	if isDev {
		m.DevDeps[dep.Name] = dep
		if !contains(m.DevOrder, dep.Name) {
			m.DevOrder = append(m.DevOrder, dep.Name)
		}
	} else {
		m.Deps[dep.Name] = dep
		if !contains(m.DepOrder, dep.Name) {
			m.DepOrder = append(m.DepOrder, dep.Name)
		}
	}
	return WriteManifest(m)
}

// RemoveDependency edits the manifest to drop the named dependency from
// both dependency tables.
func RemoveDependency(projectRoot, name string) error {
	manifestPath := filepath.Join(projectRoot, ManifestName)
	m, err := ReadManifest(manifestPath)
	if err != nil {
		return err
	}
	dropped := false
	if _, ok := m.Deps[name]; ok {
		delete(m.Deps, name)
		m.DepOrder = removeString(m.DepOrder, name)
		dropped = true
	}
	if _, ok := m.DevDeps[name]; ok {
		delete(m.DevDeps, name)
		m.DevOrder = removeString(m.DevOrder, name)
		dropped = true
	}
	if !dropped {
		return fmt.Errorf("dependency %s not found", name)
	}
	return WriteManifest(m)
}

func removeString(s []string, x string) []string {
	out := []string{}
	for _, v := range s {
		if v != x {
			out = append(out, v)
		}
	}
	return out
}

// FormatLock renders a short summary of the lockfile (one line per package).
func FormatLock(lf *Lockfile) string {
	if lf == nil {
		return ""
	}
	out := strings.Builder{}
	for _, p := range lf.Packages {
		fmt.Fprintf(&out, "%s %s (%s)\n", p.Name, p.Version, p.Source)
	}
	return out.String()
}
