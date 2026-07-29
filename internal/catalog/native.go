package catalog

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/fsutil"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
	"github.com/LuchoC-Dev/agent-kits/internal/security"
	"github.com/LuchoC-Dev/agent-kits/internal/source"
)

// loadNative reads every agent-kit.json in the checkout. It reports whether the layout
// was present at all, so LoadCheckout can tell an empty source from an unrecognised one.
func (l *Loader) loadNative(checkout source.Checkout, cat *Catalog) (bool, error) {
	root := checkout.Root
	var manifests []string

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if skipDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() == model.ManifestFilename {
			manifests = append(manifests, path)
		}
		return nil
	})
	if err != nil {
		return false, errs.Wrap(errs.CodeSourceUnavailable, err,
			"cannot read source %s", checkout.Source.Name)
	}
	if len(manifests) == 0 {
		return false, nil
	}
	for _, path := range manifests {
		res, err := l.readNativeManifest(checkout, path)
		if err != nil {
			return true, err
		}
		if err := cat.add(res); err != nil {
			return true, err
		}
	}
	return true, nil
}

// skipDir excludes directories that never contain catalog resources.
func skipDir(name string) bool {
	switch name {
	case ".git", ".agents", "node_modules", "testdata", "docs", "meta":
		return true
	}
	return strings.HasPrefix(name, ".") && name != "."
}

func (l *Loader) readNativeManifest(checkout source.Checkout, path string) (*model.Resource, error) {
	var manifest model.Manifest
	if err := fsutil.ReadJSON(path, &manifest); err != nil {
		return nil, errs.Wrap(errs.CodeInvalidManifest, err, "cannot read %s", path)
	}
	dir := filepath.Dir(path)

	if len(manifest.Files) == 0 {
		discovered, err := l.discoverFiles(dir, manifest.ID)
		if err != nil {
			return nil, err
		}
		manifest.Files = discovered
	}
	for _, rel := range manifest.Files {
		if err := security.CheckRelPath(rel); err != nil {
			return nil, err
		}
		target := filepath.Join(dir, fsutil.FromSlash(rel))
		info, statErr := os.Lstat(target)
		if statErr != nil {
			return nil, errs.New(errs.CodeInvalidManifest,
				"resource %s declares missing file %q", manifest.ID, rel)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, errs.New(errs.CodeUnsafePath,
				"resource %s declares symlink %q", manifest.ID, rel)
		}
		if err := l.Limits.CheckSize(rel, info.Size()); err != nil {
			return nil, err
		}
	}
	if err := l.Limits.CheckFileCount(string(manifest.ID), len(manifest.Files)); err != nil {
		return nil, err
	}
	if err := manifest.Validate(); err != nil {
		return nil, err
	}
	return &model.Resource{
		Manifest: manifest,
		Source:   checkout.Source.Name,
		Root:     dir,
		Commit:   checkout.Commit,
		Access:   checkout.Source.Access,
		Trust:    checkout.Source.Trust,
	}, nil
}

// discoverFiles lists a resource directory when the manifest does not enumerate files.
func (l *Loader) discoverFiles(dir string, id model.ID) ([]string, error) {
	files, symlinks, err := fsutil.WalkFiles(dir)
	if err != nil {
		return nil, errs.Wrap(errs.CodeSourceUnavailable, err, "cannot read %s", dir)
	}
	if len(symlinks) > 0 {
		return nil, errs.New(errs.CodeUnsafePath,
			"resource %s contains symlinks or special files: %s", id, strings.Join(symlinks, ", "))
	}
	out := make([]string, 0, len(files))
	for _, rel := range files {
		if rel == model.ManifestFilename {
			continue
		}
		out = append(out, rel)
	}
	return out, nil
}
