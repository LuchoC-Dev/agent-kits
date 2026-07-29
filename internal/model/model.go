// Package model defines the canonical, runtime-independent vocabulary of Agent Kits:
// what a resource is, what a source declares, what a plan proposes and what a lockfile
// records. Nothing in this package knows about a concrete runtime or filesystem layout.
package model

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/semver"
)

// ManifestSchemaVersion is the current version of agent-kit.json.
const ManifestSchemaVersion = 1

// LockSchemaVersion is the current version of agent-kits.lock.json.
const LockSchemaVersion = 1

// ManifestFilename is the per-resource manifest recognised in native source layouts.
const ManifestFilename = "agent-kit.json"

// Type enumerates the resource types supported by the MVP (D-020).
type Type string

const (
	TypeSkill    Type = "skill"
	TypeAgent    Type = "agent"
	TypeWorkflow Type = "workflow"
	TypeKit      Type = "kit"
)

// Types lists every supported type in presentation order.
func Types() []Type { return []Type{TypeSkill, TypeAgent, TypeWorkflow, TypeKit} }

// Valid reports whether t is a supported type.
func (t Type) Valid() bool {
	switch t {
	case TypeSkill, TypeAgent, TypeWorkflow, TypeKit:
		return true
	}
	return false
}

// Access describes how a source is reached. It is informative metadata: Git and the
// remote provider enforce authorisation (D-005).
type Access string

const (
	AccessPublic  Access = "public"
	AccessPrivate Access = "private"
)

// Trust is the per-source trust level fixed by D-025.
type Trust string

const (
	TrustTrusted Trust = "trusted"
	TrustReview  Trust = "review"
)

// Dependency is a requirement on another canonical resource.
type Dependency struct {
	ID      ID     `json:"id"`
	Version string `json:"version,omitempty"`
}

// Constraint parses the dependency's version requirement.
func (d Dependency) Constraint() (semver.Constraint, error) {
	return semver.ParseConstraint(d.Version)
}

// UnmarshalJSON accepts both the shorthand and the explicit form, so that
// `"tdd"` and `{"id":"tdd","version":"^0.1.0"}` are both valid.
func (d *Dependency) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, `"`) {
		var raw string
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
		id, err := ParseID(raw)
		if err != nil {
			return err
		}
		d.ID, d.Version = id, ""
		return nil
	}
	var alias struct {
		ID      string `json:"id"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	id, err := ParseID(alias.ID)
	if err != nil {
		return err
	}
	d.ID, d.Version = id, alias.Version
	return nil
}

// Artifact is a declarative integration contract inherited from the legacy catalog's
// produces/consumes fields.
type Artifact struct {
	Artifact    string `json:"artifact"`
	Path        string `json:"path,omitempty"`
	Description string `json:"description,omitempty"`
	Optional    bool   `json:"optional,omitempty"`
}

// Manifest is the declarative description of one resource.
type Manifest struct {
	SchemaVersion int             `json:"schema_version"`
	ID            ID              `json:"id"`
	Type          Type            `json:"type"`
	Name          string          `json:"name,omitempty"`
	Version       string          `json:"version"`
	Description   string          `json:"description,omitempty"`
	Dependencies  []Dependency    `json:"dependencies,omitempty"`
	Files         []string        `json:"files,omitempty"`
	Traits        map[string]bool `json:"traits,omitempty"`
	// Labels carries free-form metadata preserved from the source, such as the legacy
	// catalog's class, invocation and license fields. It is informational: no resolution
	// or installation decision reads it.
	Labels   map[string]string `json:"labels,omitempty"`
	Produces []Artifact        `json:"produces,omitempty"`
	Consumes []Artifact        `json:"consumes,omitempty"`
	Runtimes []string          `json:"runtimes,omitempty"`
}

// Validate checks the manifest's internal consistency. It does not touch the filesystem.
func (m *Manifest) Validate() error {
	if m.SchemaVersion == 0 {
		m.SchemaVersion = ManifestSchemaVersion
	}
	if m.SchemaVersion != ManifestSchemaVersion {
		return errs.New(errs.CodeInvalidManifest,
			"unsupported manifest schema_version %d (expected %d)",
			m.SchemaVersion, ManifestSchemaVersion)
	}
	if _, err := ParseID(string(m.ID)); err != nil {
		return err
	}
	if !m.Type.Valid() {
		return errs.New(errs.CodeInvalidManifest,
			"resource %s declares unsupported type %q", m.ID, m.Type)
	}
	if _, err := semver.Parse(m.Version); err != nil {
		return errs.Wrap(errs.CodeInvalidManifest, err,
			"resource %s declares an invalid version", m.ID)
	}
	seen := map[ID]bool{}
	for _, dep := range m.Dependencies {
		if dep.ID == m.ID {
			return errs.New(errs.CodeInvalidManifest, "resource %s depends on itself", m.ID)
		}
		if seen[dep.ID] {
			return errs.New(errs.CodeInvalidManifest,
				"resource %s declares dependency %s twice", m.ID, dep.ID)
		}
		seen[dep.ID] = true
		if _, err := dep.Constraint(); err != nil {
			return errs.Wrap(errs.CodeInvalidManifest, err,
				"resource %s declares an invalid constraint for %s", m.ID, dep.ID)
		}
	}
	if len(m.Files) == 0 {
		return errs.New(errs.CodeInvalidManifest, "resource %s declares no files", m.ID)
	}
	return nil
}

// SemVersion returns the parsed version, or the zero value when unparseable.
func (m *Manifest) SemVersion() semver.Version {
	v, err := semver.Parse(m.Version)
	if err != nil {
		return semver.Version{}
	}
	return v
}

// DisplayName returns Name when present, falling back to the id.
func (m *Manifest) DisplayName() string {
	if m.Name != "" {
		return m.Name
	}
	return string(m.ID)
}

// SupportsRuntime reports whether the resource declares compatibility with runtime.
// An empty Runtimes list means "every runtime".
func (m *Manifest) SupportsRuntime(runtime string) bool {
	if len(m.Runtimes) == 0 {
		return true
	}
	for _, r := range m.Runtimes {
		if r == runtime {
			return true
		}
	}
	return false
}

// Resource is a manifest plus the provenance needed to read and trace its files.
type Resource struct {
	Manifest

	// Source is the configured source name the resource was loaded from.
	Source string `json:"source"`
	// Root is the absolute directory the manifest's Files are relative to.
	Root string `json:"-"`
	// Commit is the source revision, when the source is a Git checkout.
	Commit string `json:"commit,omitempty"`
	// Access and Trust are inherited from the source.
	Access Access `json:"access,omitempty"`
	Trust  Trust  `json:"trust,omitempty"`
	// Legacy marks resources synthesised from the inherited Markdown catalog (D-026).
	Legacy bool `json:"legacy,omitempty"`
}

// Ref is a short human label used in messages and plan output.
func (r *Resource) Ref() string { return fmt.Sprintf("%s (%s %s)", r.ID, r.Type, r.Version) }

// FileAction classifies what installing a file would do to the project.
type FileAction string

const (
	// ActionCreate writes a file that does not exist yet.
	ActionCreate FileAction = "create"
	// ActionUnchanged leaves an identical file alone.
	ActionUnchanged FileAction = "unchanged"
	// ActionUpdate replaces a managed file the project has not touched.
	ActionUpdate FileAction = "update"
	// ActionAdopt takes over an untracked file whose content already matches.
	ActionAdopt FileAction = "adopt"
	// ActionDivergent marks a managed file modified locally. It blocks (D-023).
	ActionDivergent FileAction = "divergent"
	// ActionRemove deletes a managed file.
	ActionRemove FileAction = "remove"
)

// Writes reports whether the action mutates the filesystem.
func (a FileAction) Writes() bool {
	switch a {
	case ActionCreate, ActionUpdate, ActionAdopt, ActionRemove:
		return true
	}
	return false
}

// FileChange is one planned filesystem operation.
type FileChange struct {
	Path       string     `json:"path"`
	Action     FileAction `json:"action"`
	ResourceID ID         `json:"resource"`
	Checksum   string     `json:"checksum,omitempty"`
	Bytes      int64      `json:"bytes,omitempty"`
	// SourcePath is the absolute file to read. Empty for removals.
	SourcePath string `json:"-"`
}

// PlanResource summarises one resolved resource inside a plan.
type PlanResource struct {
	ID        ID     `json:"id"`
	Type      Type   `json:"type"`
	Version   string `json:"version"`
	Source    string `json:"source"`
	Requested bool   `json:"requested"`
	// State is one of "new", "up-to-date", "update" or "downgrade".
	State string `json:"state"`
	// From is the currently installed version, when there is one.
	From string `json:"from,omitempty"`
}

// Diagnostic is a non-fatal finding attached to a plan or a doctor report.
type Diagnostic struct {
	Code    errs.Code `json:"code"`
	Message string    `json:"message"`
	Path    string    `json:"path,omitempty"`
	Ref     string    `json:"resource,omitempty"`
}

// Plan is the deterministic description of an operation, produced before any write.
type Plan struct {
	Operation string         `json:"operation"`
	Project   string         `json:"project"`
	Runtime   string         `json:"runtime"`
	Requested []string       `json:"requested,omitempty"`
	Resources []PlanResource `json:"resources"`
	Changes   []FileChange   `json:"changes"`
	// Metadata lists the bookkeeping files the operation rewrites — the lockfile and
	// workspace.json. They are tracked separately so they never make an otherwise
	// idempotent plan look non-empty.
	Metadata []FileChange `json:"metadata,omitempty"`
	Warnings []Diagnostic `json:"warnings,omitempty"`
	Blockers []Diagnostic `json:"blockers,omitempty"`
	// Lock is the lockfile the plan would write.
	Lock *Lock `json:"-"`
}

// Empty reports whether applying the plan would change nothing (RF-08).
func (p *Plan) Empty() bool {
	for _, c := range p.Changes {
		if c.Action.Writes() {
			return false
		}
	}
	return true
}

// Blocked reports whether the plan may not be applied.
func (p *Plan) Blocked() bool { return len(p.Blockers) > 0 }

// Counts tallies the changes by action, for compact output.
func (p *Plan) Counts() map[FileAction]int {
	out := map[FileAction]int{}
	for _, c := range p.Changes {
		out[c.Action]++
	}
	return out
}

// Sort orders resources and changes so that the same state always produces the same plan.
func (p *Plan) Sort() {
	sort.SliceStable(p.Resources, func(i, j int) bool {
		if p.Resources[i].Type != p.Resources[j].Type {
			return typeRank(p.Resources[i].Type) < typeRank(p.Resources[j].Type)
		}
		return p.Resources[i].ID < p.Resources[j].ID
	})
	sort.SliceStable(p.Changes, func(i, j int) bool { return p.Changes[i].Path < p.Changes[j].Path })
	sortDiagnostics(p.Warnings)
	sortDiagnostics(p.Blockers)
}

func sortDiagnostics(d []Diagnostic) {
	sort.SliceStable(d, func(i, j int) bool {
		if d[i].Code != d[j].Code {
			return d[i].Code < d[j].Code
		}
		if d[i].Ref != d[j].Ref {
			return d[i].Ref < d[j].Ref
		}
		return d[i].Path < d[j].Path
	})
}

func typeRank(t Type) int {
	for i, candidate := range Types() {
		if candidate == t {
			return i
		}
	}
	return len(Types())
}

// LockFile records one installed file and its checksum at install time.
type LockFile struct {
	Path     string `json:"path"`
	Checksum string `json:"checksum"`
}

// LockResource records one installed resource and its provenance (RF-09).
type LockResource struct {
	ID        ID         `json:"id"`
	Type      Type       `json:"type"`
	Source    string     `json:"source"`
	Version   string     `json:"version"`
	Commit    string     `json:"commit,omitempty"`
	Checksum  string     `json:"checksum"`
	Requested bool       `json:"requested"`
	Files     []LockFile `json:"files"`
}

// Lock is the reproducible record of what a workspace has installed.
type Lock struct {
	SchemaVersion int            `json:"schema_version"`
	Runtime       string         `json:"runtime"`
	GeneratedAt   string         `json:"generated_at"`
	Resources     []LockResource `json:"resources"`
}

// NewLock returns an empty lock for a runtime.
func NewLock(runtime string) *Lock {
	return &Lock{SchemaVersion: LockSchemaVersion, Runtime: runtime, Resources: []LockResource{}}
}

// Validate checks a lock read from disk.
func (l *Lock) Validate() error {
	if l.SchemaVersion != LockSchemaVersion {
		return errs.New(errs.CodeWorkspaceInvalid,
			"unsupported lockfile schema_version %d (expected %d)",
			l.SchemaVersion, LockSchemaVersion)
	}
	seen := map[ID]bool{}
	for _, r := range l.Resources {
		if seen[r.ID] {
			return errs.New(errs.CodeWorkspaceInvalid, "lockfile records %s twice", r.ID)
		}
		seen[r.ID] = true
	}
	return nil
}

// Find returns the recorded resource with the given id.
func (l *Lock) Find(id ID) (LockResource, bool) {
	for _, r := range l.Resources {
		if r.ID == id {
			return r, true
		}
	}
	return LockResource{}, false
}

// FileOwner returns the id of the resource that manages a project-relative path.
func (l *Lock) FileOwner(path string) (ID, string, bool) {
	for _, r := range l.Resources {
		for _, f := range r.Files {
			if f.Path == path {
				return r.ID, f.Checksum, true
			}
		}
	}
	return "", "", false
}

// Upsert replaces or appends a resource record, keeping the list sorted by id.
func (l *Lock) Upsert(r LockResource) {
	for i := range l.Resources {
		if l.Resources[i].ID == r.ID {
			l.Resources[i] = r
			return
		}
	}
	l.Resources = append(l.Resources, r)
	sort.SliceStable(l.Resources, func(i, j int) bool { return l.Resources[i].ID < l.Resources[j].ID })
}

// Delete drops a resource record and reports whether it existed.
func (l *Lock) Delete(id ID) bool {
	for i := range l.Resources {
		if l.Resources[i].ID == id {
			l.Resources = append(l.Resources[:i], l.Resources[i+1:]...)
			return true
		}
	}
	return false
}

// Clone returns a deep copy, so a plan can propose a lock without mutating the current
// one until the plan is applied.
func (l *Lock) Clone() *Lock {
	out := &Lock{
		SchemaVersion: l.SchemaVersion,
		Runtime:       l.Runtime,
		GeneratedAt:   l.GeneratedAt,
		Resources:     make([]LockResource, len(l.Resources)),
	}
	for i, r := range l.Resources {
		copied := r
		copied.Files = make([]LockFile, len(r.Files))
		copy(copied.Files, r.Files)
		out.Resources[i] = copied
	}
	return out
}

// RequestedIDs lists the resources the user asked for explicitly.
func (l *Lock) RequestedIDs() []ID {
	var out []ID
	for _, r := range l.Resources {
		if r.Requested {
			out = append(out, r.ID)
		}
	}
	return out
}
