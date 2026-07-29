// Package fsutil holds the filesystem primitives shared by the planner and the
// installer: content addressing, atomic writes and JSON persistence.
package fsutil

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ChecksumPrefix is the algorithm label carried by every checksum string.
const ChecksumPrefix = "sha256:"

// ChecksumBytes returns the content address of data.
func ChecksumBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return ChecksumPrefix + hex.EncodeToString(sum[:])
}

// ChecksumFile returns the content address and size of the file at path.
func ChecksumFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return ChecksumPrefix + hex.EncodeToString(h.Sum(nil)), n, nil
}

// ChecksumTree folds a set of per-file checksums into one stable resource checksum.
// Paths are sorted first so the result does not depend on walk order.
func ChecksumTree(files map[string]string) string {
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	h := sha256.New()
	for _, p := range paths {
		fmt.Fprintf(h, "%s\x00%s\x00", ToSlash(p), files[p])
	}
	return ChecksumPrefix + hex.EncodeToString(h.Sum(nil))
}

// ToSlash normalises a path to forward slashes, the form used in manifests, lockfiles
// and plan output regardless of the host platform.
func ToSlash(path string) string { return filepath.ToSlash(path) }

// FromSlash converts a canonical slash path to the host form.
func FromSlash(path string) string { return filepath.FromSlash(path) }

// Exists reports whether path exists, following no symlinks.
func Exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// IsRegular reports whether path is an existing regular file (not a symlink or device).
func IsRegular(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular()
}

// EnsureDir creates a directory and its parents.
func EnsureDir(path string) error { return os.MkdirAll(path, 0o755) }

// WriteFileAtomic writes data to path via a temporary file in the same directory,
// then renames it into place. A failed write therefore never leaves a partial file.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := EnsureDir(dir); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".agent-kits-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		tmp.Close()
		os.Remove(tmpName)
	}()

	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return err
	}
	// Windows rejects a rename onto an existing file, so clear the target first.
	if Exists(path) {
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return os.Rename(tmpName, path)
}

// ReadJSON decodes the JSON file at path into v, rejecting unknown fields so that a
// manifest with a typo fails loudly instead of silently losing data.
func ReadJSON(path string, v any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	return nil
}

// WriteJSON writes v as indented JSON with a trailing newline.
func WriteJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return WriteFileAtomic(path, append(data, '\n'), 0o644)
}

// RemoveEmptyDirs deletes dir and its now-empty parents, stopping at stop (exclusive).
// It is how removing a resource leaves no empty scaffolding behind.
func RemoveEmptyDirs(dir, stop string) error {
	stopAbs, err := filepath.Abs(stop)
	if err != nil {
		return err
	}
	current, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	for strings.HasPrefix(current, stopAbs) && current != stopAbs {
		entries, err := os.ReadDir(current)
		if err != nil || len(entries) > 0 {
			return nil
		}
		if err := os.Remove(current); err != nil {
			return nil
		}
		current = filepath.Dir(current)
	}
	return nil
}

// WalkFiles returns every regular file under root, as paths relative to root in slash
// form, sorted. Symlinks are reported separately so callers can reject them.
func WalkFiles(root string) (files []string, symlinks []string, err error) {
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return nil
		}
		rel = ToSlash(rel)
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			symlinks = append(symlinks, rel)
			if info.IsDir() {
				return filepath.SkipDir
			}
		case info.IsDir():
			return nil
		case info.Mode().IsRegular():
			files = append(files, rel)
		default:
			symlinks = append(symlinks, rel) // devices, sockets and pipes are equally unwelcome
		}
		return nil
	})
	sort.Strings(files)
	sort.Strings(symlinks)
	return files, symlinks, err
}
