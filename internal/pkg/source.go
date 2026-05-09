package pkg

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// FetchedPackage points at a directory on disk containing a tya.toml at its
// root. The Manager copies / clones source repositories into per-project
// .tya/packages/<name>-<version>/ so that import resolution finds them
// directly.
type FetchedPackage struct {
	Name     string
	Version  Version
	Dir      string // absolute path to the package root
	Source   string // "git" or "path"
	Git      string
	Rev      string
	PathRef  string
	Checksum string
}

// FetchGit clones (or reuses) a git source and checks out the requested ref.
// It writes the package into projectRoot/.tya/packages/<name>-<version>/.
// Returns the on-disk path and resolved commit rev.
func FetchGit(projectRoot, name, url, ref string) (string, string, error) {
	cacheRoot := filepath.Join(projectRoot, ".tya", "cache", "git", safeKey(url))
	if err := os.MkdirAll(filepath.Dir(cacheRoot), 0755); err != nil {
		return "", "", err
	}
	if _, err := os.Stat(filepath.Join(cacheRoot, ".git")); os.IsNotExist(err) {
		_ = os.RemoveAll(cacheRoot)
		cmd := exec.Command("git", "clone", "--quiet", url, cacheRoot)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", "", fmt.Errorf("git clone %s: %v: %s", url, err, string(out))
		}
	} else {
		cmd := exec.Command("git", "-C", cacheRoot, "fetch", "--quiet", "--all", "--tags")
		_ = cmd.Run() // best-effort; offline still works
	}
	cmd := exec.Command("git", "-C", cacheRoot, "checkout", "--quiet", "--detach", ref)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", "", fmt.Errorf("git checkout %s: %v: %s", ref, err, string(out))
	}
	revOut, err := exec.Command("git", "-C", cacheRoot, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", "", err
	}
	rev := strings.TrimSpace(string(revOut))
	return cacheRoot, rev, nil
}

// CopyTreeIntoPackages copies a source directory (excluding .git) into
// projectRoot/.tya/packages/<name>-<version>/, returning the destination
// path.
func CopyTreeIntoPackages(projectRoot, name string, ver Version, src string) (string, error) {
	dst := filepath.Join(projectRoot, ".tya", "packages", fmt.Sprintf("%s-%s", name, ver.String()))
	if err := os.RemoveAll(dst); err != nil {
		return "", err
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return "", err
	}
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if strings.HasPrefix(rel, ".git") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(p, target)
	})
	return dst, err
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// TreeChecksum returns a sha256 over the sorted relative paths and contents.
func TreeChecksum(root string) (string, error) {
	type entry struct {
		path string
		data []byte
	}
	entries := []entry{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		entries = append(entries, entry{path: rel, data: data})
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })
	h := sha256.New()
	for _, e := range entries {
		fmt.Fprintf(h, "%s\x00%d\x00", e.path, len(e.data))
		h.Write(e.data)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

func safeKey(url string) string {
	out := strings.Builder{}
	for _, r := range url {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
		} else {
			out.WriteByte('_')
		}
	}
	return out.String()
}

// PackageDir returns the per-project package directory for a locked package.
func PackageDir(projectRoot string, lp *LockedPackage) string {
	if lp.Source == "path" {
		// Path sources are referenced directly.
		if filepath.IsAbs(lp.PathRef) {
			return lp.PathRef
		}
		return filepath.Join(projectRoot, lp.PathRef)
	}
	return filepath.Join(projectRoot, ".tya", "packages", fmt.Sprintf("%s-%s", lp.Name, lp.Version.String()))
}
