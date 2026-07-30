// Package model defines the canonical, runtime-independent vocabulary of Agent Kits:
// what a resource is, what a source declares, what a plan proposes and what a lockfile
// records. Nothing in this package knows about a concrete runtime or filesystem layout.
package model

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/semver"
)

// ManifestSchemaVersion is the current version of agent-kit.json.
const ManifestSchemaVersion = 1

// LockSchemaVersion is the version of agent-kits.lock.json this build writes. Version 2
// makes the lockfile the single source of truth of a project, absorbing the identity and
// history that workspace.json used to hold (D-030).
const LockSchemaVersion = 2

// LockSchemaVersionLegacy is the older lockfile this build still reads. It is upgraded in
// memory on load, so every write produces LockSchemaVersion.
const LockSchemaVersionLegacy = 1

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

// Dependency is a requirement on another resource, addressed by identity.
//
// Name is informative: a list of UUIDs is unreadable, so a manifest may record the name a
// dependency had when it was written. The catalog verifies it against the resolved
// resource, which turns a stale copy into a reported inconsistency instead of a silent
// lie. Resolution never uses it.
type Dependency struct {
	ID      ID     `json:"id"`
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

// Constraint parses the dependency's version requirement.
func (d Dependency) Constraint() (semver.Constraint, error) {
	return semver.ParseConstraint(d.Version)
}

// UnmarshalJSON accepts the shorthand `"<uuid>"` and the explicit form
// `{"id":"<uuid>","name":"tdd","version":"^1.0.0"}`.
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
		d.ID, d.Name, d.Version = id, "", ""
		return nil
	}
	var alias struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	id, err := ParseID(alias.ID)
	if err != nil {
		return err
	}
	d.ID, d.Name, d.Version = id, alias.Name, alias.Version
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
	SchemaVersion int `json:"schema_version"`
	// ID is the resource's identity and never changes (D-035).
	ID   ID   `json:"id"`
	Type Type `json:"type"`
	// Name is the install name: what a caller types and where the resource lands on disk.
	// It is unique within a source and may be changed by its author (D-036).
	Name string `json:"name"`
	// Title is the human label. It is presentation only.
	Title           string          `json:"title,omitempty"`
	Version         string          `json:"version"`
	Description     string          `json:"description,omitempty"`
	Dependencies    []Dependency    `json:"dependencies,omitempty"`
	Files           []string        `json:"files,omitempty"`
	Traits          map[string]bool `json:"traits,omitempty"`
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
		return errs.Wrap(errs.CodeInvalidManifest, err, "resource %q has no valid id", m.Name)
	}
	if _, err := ParseName(m.Name); err != nil {
		return errs.Wrap(errs.CodeInvalidManifest, err, "resource %s has no valid name", m.ID.Short())
	}
	if !m.Type.Valid() {
		return errs.New(errs.CodeInvalidManifest,
			"resource %s declares unsupported type %q", m.Name, m.Type)
	}
	if _, err := semver.Parse(m.Version); err != nil {
		return errs.Wrap(errs.CodeInvalidManifest, err,
			"resource %s declares an invalid version", m.Name)
	}
	seen := map[ID]bool{}
	for _, dep := range m.Dependencies {
		if dep.ID == m.ID {
			return errs.New(errs.CodeInvalidManifest, "resource %s depends on itself", m.Name)
		}
		if seen[dep.ID] {
			return errs.New(errs.CodeInvalidManifest,
				"resource %s declares dependency %s twice", m.Name, dep.Label())
		}
		seen[dep.ID] = true
		if _, err := dep.Constraint(); err != nil {
			return errs.Wrap(errs.CodeInvalidManifest, err,
				"resource %s declares an invalid constraint for %s", m.Name, dep.Label())
		}
	}
	if len(m.Files) == 0 {
		return errs.New(errs.CodeInvalidManifest, "resource %s declares no files", m.Name)
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

// DisplayName returns the human label when there is one, falling back to the install name.
func (m *Manifest) DisplayName() string {
	if m.Title != "" {
		return m.Title
	}
	return m.Name
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
}

// Ref is a short human label used in messages and plan output.
func (r *Resource) Ref() string { return fmt.Sprintf("%s (%s %s)", r.Name, r.Type, r.Version) }

// Qualified renders the resource as `<source>:<name>`, which always identifies it
// unambiguously among the configured sources.
func (r *Resource) Qualified() string { return Qualify(r.Source, r.Name) }

// Label renders a dependency for a message: its recorded name when it has one, and the
// abbreviated identity otherwise.
func (d Dependency) Label() string {
	if d.Name != "" {
		return d.Name
	}
	return d.ID.Short()
}

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
	Name      string `json:"name"`
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
	// Metadata lists the bookkeeping files the operation rewrites. They are tracked
	// separately so they never make an otherwise idempotent plan look non-empty.
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
	// Ordering is by name, not identity: a plan is read by a person, and a UUID order is
	// arbitrary to them. The identity breaks ties so the order stays deterministic.
	sort.SliceStable(p.Resources, func(i, j int) bool {
		if p.Resources[i].Type != p.Resources[j].Type {
			return typeRank(p.Resources[i].Type) < typeRank(p.Resources[j].Type)
		}
		if p.Resources[i].Name != p.Resources[j].Name {
			return p.Resources[i].Name < p.Resources[j].Name
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
	// ID is the identity: it survives a rename, a change of kit and a publication.
	ID ID `json:"id"`
	// Name is the install name the resource had when it was installed. It is what the
	// project shows and what a caller types; a later rename in the catalog shows up as a
	// difference here, not as a different resource.
	Name      string `json:"name"`
	Type      Type   `json:"type"`
	Source    string `json:"source"`
	Version   string `json:"version"`
	Commit    string `json:"commit,omitempty"`
	Checksum  string `json:"checksum"`
	Requested bool   `json:"requested"`
	// InstalledAt preserves when the resource entered the project. A migration carries the
	// inherited timestamp over, so adopting a workspace does not rewrite its history.
	InstalledAt string     `json:"installed_at,omitempty"`
	Files       []LockFile `json:"files"`
}

// LockStack is the project's detected technology stack.
type LockStack struct {
	Detected   []string `json:"detected"`
	Source     string   `json:"source"`
	Confidence string   `json:"confidence"`
}

// LockProject is the stable identity of the project the lockfile describes.
//
// ID and CreatedAt never change once written: they survive migrations, updates and
// removals, so a project keeps one identity for its whole life (07-cli-only-transition
// -plan.md §4).
type LockProject struct {
	ID        string     `json:"id"`
	CreatedAt string     `json:"created_at"`
	Stack     *LockStack `json:"stack,omitempty"`
	// Disciplines is explicit rather than derived: it can affect behaviour, so a migration
	// preserves it instead of recomputing it.
	Disciplines []string `json:"disciplines,omitempty"`
}

// LockMigration records that this lockfile absorbed an inherited workspace.json.
//
// The migration that wrote it no longer exists (D-041), but the record does, and it is
// read and written back untouched. Dropping it would delete the only trace of what a
// project used to be — including fields no version of Agent Kits ever understood — which
// is exactly what the migration was designed to avoid. It is history now, and history is
// carried, not collected.
type LockMigration struct {
	Source              string `json:"source"`
	SourceSchemaVersion int    `json:"source_schema_version,omitempty"`
	MigratedAt          string `json:"migrated_at"`
	LegacyUpdatedAt     string `json:"legacy_updated_at,omitempty"`
	LegacySystemVersion string `json:"legacy_system_version,omitempty"`
	// Legacy* fields keep the inherited values as raw JSON, so a field this build does not
	// understand is still preserved byte for byte.
	LegacyPack      json.RawMessage `json:"legacy_pack,omitempty"`
	LegacyFlags     json.RawMessage `json:"legacy_flags,omitempty"`
	LegacyStructure []string        `json:"legacy_structure,omitempty"`
	// Extra holds the fields of workspace.json this build does not manage at all.
	Extra map[string]json.RawMessage `json:"extra,omitempty"`
	// Backup is the project-relative path of the byte-for-byte copy of workspace.json.
	Backup string `json:"backup,omitempty"`
}

// Lock is the reproducible record of what a project has installed, and — from schema
// version 2 on — the only state file Agent Kits owns (D-030).
type Lock struct {
	SchemaVersion int            `json:"schema_version"`
	Project       *LockProject   `json:"project,omitempty"`
	Runtime       string         `json:"runtime"`
	GeneratedAt   string         `json:"generated_at"`
	Resources     []LockResource `json:"resources"`
	Migration     *LockMigration `json:"migration,omitempty"`
}

// NewLock returns an empty lock for a runtime.
func NewLock(runtime string) *Lock {
	return &Lock{SchemaVersion: LockSchemaVersion, Runtime: runtime, Resources: []LockResource{}}
}

// Validate checks a lock read from disk. Both the current and the legacy schema are
// accepted, because a project written by an earlier build must remain readable; Upgrade
// converts the legacy shape before anything is written back.
func (l *Lock) Validate() error {
	if l.SchemaVersion != LockSchemaVersion && l.SchemaVersion != LockSchemaVersionLegacy {
		return errs.New(errs.CodeWorkspaceInvalid,
			"unsupported lockfile schema_version %d (expected %d or %d)",
			l.SchemaVersion, LockSchemaVersionLegacy, LockSchemaVersion)
	}
	seen := map[ID]bool{}
	for _, r := range l.Resources {
		if seen[r.ID] {
			return errs.New(errs.CodeWorkspaceInvalid, "lockfile records %s twice", r.ID)
		}
		seen[r.ID] = true
	}
	if l.Project != nil && l.Project.ID == "" {
		return errs.New(errs.CodeWorkspaceInvalid, "lockfile declares a project without an id")
	}
	return nil
}

// Legacy reports whether the lock was read in the superseded schema.
func (l *Lock) Legacy() bool { return l.SchemaVersion == LockSchemaVersionLegacy }

// Upgrade converts a legacy lock to the current schema in memory.
//
// The conversion is deterministic and adds nothing: a v1 lock carries no project identity,
// so the upgraded lock has none either. Assigning one is an operation's job, not the
// reader's, which keeps reading a lockfile free of side effects.
func (l *Lock) Upgrade() {
	if l.SchemaVersion == LockSchemaVersionLegacy {
		l.SchemaVersion = LockSchemaVersion
	}
	if l.Resources == nil {
		l.Resources = []LockResource{}
	}
}

// EnsureProject assigns the project identity when the lock does not have one yet, and
// returns it. An existing identity is never replaced.
func (l *Lock) EnsureProject(id string, createdAt string) *LockProject {
	if l.Project == nil {
		l.Project = &LockProject{ID: id, CreatedAt: createdAt}
	}
	return l.Project
}

// Proposal returns a lock that keeps this one's identity, migration record and runtime but
// carries no resources. Planning builds on it so an install never drops project state.
func (l *Lock) Proposal(runtime string) *Lock {
	out := NewLock(runtime)
	if l == nil {
		return out
	}
	out.Project = cloneProject(l.Project)
	out.Migration = cloneMigration(l.Migration)
	return out
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
	l.sort()
}

// sort keeps the lockfile ordered by name, so a diff between two lockfiles reads like a
// diff of what the project has rather than of opaque identities.
func (l *Lock) sort() {
	sort.SliceStable(l.Resources, func(i, j int) bool {
		if l.Resources[i].Name != l.Resources[j].Name {
			return l.Resources[i].Name < l.Resources[j].Name
		}
		return l.Resources[i].ID < l.Resources[j].ID
	})
}

// FindByName returns the recorded resource installed under a name.
func (l *Lock) FindByName(name string) (LockResource, bool) {
	for _, r := range l.Resources {
		if r.Name == name {
			return r, true
		}
	}
	return LockResource{}, false
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
		Project:       cloneProject(l.Project),
		Runtime:       l.Runtime,
		GeneratedAt:   l.GeneratedAt,
		Resources:     make([]LockResource, len(l.Resources)),
		Migration:     cloneMigration(l.Migration),
	}
	for i, r := range l.Resources {
		copied := r
		copied.Files = make([]LockFile, len(r.Files))
		copy(copied.Files, r.Files)
		out.Resources[i] = copied
	}
	return out
}

func cloneProject(in *LockProject) *LockProject {
	if in == nil {
		return nil
	}
	out := *in
	if in.Stack != nil {
		stack := *in.Stack
		stack.Detected = append([]string(nil), in.Stack.Detected...)
		out.Stack = &stack
	}
	out.Disciplines = append([]string(nil), in.Disciplines...)
	return &out
}

func cloneMigration(in *LockMigration) *LockMigration {
	if in == nil {
		return nil
	}
	out := *in
	out.LegacyPack = append(json.RawMessage(nil), in.LegacyPack...)
	out.LegacyFlags = append(json.RawMessage(nil), in.LegacyFlags...)
	out.LegacyStructure = append([]string(nil), in.LegacyStructure...)
	if in.Extra != nil {
		out.Extra = make(map[string]json.RawMessage, len(in.Extra))
		for key, value := range in.Extra {
			out.Extra[key] = append(json.RawMessage(nil), value...)
		}
	}
	return &out
}

// NewProjectID generates the random identifier of a project, in the UUID v4 form the
// inherited workspace.json used, so a migrated project can keep the id it already had.
func NewProjectID() (string, error) {
	id, err := newUUID()
	if err != nil {
		return "", errs.Wrap(errs.CodeInternal, err, "cannot generate a project id")
	}
	return id, nil
}

// newUUID generates a version 4 UUID. It is the identity of both a project and a resource.
func newUUID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	buf[6] = (buf[6] & 0x0f) | 0x40 // version 4
	buf[8] = (buf[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16]), nil
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
