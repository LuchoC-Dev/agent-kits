// Package workspace reads and writes the one bookkeeping file a project carries: the
// Agent Kits lockfile.
//
// From lockfile schema 2 on, that file holds every piece of state Agent Kits owns —
// project identity, installed resources and the record of any migration (D-030). The
// inherited workspace.json is no longer written by any command; what remains of it lives
// in legacy.go and exists only so a project can be migrated off it.
package workspace

import (
	"encoding/json"

	"github.com/LuchoC-Dev/agent-kits/internal/adapter"
	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/fsutil"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
	"github.com/LuchoC-Dev/agent-kits/internal/security"
)

// LoadLock reads the lockfile, returning an empty lock when the project has none.
//
// A lockfile written under the superseded schema is upgraded in memory, so the rest of the
// system only ever sees the current one and every write produces it (D-030).
func LoadLock(project string, a adapter.Adapter) (*model.Lock, error) {
	lock, _, err := LoadLockDetail(project, a)
	return lock, err
}

// LoadLockDetail is LoadLock plus the schema version the file declared on disk, which the
// migration reports and a caller may need to distinguish an upgrade from a no-op.
// The reported version is 0 when the project has no lockfile.
func LoadLockDetail(project string, a adapter.Adapter) (*model.Lock, int, error) {
	path, err := security.Contain(project, a.LockPath())
	if err != nil {
		return nil, 0, err
	}
	if !fsutil.Exists(path) {
		return model.NewLock(a.Name()), 0, nil
	}
	var lock model.Lock
	if err := fsutil.ReadJSON(path, &lock); err != nil {
		return nil, 0, errs.Wrap(errs.CodeWorkspaceInvalid, err, "cannot read %s", a.LockPath())
	}
	if err := lock.Validate(); err != nil {
		return nil, 0, err
	}
	found := lock.SchemaVersion
	lock.Upgrade()
	return &lock, found, nil
}

// LockBytes renders a lockfile exactly as it would be written.
func LockBytes(lock *model.Lock) ([]byte, error) {
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, err, "cannot encode the lockfile")
	}
	return append(data, '\n'), nil
}
