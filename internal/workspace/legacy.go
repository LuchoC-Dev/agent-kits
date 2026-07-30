package workspace

import (
	"encoding/json"
	"os"

	"github.com/LuchoC-Dev/agent-kits/internal/adapter"
	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/fsutil"
	"github.com/LuchoC-Dev/agent-kits/internal/security"
)

// This file is the whole of Agent Kits' remaining knowledge of workspace.json. It exists
// only to migrate a project onto lockfile v2 (D-031); when the migration window closes it
// is deleted as a unit, and nothing else in the codebase has to change.
//
// Everything here reads. No command writes a descriptor any more, which is why the type
// below has an UnmarshalJSON and no marshaller: the original bytes are what a migration
// preserves, not a re-encoding of this struct.

// SchemaVersion is the newest workspace.json schema this build can read.
const SchemaVersion = 2

// Entry is an installed resource recorded in workspace.json.
type Entry struct {
	ID          string `json:"id"`
	Class       int    `json:"class,omitempty"`
	Source      string `json:"source"`
	InstalledAt string `json:"installed_at"`
}

// Pack is the composition record of workspace.json.
type Pack struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	InstalledAt string `json:"installed_at"`
}

// Stack is the detected project stack.
type Stack struct {
	Detected   []string `json:"detected"`
	Source     string   `json:"source"`
	Confidence string   `json:"confidence"`
}

// Flags are the workspace lifecycle markers.
type Flags struct {
	Initialized bool    `json:"initialized"`
	RepairedAt  *string `json:"repaired_at"`
	UpgradedAt  *string `json:"upgraded_at"`
}

// Descriptor is workspace.json.
type Descriptor struct {
	SchemaVersion int      `json:"$schema_version"`
	ID            string   `json:"id"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
	SystemVersion string   `json:"system_version"`
	Runtime       string   `json:"runtime"`
	Pack          *Pack    `json:"pack"`
	Stack         *Stack   `json:"stack,omitempty"`
	Skills        []Entry  `json:"skills"`
	Agents        []Entry  `json:"agents"`
	Disciplines   []string `json:"disciplines"`
	Flags         Flags    `json:"flags"`
	Structure     []string `json:"structure"`

	// extra keeps fields written by another tool so nothing is lost on migration.
	extra map[string]json.RawMessage
	// raw keeps every field exactly as it was read, including the managed ones, so a
	// migration can preserve an inherited value verbatim instead of re-encoding it.
	raw map[string]json.RawMessage
}

// managedFields are the keys Descriptor itself decodes; anything else is unmanaged.
var managedFields = map[string]bool{
	"$schema_version": true, "id": true, "created_at": true, "updated_at": true,
	"system_version": true, "runtime": true, "pack": true, "stack": true,
	"skills": true, "agents": true, "disciplines": true, "flags": true, "structure": true,
}

// UnmarshalJSON decodes the descriptor while retaining every field it does not manage.
func (d *Descriptor) UnmarshalJSON(data []byte) error {
	type alias Descriptor
	var base alias
	if err := json.Unmarshal(data, &base); err != nil {
		return err
	}
	*d = Descriptor(base)

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	d.extra = map[string]json.RawMessage{}
	d.raw = raw
	for key, value := range raw {
		if !managedFields[key] {
			d.extra[key] = value
		}
	}
	return nil
}

// Extra returns the fields no version of Agent Kits manages, exactly as they were read.
func (d *Descriptor) Extra() map[string]json.RawMessage {
	out := map[string]json.RawMessage{}
	for key, value := range d.extra {
		out[key] = append(json.RawMessage(nil), value...)
	}
	return out
}

// RawField returns one field of the descriptor as it was read on disk.
func (d *Descriptor) RawField(name string) (json.RawMessage, bool) {
	value, ok := d.raw[name]
	if !ok {
		return nil, false
	}
	return append(json.RawMessage(nil), value...), true
}

// ParseDescriptor decodes workspace.json and checks that its schema is one this build
// understands. Every field is retained, including the ones it does not manage.
func ParseDescriptor(data []byte) (*Descriptor, error) {
	var descriptor Descriptor
	if err := json.Unmarshal(data, &descriptor); err != nil {
		return nil, errs.Wrap(errs.CodeWorkspaceInvalid, err, "cannot parse %s", LegacyPath)
	}
	if descriptor.SchemaVersion != SchemaVersion && descriptor.SchemaVersion != 1 {
		return nil, errs.New(errs.CodeWorkspaceInvalid,
			"unsupported workspace.json $schema_version %d (expected 1 or %d)",
			descriptor.SchemaVersion, SchemaVersion)
	}
	return &descriptor, nil
}

// Pending reports whether a project still carries a workspace.json that must be migrated
// before any command may change its state.
func Pending(project string) bool {
	abs, err := security.Contain(project, LegacyPath)
	if err != nil {
		return false
	}
	return fsutil.Exists(abs)
}

// PendingError is the failure every mutating command returns while a project still has an
// unmigrated workspace.json. Two sources of truth are never operated on at once.
func PendingError() error {
	return errs.New(errs.CodeWorkspaceInvalid,
		"%s has not been migrated yet, so this project has two sources of truth", LegacyPath).
		Hint("run `agent-kits migrate --project <path>` first")
}

// LegacyPath is where the conversational kits-init flow writes its descriptor. It is not
// runtime-specific: every supported adapter shares the .agents layout.
const LegacyPath = adapter.WorkspaceDir + "/" + adapter.WorkspaceFile

// BackupPath is the byte-for-byte copy a migration leaves behind. It belongs to the user
// and Agent Kits never deletes it.
const BackupPath = adapter.WorkspaceDir + "/workspace.json.migrated.bak"

// MaxLegacyBytes bounds how large a descriptor may be before it is refused. A workspace
// descriptor is a small metadata file; anything larger is not one.
const MaxLegacyBytes int64 = 1 << 20 // 1 MiB

// Legacy is an inherited workspace.json read without loss: the parsed descriptor for the
// fields the migration maps, and the original bytes for everything else.
type Legacy struct {
	// Path is project-relative and slash-separated.
	Path string
	// Raw is the file exactly as it is on disk. The backup is written from these bytes, not
	// from a re-serialisation, so no formatting or ordering detail is invented.
	Raw []byte
	// Descriptor is the parsed view, which retains unknown fields (see Descriptor.Extra).
	Descriptor *Descriptor
}

// LoadLegacy reads workspace.json losslessly, or reports that the project has none.
//
// A descriptor that cannot be read, parsed or trusted fails with workspace_invalid rather
// than being ignored: silently skipping it would let a later command overwrite state the
// migration was supposed to preserve.
func LoadLegacy(project string) (*Legacy, bool, error) {
	abs, err := security.Contain(project, LegacyPath)
	if err != nil {
		return nil, false, err
	}
	info, err := os.Lstat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, errs.Wrap(errs.CodeWorkspaceInvalid, err, "cannot inspect %s", LegacyPath)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, false, errs.New(errs.CodeUnsafePath,
			"%s is a symlink; Agent Kits will not migrate through it", LegacyPath)
	}
	if !info.Mode().IsRegular() {
		return nil, false, errs.New(errs.CodeUnsafePath, "%s is not a regular file", LegacyPath)
	}
	if info.Size() > MaxLegacyBytes {
		return nil, false, errs.New(errs.CodeWorkspaceInvalid,
			"%s is %d bytes, over the %d byte limit", LegacyPath, info.Size(), MaxLegacyBytes)
	}

	raw, err := os.ReadFile(abs)
	if err != nil {
		return nil, false, errs.Wrap(errs.CodeWorkspaceInvalid, err, "cannot read %s", LegacyPath)
	}
	descriptor, err := ParseDescriptor(raw)
	if err != nil {
		return nil, false, err
	}
	return &Legacy{Path: LegacyPath, Raw: raw, Descriptor: descriptor}, true, nil
}

// LoadBackup reads an existing migration backup, or reports that there is none.
func LoadBackup(project string) ([]byte, bool, error) {
	abs, err := security.Contain(project, BackupPath)
	if err != nil {
		return nil, false, err
	}
	if !fsutil.Exists(abs) {
		return nil, false, nil
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		return nil, false, errs.Wrap(errs.CodeWorkspaceInvalid, err, "cannot read %s", BackupPath)
	}
	return raw, true, nil
}
