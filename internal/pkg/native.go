package pkg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type NativePackage struct {
	Name string
	Root string
	Data Native
}

type NativePlan struct {
	Packages    []NativePackage
	Sources     []string
	IncludeDirs []string
	CFlags      []string
	LDFlags     []string
	Functions   map[string]NativeFunction
	FuncOrder   []string
}

func CollectNative(projectRoot string) (*NativePlan, error) {
	manifestPath := filepath.Join(projectRoot, ManifestName)
	m, err := ReadManifest(manifestPath)
	if err != nil {
		return nil, err
	}
	plan := &NativePlan{Functions: map[string]NativeFunction{}}
	if err := addNativePackage(plan, NativePackage{Name: m.Name, Root: projectRoot, Data: m.Native}); err != nil {
		return nil, err
	}
	lockPath := filepath.Join(projectRoot, LockfileName)
	lf, err := ReadLockfile(lockPath)
	if err == nil {
		for i := range lf.Packages {
			lp := &lf.Packages[i]
			root := packageRoot(projectRoot, lp)
			sub, err := ReadManifest(filepath.Join(root, ManifestName))
			if err != nil {
				return nil, fmt.Errorf("native package %s: %v", lp.Name, err)
			}
			if err := addNativePackage(plan, NativePackage{Name: lp.Name, Root: root, Data: sub.Native}); err != nil {
				return nil, err
			}
		}
	}
	return plan, nil
}

func packageRoot(projectRoot string, lp *LockedPackage) string {
	if lp.Source == "path" {
		if filepath.IsAbs(lp.PathRef) {
			return lp.PathRef
		}
		return filepath.Join(projectRoot, lp.PathRef)
	}
	return PackageDir(projectRoot, lp)
}

func addNativePackage(plan *NativePlan, p NativePackage) error {
	if !nativeHasFields(p.Data) {
		return nil
	}
	for _, source := range p.Data.Sources {
		full := filepath.Join(p.Root, source)
		if info, err := os.Stat(full); err != nil || info.IsDir() {
			return fmt.Errorf("[TYA-E0920] native package %s: missing source %s", p.Name, source)
		}
		plan.Sources = append(plan.Sources, full)
	}
	for _, header := range p.Data.Headers {
		full := filepath.Join(p.Root, header)
		if info, err := os.Stat(full); err != nil || info.IsDir() {
			return fmt.Errorf("[TYA-E0921] native package %s: missing header %s", p.Name, header)
		}
	}
	for _, include := range p.Data.IncludeDirs {
		full := filepath.Join(p.Root, include)
		if info, err := os.Stat(full); err != nil || !info.IsDir() {
			return fmt.Errorf("[TYA-E0922] native package %s: missing include directory %s", p.Name, include)
		}
		plan.IncludeDirs = appendUnique(plan.IncludeDirs, full)
	}
	for _, flag := range p.Data.CFlags {
		plan.CFlags = appendUnique(plan.CFlags, flag)
	}
	for _, flag := range p.Data.LDFlags {
		plan.LDFlags = appendUnique(plan.LDFlags, flag)
	}
	for _, dep := range p.Data.PkgConfig {
		cflags, ldflags, err := pkgConfigFlags(p.Name, dep)
		if err != nil {
			return err
		}
		for _, flag := range cflags {
			plan.CFlags = appendUnique(plan.CFlags, flag)
		}
		for _, flag := range ldflags {
			plan.LDFlags = appendUnique(plan.LDFlags, flag)
		}
	}
	for _, name := range p.Data.FuncOrder {
		fn := p.Data.Functions[name]
		if existing, ok := plan.Functions[name]; ok && existing.Symbol != fn.Symbol {
			return fmt.Errorf("[TYA-E0923] native function %s declared by multiple packages", name)
		}
		plan.Functions[name] = fn
		if !contains(plan.FuncOrder, name) {
			plan.FuncOrder = append(plan.FuncOrder, name)
		}
	}
	plan.Packages = append(plan.Packages, p)
	return nil
}

func pkgConfigFlags(packageName, dep string) ([]string, []string, error) {
	if _, err := exec.LookPath("pkg-config"); err != nil {
		return nil, nil, fmt.Errorf("[TYA-E0924] native package %s requires pkg-config, but pkg-config was not found", packageName)
	}
	if err := exec.Command("pkg-config", "--exists", dep).Run(); err != nil {
		return nil, nil, fmt.Errorf("[TYA-E0925] native package %s requires missing pkg-config dependency %s", packageName, dep)
	}
	cflags, err := exec.Command("pkg-config", "--cflags", dep).Output()
	if err != nil {
		return nil, nil, fmt.Errorf("[TYA-E0925] native package %s requires missing pkg-config dependency %s", packageName, dep)
	}
	ldflags, err := exec.Command("pkg-config", "--libs", dep).Output()
	if err != nil {
		return nil, nil, fmt.Errorf("[TYA-E0925] native package %s requires missing pkg-config dependency %s", packageName, dep)
	}
	return strings.Fields(string(cflags)), strings.Fields(string(ldflags)), nil
}

func appendUnique(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
