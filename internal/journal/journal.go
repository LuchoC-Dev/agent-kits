// Package journal makes a multi-file operation recoverable.
//
// Every file an operation overwrites or deletes is copied aside first, so any failure can
// restore the project to its previous state. A partially applied operation is never left
// behind (02-architecture-direction.md §7).
//
// Backups live in a temporary directory outside the project: a failed operation must not
// leave stray files inside the workspace it was supposed to change.
package journal

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/fsutil"
	"github.com/LuchoC-Dev/agent-kits/internal/security"
)

// Journal records the filesystem mutations of one operation so they can be undone.
type Journal struct {
	project   string
	backupDir string
	entries   []entry
}

type entry struct {
	// path is project-relative and slash-separated.
	path string
	// existed reports whether the file was present before the operation.
	existed bool
	// backup is the absolute path of the saved copy, empty when existed is false.
	backup string
	// removed reports whether the operation deleted the file.
	removed bool
}

// New opens a journal for one project.
func New(project string) (*Journal, error) {
	dir, err := os.MkdirTemp("", "agent-kits-journal-*")
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, err, "cannot create a rollback directory")
	}
	return &Journal{project: project, backupDir: dir}, nil
}

// save copies the current content of a path, if any, and records the intent.
func (j *Journal) save(path string, removing bool) (string, error) {
	abs, err := security.Contain(j.project, path)
	if err != nil {
		return "", err
	}
	record := entry{path: path, removed: removing}

	if fsutil.Exists(abs) {
		content, readErr := os.ReadFile(abs)
		if readErr != nil {
			return "", errs.Wrap(errs.CodeInternal, readErr, "cannot back up %s", path)
		}
		record.existed = true
		record.backup = filepath.Join(j.backupDir, fmt.Sprintf("%04d.bak", len(j.entries)))
		if writeErr := os.WriteFile(record.backup, content, 0o644); writeErr != nil {
			return "", errs.Wrap(errs.CodeInternal, writeErr, "cannot back up %s", path)
		}
	}
	j.entries = append(j.entries, record)
	return abs, nil
}

// Write installs new content at a project-relative path.
func (j *Journal) Write(path string, content []byte) error {
	abs, err := j.save(path, false)
	if err != nil {
		return err
	}
	if err := fsutil.WriteFileAtomic(abs, content, 0o644); err != nil {
		return errs.Wrap(errs.CodeInternal, err, "cannot write %s", path)
	}
	return nil
}

// Remove deletes a project-relative path.
func (j *Journal) Remove(path string) error {
	abs, err := j.save(path, true)
	if err != nil {
		return err
	}
	if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
		return errs.Wrap(errs.CodeInternal, err, "cannot remove %s", path)
	}
	return nil
}

// Rollback restores every recorded path to its pre-operation state, newest first.
func (j *Journal) Rollback() error {
	var firstErr error
	for i := len(j.entries) - 1; i >= 0; i-- {
		record := j.entries[i]
		abs, err := security.Contain(j.project, record.path)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		switch {
		case record.existed:
			content, readErr := os.ReadFile(record.backup)
			if readErr != nil {
				if firstErr == nil {
					firstErr = readErr
				}
				continue
			}
			if writeErr := fsutil.WriteFileAtomic(abs, content, 0o644); writeErr != nil && firstErr == nil {
				firstErr = writeErr
			}
		case !record.removed:
			// The operation created this file, so undoing it means deleting it.
			if removeErr := os.Remove(abs); removeErr != nil && !os.IsNotExist(removeErr) && firstErr == nil {
				firstErr = removeErr
			}
		}
	}
	return firstErr
}

// Discard drops the backups. It is safe to call twice.
func (j *Journal) Discard() {
	if j.backupDir == "" {
		return
	}
	_ = os.RemoveAll(j.backupDir)
	j.backupDir = ""
}
