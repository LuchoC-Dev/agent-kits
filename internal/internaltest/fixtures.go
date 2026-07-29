// Package internaltest builds throwaway sources and projects for tests.
//
// Fixtures are constructed in code rather than checked in, so a test states exactly the
// catalog shape it depends on and nothing else.
package internaltest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LuchoC-Dev/agent-kits/internal/fsutil"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
	"github.com/LuchoC-Dev/agent-kits/internal/source"
)

// Resource declares one resource to materialise in a native-layout source.
type Resource struct {
	ID           string
	Type         model.Type
	Version      string
	Description  string
	Dependencies []model.Dependency
	// Files maps a resource-relative path to its content.
	Files map[string]string
	// Runtimes restricts runtime compatibility. Empty means every runtime.
	Runtimes []string
}

// WriteNativeSource materialises resources into dir using the native layout.
func WriteNativeSource(t *testing.T, dir string, resources ...Resource) {
	t.Helper()
	for _, res := range resources {
		resourceDir := filepath.Join(dir, string(res.Type)+"s", filepath.FromSlash(res.ID))
		if err := fsutil.EnsureDir(resourceDir); err != nil {
			t.Fatalf("cannot create %s: %v", resourceDir, err)
		}
		files := res.Files
		if len(files) == 0 {
			files = map[string]string{"README.md": "# " + res.ID + "\n"}
		}
		names := make([]string, 0, len(files))
		for name, content := range files {
			target := filepath.Join(resourceDir, filepath.FromSlash(name))
			if err := fsutil.EnsureDir(filepath.Dir(target)); err != nil {
				t.Fatalf("cannot create %s: %v", filepath.Dir(target), err)
			}
			if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
				t.Fatalf("cannot write %s: %v", target, err)
			}
			names = append(names, name)
		}
		version := res.Version
		if version == "" {
			version = "1.0.0"
		}
		manifest := model.Manifest{
			SchemaVersion: model.ManifestSchemaVersion,
			ID:            model.ID(res.ID),
			Type:          res.Type,
			Version:       version,
			Description:   res.Description,
			Dependencies:  res.Dependencies,
			Files:         names,
			Runtimes:      res.Runtimes,
		}
		manifestPath := filepath.Join(resourceDir, model.ManifestFilename)
		if err := fsutil.WriteJSON(manifestPath, manifest); err != nil {
			t.Fatalf("cannot write %s: %v", manifestPath, err)
		}
	}
}

// Checkout returns a source checkout rooted at dir.
func Checkout(name, dir string, access model.Access, trust model.Trust) source.Checkout {
	return source.Checkout{
		Source: source.Source{Name: name, URL: dir, Access: access, Trust: trust},
		Root:   dir,
		Local:  true,
	}
}

// PublicCheckout is Checkout with the common public and trusted settings.
func PublicCheckout(name, dir string) source.Checkout {
	return Checkout(name, dir, model.AccessPublic, model.TrustTrusted)
}

// Dep is a convenience constructor for a dependency.
func Dep(id string, constraint ...string) model.Dependency {
	dep := model.Dependency{ID: model.ID(id)}
	if len(constraint) > 0 {
		dep.Version = constraint[0]
	}
	return dep
}

// WriteFile writes content to a path inside dir, creating parents.
func WriteFile(t *testing.T, dir, rel, content string) string {
	t.Helper()
	target := filepath.Join(dir, filepath.FromSlash(rel))
	if err := fsutil.EnsureDir(filepath.Dir(target)); err != nil {
		t.Fatalf("cannot create %s: %v", filepath.Dir(target), err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write %s: %v", target, err)
	}
	return target
}

// ReadFile reads a file inside dir and fails the test if it is unreadable.
func ReadFile(t *testing.T, dir, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("cannot read %s: %v", rel, err)
	}
	return string(data)
}

// Exists reports whether a path inside dir exists.
func Exists(dir, rel string) bool {
	return fsutil.Exists(filepath.Join(dir, filepath.FromSlash(rel)))
}
