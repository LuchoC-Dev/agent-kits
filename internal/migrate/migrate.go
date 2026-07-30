// Package migrate computes the transition of a project onto lockfile v2, where the
// lockfile becomes the only state Agent Kits owns and workspace.json is retired (D-030,
// D-031).
//
// Computing a migration touches no filesystem: Compute is a pure function of the state a
// caller has already gathered, which is what makes every row of the transition matrix
// (07-cli-only-transition-plan.md §5) testable without a project on disk.
//
// The package is temporary by construction. When the migration window closes it is deleted
// together with internal/workspace/legacy.go.
package migrate

import (
	"bytes"
	"encoding/json"
	"sort"
	"time"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
	"github.com/LuchoC-Dev/agent-kits/internal/workspace"
)

// Origin names where a migration's inputs came from.
const (
	// OriginNone means the project carries no state at all.
	OriginNone = "none"
	// OriginLock means only a lockfile was found, so the migration is a schema upgrade.
	OriginLock = "lockfile"
	// OriginLegacy means an inherited workspace.json took part in the migration.
	OriginLegacy = "workspace.json"
)

// Adoption is one resource a caller found in a workspace that the lockfile does not
// record. The caller owns the catalog and the filesystem, so it supplies the verdict; this
// package only decides what a verdict means for the project's state.
type Adoption struct {
	// Ref is what the workspace called the resource, used in diagnostics.
	Ref string
	// Record is the lockfile entry to add. It is nil when the resource cannot be adopted.
	Record *model.LockResource
	// Blocking diagnostics stop the migration. A resource the workspace declares but that
	// cannot be matched or verified is a contradiction, and contradictions are never
	// resolved by precedence (§5).
	Blocking []model.Diagnostic
	// Warnings are reported and let the migration continue.
	Warnings []model.Diagnostic
}

// State is everything a migration reads, gathered by the caller.
type State struct {
	// Project is the destination, slash-separated, for presentation only.
	Project string
	// Runtime is the adapter's runtime name.
	Runtime string
	// Lock is the project's current lockfile, already upgraded in memory. Never nil.
	Lock *model.Lock
	// LockPath is the project-relative path of the lockfile.
	LockPath string
	// LockSchema is the schema version found on disk, or 0 when there is no lockfile.
	LockSchema int
	// Legacy is the inherited workspace.json, or nil when the project has none.
	Legacy *workspace.Legacy
	// Backup is an existing migration backup and whether one was found.
	Backup        []byte
	BackupPresent bool
	// Adoptions are the resources found in the workspace but absent from the lockfile.
	Adoptions []Adoption
	// Now stamps the migration.
	Now time.Time
	// NewProjectID generates an identity when no inherited one exists. Injectable so a test
	// gets a deterministic plan.
	NewProjectID func() (string, error)
}

// Plan is the deterministic description of a migration, produced before any write.
type Plan struct {
	Operation  string `json:"operation"`
	Project    string `json:"project"`
	Runtime    string `json:"runtime"`
	Origin     string `json:"origin"`
	FromSchema int    `json:"from_schema"`
	ToSchema   int    `json:"to_schema"`
	// Adopted lists the resources the migration records for the first time.
	Adopted []model.PlanResource `json:"adopted,omitempty"`
	// Preserved names the inherited data the new lockfile carries over, so a caller can see
	// that nothing was dropped without diffing two schemas.
	Preserved []string `json:"preserved,omitempty"`
	Backup    string   `json:"backup,omitempty"`
	// Retired lists the files the migration removes.
	Retired  []string           `json:"retired,omitempty"`
	Changes  []model.FileChange `json:"changes"`
	Warnings []model.Diagnostic `json:"warnings,omitempty"`
	Blockers []model.Diagnostic `json:"blockers,omitempty"`

	// Lock is the lockfile the migration would write.
	Lock *model.Lock `json:"-"`
	// LegacyBytes are the exact bytes the backup must contain. Copying these, rather than
	// re-encoding the descriptor, is what makes the backup lossless.
	LegacyBytes []byte `json:"-"`
}

// Blocked reports whether the migration may not be applied.
func (p *Plan) Blocked() bool { return len(p.Blockers) > 0 }

// Empty reports whether applying the plan would change nothing, which is what makes
// repeating a migration a no-op.
func (p *Plan) Empty() bool {
	for _, change := range p.Changes {
		if change.Action.Writes() {
			return false
		}
	}
	return true
}

// Compute decides what migrating a project would do.
func Compute(state State) (*Plan, error) {
	out := &Plan{
		Operation: "migrate",
		Project:   state.Project,
		Runtime:   state.Runtime,
		Origin:    OriginNone,
		ToSchema:  model.LockSchemaVersion,
		Changes:   []model.FileChange{},
	}
	hasLock := state.LockSchema > 0
	hasLegacy := state.Legacy != nil
	out.FromSchema = state.LockSchema

	switch {
	case !hasLock && !hasLegacy:
		// Nothing to migrate. A project without state is not made to acquire any.
		return out, nil
	case hasLegacy:
		out.Origin = OriginLegacy
	default:
		out.Origin = OriginLock
	}

	proposed := state.Lock.Clone()
	proposed.SchemaVersion = model.LockSchemaVersion
	proposed.Runtime = state.Runtime
	proposed.GeneratedAt = stamp(state.Now)

	if hasLegacy {
		if err := absorb(out, proposed, state); err != nil {
			return nil, err
		}
	}
	if err := ensureIdentity(out, proposed, state); err != nil {
		return nil, err
	}

	out.Lock = proposed
	planChanges(out, state, proposed, hasLock, hasLegacy)
	sortDiagnostics(out.Warnings)
	sortDiagnostics(out.Blockers)
	return out, nil
}

// absorb folds an inherited workspace.json into the proposed lockfile.
func absorb(out *Plan, proposed *model.Lock, state State) error {
	descriptor := state.Legacy.Descriptor
	out.LegacyBytes = state.Legacy.Raw

	if state.BackupPresent && !bytes.Equal(state.Backup, state.Legacy.Raw) {
		// Overwriting a backup would destroy the only untouched copy of an earlier state.
		block(out, errs.CodeIntegrityMismatch, "", workspace.BackupPath,
			"a different backup already exists; move it aside before migrating again")
	}

	adoptIdentity(out, proposed, descriptor)
	adoptStack(out, proposed, descriptor)
	adoptDisciplines(out, proposed, descriptor)

	if descriptor.Runtime != "" && descriptor.Runtime != state.Runtime {
		// The runtime is derivable from the environment and every supported adapter shares
		// the .agents layout, so it is recalculated rather than treated as a contradiction.
		// The inherited value survives byte for byte in the backup.
		warn(out, errs.CodeRuntimeUnsupported, "", workspace.LegacyPath,
			"the workspace was initialised for runtime "+descriptor.Runtime+
				"; the lockfile records "+state.Runtime)
	}

	adoptResources(out, proposed, state)
	proposed.Migration = record(state, descriptor)
	out.Preserved = append(out.Preserved, "history")
	out.Backup = workspace.BackupPath
	out.Retired = append(out.Retired, workspace.LegacyPath)
	sort.Strings(out.Preserved)
	return nil
}

// adoptIdentity carries the project's identity across, or blocks when the two files claim
// different ones.
func adoptIdentity(out *Plan, proposed *model.Lock, descriptor *workspace.Descriptor) {
	if descriptor.ID == "" {
		return
	}
	if proposed.Project == nil {
		proposed.Project = &model.LockProject{ID: descriptor.ID, CreatedAt: descriptor.CreatedAt}
		out.Preserved = append(out.Preserved, "project.id", "project.created_at")
		return
	}
	if proposed.Project.ID != descriptor.ID {
		block(out, errs.CodeWorkspaceInvalid, "", workspace.LegacyPath,
			"the lockfile and the workspace declare different project ids ("+
				proposed.Project.ID+" and "+descriptor.ID+")")
		return
	}
	if descriptor.CreatedAt != "" && proposed.Project.CreatedAt != "" &&
		proposed.Project.CreatedAt != descriptor.CreatedAt {
		block(out, errs.CodeWorkspaceInvalid, "", workspace.LegacyPath,
			"the lockfile and the workspace disagree on when the project was created")
		return
	}
	if proposed.Project.CreatedAt == "" {
		proposed.Project.CreatedAt = descriptor.CreatedAt
		out.Preserved = append(out.Preserved, "project.created_at")
	}
}

func adoptStack(out *Plan, proposed *model.Lock, descriptor *workspace.Descriptor) {
	if descriptor.Stack == nil || proposed.Project == nil {
		return
	}
	inherited := &model.LockStack{
		Detected:   append([]string(nil), descriptor.Stack.Detected...),
		Source:     descriptor.Stack.Source,
		Confidence: descriptor.Stack.Confidence,
	}
	if proposed.Project.Stack == nil {
		proposed.Project.Stack = inherited
		out.Preserved = append(out.Preserved, "project.stack")
		return
	}
	if !sameStack(proposed.Project.Stack, inherited) {
		block(out, errs.CodeWorkspaceInvalid, "", workspace.LegacyPath,
			"the lockfile and the workspace declare different project stacks")
	}
}

// adoptDisciplines merges the two lists. Disciplines are additive rather than exclusive, so
// a union preserves both without choosing a winner.
func adoptDisciplines(out *Plan, proposed *model.Lock, descriptor *workspace.Descriptor) {
	if len(descriptor.Disciplines) == 0 || proposed.Project == nil {
		return
	}
	seen := map[string]bool{}
	for _, name := range proposed.Project.Disciplines {
		seen[name] = true
	}
	added := false
	for _, name := range descriptor.Disciplines {
		if !seen[name] {
			seen[name] = true
			proposed.Project.Disciplines = append(proposed.Project.Disciplines, name)
			added = true
		}
	}
	sort.Strings(proposed.Project.Disciplines)
	if added {
		out.Preserved = append(out.Preserved, "project.disciplines")
	}
}

// adoptResources records the resources the workspace holds but the lockfile does not.
func adoptResources(out *Plan, proposed *model.Lock, state State) {
	installedAt := inheritedTimestamps(state.Legacy.Descriptor)
	for _, adoption := range state.Adoptions {
		out.Warnings = append(out.Warnings, adoption.Warnings...)
		out.Blockers = append(out.Blockers, adoption.Blocking...)
		if adoption.Record == nil {
			continue
		}
		if _, already := proposed.Find(adoption.Record.ID); already {
			// The lockfile already proves ownership; it is never overwritten by the workspace.
			continue
		}
		entry := *adoption.Record
		if entry.InstalledAt == "" {
			// The inherited descriptor recorded install times by name, which is what it had.
			if when, ok := installedAt[entry.Name]; ok {
				entry.InstalledAt = when
			} else if when, ok := installedAt[adoption.Ref]; ok {
				entry.InstalledAt = when
			} else {
				entry.InstalledAt = stamp(state.Now)
			}
		}
		proposed.Upsert(entry)
		out.Adopted = append(out.Adopted, model.PlanResource{
			ID: entry.ID, Name: entry.Name, Type: entry.Type, Version: entry.Version,
			Source: entry.Source, Requested: entry.Requested, State: "adopt",
		})
	}
	if len(out.Adopted) > 0 {
		out.Preserved = append(out.Preserved, "resources.installed_at")
		sort.SliceStable(out.Adopted, func(i, j int) bool {
			return out.Adopted[i].Name < out.Adopted[j].Name
		})
	}
}

// inheritedTimestamps indexes the install times the descriptor recorded.
func inheritedTimestamps(descriptor *workspace.Descriptor) map[string]string {
	out := map[string]string{}
	for _, group := range [][]workspace.Entry{descriptor.Skills, descriptor.Agents} {
		for _, entry := range group {
			if entry.InstalledAt != "" {
				out[entry.ID] = entry.InstalledAt
			}
		}
	}
	if descriptor.Pack != nil && descriptor.Pack.Name != "" && descriptor.Pack.InstalledAt != "" {
		out[descriptor.Pack.Name] = descriptor.Pack.InstalledAt
	}
	return out
}

// record builds the migration entry: the inherited data that has no operational home in
// the new schema but must not disappear.
func record(state State, descriptor *workspace.Descriptor) *model.LockMigration {
	out := &model.LockMigration{
		Source:              OriginLegacy,
		SourceSchemaVersion: descriptor.SchemaVersion,
		MigratedAt:          stamp(state.Now),
		LegacyUpdatedAt:     descriptor.UpdatedAt,
		LegacySystemVersion: descriptor.SystemVersion,
		LegacyStructure:     append([]string(nil), descriptor.Structure...),
		Backup:              workspace.BackupPath,
	}
	if pack, ok := descriptor.RawField("pack"); ok {
		out.LegacyPack = pack
	}
	if flags, ok := descriptor.RawField("flags"); ok {
		out.LegacyFlags = flags
	}
	if extra := descriptor.Extra(); len(extra) > 0 {
		out.Extra = extra
	}
	return out
}

// ensureIdentity gives a migrated project an identity when there was none to inherit.
func ensureIdentity(out *Plan, proposed *model.Lock, state State) error {
	if proposed.Project != nil {
		if proposed.Project.CreatedAt == "" {
			proposed.Project.CreatedAt = stamp(state.Now)
		}
		return nil
	}
	newID := state.NewProjectID
	if newID == nil {
		newID = model.NewProjectID
	}
	id, err := newID()
	if err != nil {
		return err
	}
	proposed.EnsureProject(id, stamp(state.Now))
	out.Preserved = append(out.Preserved, "project.id")
	sort.Strings(out.Preserved)
	return nil
}

// planChanges lists the files the migration writes and removes.
//
// The order is the order of application: the lockfile and the backup must both exist
// before workspace.json is retired, so a failure can never leave a project with neither.
func planChanges(out *Plan, state State, proposed *model.Lock, hasLock, hasLegacy bool) {
	lockAction := model.ActionCreate
	if hasLock {
		lockAction = model.ActionUpdate
		if state.LockSchema == model.LockSchemaVersion && equivalent(state.Lock, proposed) {
			// The lockfile already says everything the migration would say. Keeping its
			// timestamp too is what makes a repeated migration touch no file at all.
			proposed.GeneratedAt = state.Lock.GeneratedAt
			lockAction = model.ActionUnchanged
		}
	}
	out.Changes = append(out.Changes, model.FileChange{
		Path: state.LockPath, Action: lockAction,
	})

	if !hasLegacy {
		return
	}
	backupAction := model.ActionCreate
	if state.BackupPresent && bytes.Equal(state.Backup, state.Legacy.Raw) {
		// The backup already holds exactly these bytes: an interrupted migration is being
		// completed, not repeated.
		backupAction = model.ActionUnchanged
	}
	out.Changes = append(out.Changes,
		model.FileChange{Path: workspace.BackupPath, Action: backupAction},
		model.FileChange{Path: workspace.LegacyPath, Action: model.ActionRemove},
	)
}

// equivalent reports whether two lockfiles describe the same state, ignoring the moment
// they were generated.
func equivalent(current, proposed *model.Lock) bool {
	a, b := current.Clone(), proposed.Clone()
	a.GeneratedAt, b.GeneratedAt = "", ""
	encodedA, errA := json.Marshal(a)
	encodedB, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return false
	}
	return bytes.Equal(encodedA, encodedB)
}

func sameStack(a, b *model.LockStack) bool {
	if a.Source != b.Source || a.Confidence != b.Confidence || len(a.Detected) != len(b.Detected) {
		return false
	}
	for i := range a.Detected {
		if a.Detected[i] != b.Detected[i] {
			return false
		}
	}
	return true
}

func stamp(now time.Time) string { return now.UTC().Format(time.RFC3339) }

func warn(out *Plan, code errs.Code, ref, path, message string) {
	out.Warnings = append(out.Warnings,
		model.Diagnostic{Code: code, Ref: ref, Path: path, Message: message})
}

func block(out *Plan, code errs.Code, ref, path, message string) {
	out.Blockers = append(out.Blockers,
		model.Diagnostic{Code: code, Ref: ref, Path: path, Message: message})
}

func sortDiagnostics(list []model.Diagnostic) {
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].Code != list[j].Code {
			return list[i].Code < list[j].Code
		}
		if list[i].Ref != list[j].Ref {
			return list[i].Ref < list[j].Ref
		}
		return list[i].Path < list[j].Path
	})
}
