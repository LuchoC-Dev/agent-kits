// Package plan turns a resolved dependency set into a deterministic description of the
// filesystem changes an operation would make.
//
// Planning never writes. Given the same catalog, project and lockfile it always produces
// the same plan (RF-06), which is what lets an agent show a plan, get approval and then
// apply exactly what was approved.
package plan

import (
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/LuchoC-Dev/agent-kits/internal/adapter"
	"github.com/LuchoC-Dev/agent-kits/internal/catalog"
	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/fsutil"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
	"github.com/LuchoC-Dev/agent-kits/internal/resolve"
	"github.com/LuchoC-Dev/agent-kits/internal/security"
	"github.com/LuchoC-Dev/agent-kits/internal/semver"
)

// Planner computes plans for one project and runtime.
type Planner struct {
	Adapter adapter.Adapter
	// Project is the absolute path of the destination project.
	Project string
	// Lock is the workspace's current lockfile; never nil.
	Lock *model.Lock
	// Limits bounds what a single resource may write.
	Limits security.Limits
	// Force downgrades local divergence from a blocker to a warning (D-023).
	Force bool
	// Now supplies the plan's timestamp; overridable so tests stay deterministic.
	Now func() time.Time
}

// New returns a planner with default limits and clock.
func New(a adapter.Adapter, project string, lock *model.Lock) *Planner {
	return &Planner{
		Adapter: a,
		Project: project,
		Lock:    lock,
		Limits:  security.DefaultLimits(),
		Now:     time.Now,
	}
}

func (p *Planner) timestamp() string {
	now := p.Now
	if now == nil {
		now = time.Now
	}
	return now().UTC().Format(time.RFC3339)
}

// Install plans the installation of a resolved dependency set.
func (p *Planner) Install(result *resolve.Result) (*model.Plan, error) {
	out := &model.Plan{
		Operation: "install",
		Project:   fsutil.ToSlash(p.Project),
		Runtime:   p.Adapter.Name(),
		Requested: result.Refs,
	}
	out.Warnings = append(out.Warnings, result.Diagnostics...)

	// A proposal starts from the project's own identity and migration history: installing
	// something must never drop the state the lockfile already owns.
	proposed := p.Lock.Proposal(p.Adapter.Name())
	proposed.GeneratedAt = p.timestamp()
	// Resources the operation does not touch keep their existing records.
	touched := map[model.ID]bool{}
	for _, res := range result.Order {
		touched[res.ID] = true
	}
	for _, existing := range p.Lock.Resources {
		if !touched[existing.ID] {
			proposed.Upsert(existing)
		}
	}

	// claims maps a destination path to the resource that intends to write it, so two
	// resources targeting the same file are caught before anything is written.
	claims := map[string]claim{}
	for _, existing := range proposed.Resources {
		for _, file := range existing.Files {
			claims[file.Path] = claim{id: existing.ID, name: existing.Name, source: existing.Source}
		}
	}

	for _, res := range result.Order {
		record, changes, err := p.planResource(res, result.Requested[res.ID], claims, out)
		if err != nil {
			return nil, err
		}
		out.Changes = append(out.Changes, changes...)
		proposed.Upsert(record)
		out.Resources = append(out.Resources, p.describe(res, result.Requested[res.ID]))
	}

	// Files a resource used to own but no longer declares are pruned on update.
	out.Changes = append(out.Changes, p.prune(result, claims, out)...)

	out.Lock = proposed
	p.addMetadata(out)
	out.Sort()
	return out, nil
}

// claim records which resource intends to write a destination path. It carries the name as
// well as the identity because a conflict is reported to a person, and two resources that
// collide are told apart by name and source, not by UUID.
type claim struct {
	id     model.ID
	name   string
	source string
}

func (c claim) label() string { return model.Qualify(c.source, c.name) }

// planResource classifies every file of one resource.
func (p *Planner) planResource(
	res *model.Resource, requested bool, claims map[string]claim, out *model.Plan,
) (model.LockResource, []model.FileChange, error) {
	if err := p.Limits.CheckFileCount(res.Name, len(res.Files)); err != nil {
		return model.LockResource{}, nil, err
	}

	record := model.LockResource{
		ID:        res.ID,
		Name:      res.Name,
		Type:      res.Type,
		Source:    res.Source,
		Version:   res.Version,
		Commit:    res.Commit,
		Requested: requested,
	}
	var changes []model.FileChange
	perFile := map[string]string{}

	files := make([]string, len(res.Files))
	copy(files, res.Files)
	sort.Strings(files)

	for _, rel := range files {
		dest, err := p.Adapter.Destination(res, rel)
		if err != nil {
			return model.LockResource{}, nil, err
		}
		absDest, err := security.Contain(p.Project, dest)
		if err != nil {
			return model.LockResource{}, nil, err
		}
		sourcePath := filepath.Join(res.Root, fsutil.FromSlash(rel))

		content, err := os.ReadFile(sourcePath)
		if err != nil {
			return model.LockResource{}, nil, errs.Wrap(errs.CodeSourceUnavailable, err,
				"cannot read %s of %s", rel, res.Name)
		}
		if err := p.Limits.CheckSize(rel, int64(len(content))); err != nil {
			return model.LockResource{}, nil, err
		}
		newSum := fsutil.ChecksumBytes(content)

		p.scan(res, dest, content, out)

		if owner, taken := claims[dest]; taken && owner.id != res.ID {
			out.Blockers = append(out.Blockers, model.Diagnostic{
				Code: errs.CodeDestinationConflict,
				Ref:  res.Name,
				Path: dest,
				Message: "both " + owner.label() + " and " + res.Qualified() +
					" install this path, and they are different resources",
			})
		}
		claims[dest] = claim{id: res.ID, name: res.Name, source: res.Source}

		action, err := p.classify(dest, absDest, newSum, res.ID, out)
		if err != nil {
			return model.LockResource{}, nil, err
		}
		changes = append(changes, model.FileChange{
			Path:       dest,
			Action:     action,
			ResourceID: res.ID,
			Checksum:   newSum,
			Bytes:      int64(len(content)),
			SourcePath: sourcePath,
		})
		record.Files = append(record.Files, model.LockFile{Path: dest, Checksum: newSum})
		perFile[rel] = newSum
	}

	record.Checksum = fsutil.ChecksumTree(perFile)
	sort.SliceStable(record.Files, func(i, j int) bool { return record.Files[i].Path < record.Files[j].Path })
	return record, changes, nil
}

// classify performs the three-way comparison fixed by D-023.
func (p *Planner) classify(
	dest, absDest, newSum string, id model.ID, out *model.Plan,
) (model.FileAction, error) {
	_, lockSum, tracked := p.Lock.FileOwner(dest)

	info, statErr := os.Lstat(absDest)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return model.ActionCreate, nil
		}
		return "", errs.Wrap(errs.CodeInternal, statErr, "cannot inspect %s", dest)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", errs.New(errs.CodeUnsafePath,
			"%s is a symlink; Agent Kits will not write through it", dest)
	}
	if !info.Mode().IsRegular() {
		return "", errs.New(errs.CodeUnsafePath, "%s is not a regular file", dest)
	}

	diskSum, _, err := fsutil.ChecksumFile(absDest)
	if err != nil {
		return "", errs.Wrap(errs.CodeInternal, err, "cannot checksum %s", dest)
	}

	switch {
	case diskSum == newSum:
		// Already converged. Record it if the lockfile does not know about it yet.
		if tracked {
			return model.ActionUnchanged, nil
		}
		return model.ActionAdopt, nil

	case tracked && diskSum == lockSum:
		return model.ActionUpdate, nil

	case tracked:
		return p.divergence(dest, id, "the file was modified after it was installed", out), nil

	default:
		return p.divergence(dest, id, "the project already has an unmanaged file here", out), nil
	}
}

// divergence records a conflict, or converts it to a forced overwrite.
func (p *Planner) divergence(dest string, id model.ID, reason string, out *model.Plan) model.FileAction {
	if p.Force {
		out.Warnings = append(out.Warnings, model.Diagnostic{
			Code:    errs.CodeLocalDivergence,
			Ref:     string(id),
			Path:    dest,
			Message: "overwriting local changes because --force was given: " + reason,
		})
		return model.ActionUpdate
	}
	out.Blockers = append(out.Blockers, model.Diagnostic{
		Code:    errs.CodeLocalDivergence,
		Ref:     string(id),
		Path:    dest,
		Message: reason,
	})
	return model.ActionDivergent
}

// scan reports credential-shaped content. A high-confidence match blocks the operation
// rather than writing the file (D-025).
func (p *Planner) scan(res *model.Resource, dest string, content []byte, out *model.Plan) {
	for _, finding := range security.ScanSecrets(dest, content) {
		diagnostic := model.Diagnostic{
			Code:    errs.CodeUnsafeContent,
			Ref:     res.Name,
			Path:    dest,
			Message: finding.Message(),
		}
		if finding.Severity == security.SeverityBlock {
			out.Blockers = append(out.Blockers, diagnostic)
			continue
		}
		out.Warnings = append(out.Warnings, diagnostic)
	}
}

// prune plans the removal of files a resource owned in the lockfile but no longer ships.
func (p *Planner) prune(
	result *resolve.Result, claims map[string]claim, out *model.Plan,
) []model.FileChange {
	var changes []model.FileChange
	for _, existing := range p.Lock.Resources {
		stillPlanned := false
		for _, res := range result.Order {
			if res.ID == existing.ID {
				stillPlanned = true
				break
			}
		}
		if !stillPlanned {
			continue
		}
		for _, file := range existing.Files {
			if owner, kept := claims[file.Path]; kept && owner.id == existing.ID {
				continue
			}
			change, ok := p.planRemoval(file, existing.ID, out)
			if ok {
				changes = append(changes, change)
			}
		}
	}
	return changes
}

// planRemoval classifies deleting one managed file.
func (p *Planner) planRemoval(
	file model.LockFile, id model.ID, out *model.Plan,
) (model.FileChange, bool) {
	absDest, err := security.Contain(p.Project, file.Path)
	if err != nil {
		out.Warnings = append(out.Warnings, model.Diagnostic{
			Code: errs.CodeUnsafePath, Ref: string(id), Path: file.Path, Message: err.Error(),
		})
		return model.FileChange{}, false
	}
	if !fsutil.Exists(absDest) {
		// Already gone; nothing to plan.
		return model.FileChange{}, false
	}
	diskSum, _, err := fsutil.ChecksumFile(absDest)
	if err != nil {
		out.Warnings = append(out.Warnings, model.Diagnostic{
			Code: errs.CodeInternal, Ref: string(id), Path: file.Path, Message: err.Error(),
		})
		return model.FileChange{}, false
	}
	if diskSum != file.Checksum && !p.Force {
		out.Blockers = append(out.Blockers, model.Diagnostic{
			Code:    errs.CodeLocalDivergence,
			Ref:     string(id),
			Path:    file.Path,
			Message: "refusing to delete a file that was modified after it was installed",
		})
		return model.FileChange{
			Path: file.Path, Action: model.ActionDivergent, ResourceID: id, Checksum: diskSum,
		}, true
	}
	return model.FileChange{
		Path: file.Path, Action: model.ActionRemove, ResourceID: id, Checksum: file.Checksum,
	}, true
}

// describe summarises how a resource's installed state would change.
func (p *Planner) describe(res *model.Resource, requested bool) model.PlanResource {
	entry := model.PlanResource{
		ID:        res.ID,
		Name:      res.Name,
		Type:      res.Type,
		Version:   res.Version,
		Source:    res.Source,
		Requested: requested,
		State:     "new",
	}
	current, installed := p.Lock.Find(res.ID)
	if !installed {
		return entry
	}
	entry.From = current.Version
	switch {
	case current.Version == res.Version:
		entry.State = "up-to-date"
	default:
		before, errBefore := semver.Parse(current.Version)
		after, errAfter := semver.Parse(res.Version)
		if errBefore == nil && errAfter == nil && semver.Compare(after, before) < 0 {
			entry.State = "downgrade"
			return entry
		}
		entry.State = "update"
	}
	return entry
}

// Remove plans the removal of resources, together with dependencies that nothing else
// still needs.
//
// cat may be nil when the source that provided a resource is gone; in that case only the
// named resources are removed and the caller is warned that orphans may remain.
func (p *Planner) Remove(refs []string, cat *catalog.Catalog) (*model.Plan, error) {
	out := &model.Plan{
		Operation: "remove",
		Project:   fsutil.ToSlash(p.Project),
		Runtime:   p.Adapter.Name(),
		Requested: refs,
	}

	targets := map[model.ID]bool{}
	for _, ref := range refs {
		record, err := p.findInstalled(ref, cat)
		if err != nil {
			return nil, err
		}
		targets[record.ID] = true
	}

	// Anything reachable from a requested resource that is not being removed must stay.
	keep := p.reachable(targets, cat, out)

	proposed := p.Lock.Clone()
	proposed.GeneratedAt = p.timestamp()

	for _, existing := range p.Lock.Resources {
		remove := targets[existing.ID] || (!keep[existing.ID] && !existing.Requested)
		if !remove {
			continue
		}
		for _, file := range existing.Files {
			if change, ok := p.planRemoval(file, existing.ID, out); ok {
				out.Changes = append(out.Changes, change)
			}
		}
		proposed.Delete(existing.ID)
		out.Resources = append(out.Resources, model.PlanResource{
			ID: existing.ID, Name: existing.Name, Type: existing.Type, Version: existing.Version,
			Source: existing.Source, Requested: existing.Requested, State: "remove",
			From: existing.Version,
		})
	}

	out.Lock = proposed
	p.addMetadata(out)
	out.Sort()
	return out, nil
}

// findInstalled resolves a reference against the lockfile rather than the catalog, so a
// resource can be removed even after its source disappears.
//
// The lockfile is the authority here, which is why a rename in the catalog cannot make an
// installed resource unremovable: the reference matches the name it was installed under,
// or its identity, which never changes.
func (p *Planner) findInstalled(ref string, cat *catalog.Catalog) (model.LockResource, error) {
	reference, err := model.ParseReference(ref)
	if err != nil {
		return model.LockResource{}, err
	}
	if reference.ID != "" {
		if record, ok := p.Lock.Find(reference.ID); ok {
			return record, nil
		}
		return model.LockResource{}, notInstalled(ref, p.Project)
	}

	var matches []model.LockResource
	for _, record := range p.Lock.Resources {
		if record.Name != reference.Name {
			continue
		}
		if reference.Qualified() && record.Source != reference.Source {
			continue
		}
		matches = append(matches, record)
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return model.LockResource{}, notInstalled(ref, p.Project)
	}
	candidates := make([]string, 0, len(matches))
	for _, match := range matches {
		candidates = append(candidates, model.Qualify(match.Source, match.Name))
	}
	sort.Strings(candidates)
	return model.LockResource{}, errs.New(errs.CodeAmbiguousID,
		"%q matches %d installed resources: %v", ref, len(matches), candidates).
		With("candidates", candidates).
		Hint("qualify the reference as <source>:%s, or use its id", reference.Name)
}

func notInstalled(ref, project string) error {
	return errs.New(errs.CodeNotInstalled, "%q is not installed in this project", ref).
		Hint("run `agent-kits list --project %s`", project)
}

// reachable returns the resources still required by requested resources that survive.
func (p *Planner) reachable(
	removing map[model.ID]bool, cat *catalog.Catalog, out *model.Plan,
) map[model.ID]bool {
	keep := map[model.ID]bool{}
	if cat == nil {
		// Without a catalog the dependency graph is unknown, so every remaining record is
		// preserved: deleting a file we cannot prove is orphaned would be worse.
		for _, existing := range p.Lock.Resources {
			if !removing[existing.ID] {
				keep[existing.ID] = true
			}
		}
		out.Warnings = append(out.Warnings, model.Diagnostic{
			Code:    errs.CodeSourceUnavailable,
			Message: "sources are unavailable, so unused dependencies were kept",
		})
		return keep
	}

	var visit func(id model.ID)
	visit = func(id model.ID) {
		if keep[id] || removing[id] {
			return
		}
		keep[id] = true
		res, ok := cat.Get(id)
		if !ok {
			return
		}
		for _, dep := range res.Dependencies {
			visit(dep.ID)
		}
	}
	for _, existing := range p.Lock.Resources {
		if existing.Requested && !removing[existing.ID] {
			visit(existing.ID)
		}
	}
	return keep
}

// addMetadata records the bookkeeping file the operation rewrites, which since lockfile
// schema 2 is the lockfile alone (D-030).
func (p *Planner) addMetadata(out *model.Plan) {
	if out.Empty() && !out.Blocked() {
		return
	}
	path := p.Adapter.LockPath()
	action := model.ActionCreate
	if abs, err := security.Contain(p.Project, path); err == nil && fsutil.Exists(abs) {
		action = model.ActionUpdate
	}
	out.Metadata = append(out.Metadata, model.FileChange{Path: path, Action: action})
}
