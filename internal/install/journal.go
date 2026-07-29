package install

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/fsutil"
	"github.com/LuchoC-Dev/agent-kits/internal/security"
)

// journal records every filesystem mutation of one operation so it can be undone.
//
// Backups live in a temporary directory outside the project: a failed install must not
// leave stray files inside the workspace it was supposed to change.
type journal struct {
	project   string
	backupDir string
	entries   []journalEntry
}

type journalEntry struct {
	// path is project-relative and slash-separated.
	path string
	// existed reports whether the file was present before the operation.
	existed bool
	// backup is the absolute path of the saved copy, empty when existed is false.
	backup string
	// removed reports whether the operation deleted the file.
	removed bool
}

func newJournal(project string) (*journal, error) {
	dir, err := os.MkdirTemp("", "agent-kits-journal-*")
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, err, "cannot create a rollback directory")
	}
	return &journal{project: project, backupDir: dir}, nil
}

// backup saves the current content of a path, if any, and records the intent.
func (j *journal) backup(path string, removing bool) (string, error) {
	abs, err := security.Contain(j.project, path)
	if err != nil {
		return "", err
	}
	entry := journalEntry{path: path, removed: removing}

	if fsutil.Exists(abs) {
		content, readErr := os.ReadFile(abs)
		if readErr != nil {
			return "", errs.Wrap(errs.CodeInternal, readErr, "cannot back up %s", path)
		}
		entry.existed = true
		entry.backup = filepath.Join(j.backupDir, fmt.Sprintf("%04d.bak", len(j.entries)))
		if writeErr := os.WriteFile(entry.backup, content, 0o644); writeErr != nil {
			return "", errs.Wrap(errs.CodeInternal, writeErr, "cannot back up %s", path)
		}
	}
	j.entries = append(j.entries, entry)
	return abs, nil
}

// write installs new content at a project-relative path.
func (j *journal) write(path string, content []byte) error {
	abs, err := j.backup(path, false)
	if err != nil {
		return err
	}
	if err := fsutil.WriteFileAtomic(abs, content, 0o644); err != nil {
		return errs.Wrap(errs.CodeInternal, err, "cannot write %s", path)
	}
	return nil
}

// remove deletes a project-relative path.
func (j *journal) remove(path string) error {
	abs, err := j.backup(path, true)
	if err != nil {
		return err
	}
	if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
		return errs.Wrap(errs.CodeInternal, err, "cannot remove %s", path)
	}
	return nil
}

// rollback restores every recorded path to its pre-operation state, newest first.
func (j *journal) rollback() error {
	var firstErr error
	for i := len(j.entries) - 1; i >= 0; i-- {
		entry := j.entries[i]
		abs, err := security.Contain(j.project, entry.path)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		switch {
		case entry.existed:
			content, readErr := os.ReadFile(entry.backup)
			if readErr != nil {
				if firstErr == nil {
					firstErr = readErr
				}
				continue
			}
			if writeErr := fsutil.WriteFileAtomic(abs, content, 0o644); writeErr != nil && firstErr == nil {
				firstErr = writeErr
			}
		case !entry.removed:
			// The operation created this file, so undoing it means deleting it.
			if removeErr := os.Remove(abs); removeErr != nil && !os.IsNotExist(removeErr) && firstErr == nil {
				firstErr = removeErr
			}
		}
	}
	return firstErr
}

// discard drops the backups. It is safe to call twice.
func (j *journal) discard() {
	if j.backupDir == "" {
		return
	}
	_ = os.RemoveAll(j.backupDir)
	j.backupDir = ""
}
