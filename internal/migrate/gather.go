package migrate

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/LuchoC-Dev/agent-kits/internal/adapter"
	"github.com/LuchoC-Dev/agent-kits/internal/catalog"
	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/fsutil"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
	"github.com/LuchoC-Dev/agent-kits/internal/security"
	"github.com/LuchoC-Dev/agent-kits/internal/workspace"
)

// Input describes the migration a caller wants computed.
type Input struct {
	// Project is the absolute path of the destination project.
	Project string
	Adapter adapter.Adapter
	// Catalog identifies the resources an unlocked workspace already contains. It is only
	// required when the project still has a workspace.json.
	Catalog *catalog.Catalog
	Now     func() time.Time
	// NewProjectID is injectable so a test gets a deterministic plan.
	NewProjectID func() (string, error)
}

// Gather reads the project's current state and computes the migration plan. It writes
// nothing: the returned plan is a proposal a caller may render, approve and then Apply.
func Gather(in Input) (*Plan, error) {
	now := in.Now
	if now == nil {
		now = time.Now
	}
	lock, schema, err := workspace.LoadLockDetail(in.Project, in.Adapter)
	if err != nil {
		return nil, err
	}
	legacy, present, err := workspace.LoadLegacy(in.Project)
	if err != nil {
		return nil, err
	}
	backup, backupPresent, err := workspace.LoadBackup(in.Project)
	if err != nil {
		return nil, err
	}

	state := State{
		Project:       fsutil.ToSlash(in.Project),
		Runtime:       in.Adapter.Name(),
		Lock:          lock,
		LockPath:      in.Adapter.LockPath(),
		LockSchema:    schema,
		Backup:        backup,
		BackupPresent: backupPresent,
		Now:           now().UTC(),
		NewProjectID:  in.NewProjectID,
	}
	if present {
		state.Legacy = legacy
		if in.Catalog == nil {
			return nil, errs.New(errs.CodeSourceUnavailable,
				"migrating a workspace needs a readable catalog to identify what it contains")
		}
		state.Adoptions = adoptions(in, lock, legacy.Descriptor)
	}
	return Compute(state)
}

// adoptions verifies every resource the workspace appears to contain against the catalog.
//
// Verification is conservative and fail-closed: a resource is adopted only when each of
// its files matches the catalog byte for byte. Recording a lockfile entry for content that
// does not match would make a later update silently overwrite the difference.
func adoptions(in Input, lock *model.Lock, descriptor *workspace.Descriptor) []Adoption {
	var out []Adoption
	kits := declaredKits(descriptor)
	for _, candidate := range candidates(in.Project, descriptor) {
		res, err := in.Catalog.Lookup(candidate.ref)
		if err != nil {
			if errs.Is(err, errs.CodeAmbiguousID) {
				// A bare workflow name can belong to more than one kit; the workspace's own
				// composition record is the tie-breaker.
				res, err = disambiguate(in.Catalog, candidate.ref, kits)
			}
			if err != nil {
				out = append(out, unresolved(candidate, err))
				continue
			}
		}
		if _, already := lock.Find(res.ID); already {
			continue
		}
		out = append(out, verify(in, candidate, res))
	}
	return out
}

// unresolved reports a reference the catalog cannot identify.
//
// A resource the workspace *declares* is a contradiction the migration cannot resolve, so
// it blocks. A file merely found while scanning the managed directories is reported and
// left where it is: it is content the project owns, not a claim about state.
func unresolved(candidate importCandidate, err error) Adoption {
	diagnostic := model.Diagnostic{
		Code: errs.CodeOf(err), Ref: candidate.ref,
		Message: "not adopted: " + err.Error(),
	}
	if candidate.declared {
		return Adoption{Ref: candidate.ref, Blocking: []model.Diagnostic{diagnostic}}
	}
	return Adoption{Ref: candidate.ref, Warnings: []model.Diagnostic{diagnostic}}
}

// verify compares a resource's catalog content against what the project holds.
func verify(in Input, candidate importCandidate, res *model.Resource) Adoption {
	out := Adoption{Ref: candidate.ref}
	record := model.LockResource{
		ID: res.ID, Name: res.Name, Type: res.Type, Source: res.Source,
		Version: res.Version, Commit: res.Commit, Requested: candidate.requested,
	}
	perFile := map[string]string{}

	fail := func(code errs.Code, path, message string) Adoption {
		out.Blocking = append(out.Blocking, model.Diagnostic{
			Code: code, Ref: res.Name, Path: path, Message: message,
		})
		return out
	}

	for _, rel := range res.Files {
		dest, err := in.Adapter.Destination(res, rel)
		if err != nil {
			return fail(errs.CodeOf(err), "", err.Error())
		}
		abs, err := security.Contain(in.Project, dest)
		if err != nil {
			return fail(errs.CodeUnsafePath, dest, err.Error())
		}
		if !fsutil.IsRegular(abs) {
			message := "not adopted: the workspace does not contain this file"
			if !candidate.declared {
				out.Warnings = append(out.Warnings, model.Diagnostic{
					Code: errs.CodeNotInstalled, Ref: res.Name, Path: dest, Message: message,
				})
				return Adoption{Ref: candidate.ref, Warnings: out.Warnings}
			}
			return fail(errs.CodeNotInstalled, dest, message)
		}
		diskSum, _, err := fsutil.ChecksumFile(abs)
		if err != nil {
			return fail(errs.CodeInternal, dest, err.Error())
		}
		catalogSum, err := checksumSource(res, rel)
		if err != nil {
			return fail(errs.CodeOf(err), dest, err.Error())
		}
		if diskSum != catalogSum {
			// A managed file that diverges always blocks: the migration has no way to record
			// it truthfully, and there is no --force that could make the difference safe.
			return fail(errs.CodeLocalDivergence, dest,
				"not adopted: the installed file differs from the catalog")
		}
		record.Files = append(record.Files, model.LockFile{Path: dest, Checksum: diskSum})
		perFile[rel] = diskSum
	}
	if len(record.Files) == 0 {
		return out
	}
	record.Checksum = fsutil.ChecksumTree(perFile)
	sort.SliceStable(record.Files, func(i, j int) bool {
		return record.Files[i].Path < record.Files[j].Path
	})
	out.Record = &record
	return out
}

func checksumSource(res *model.Resource, rel string) (string, error) {
	if err := security.CheckRelPath(rel); err != nil {
		return "", err
	}
	content, err := os.ReadFile(filepath.Join(res.Root, fsutil.FromSlash(rel)))
	if err != nil {
		return "", errs.Wrap(errs.CodeSourceUnavailable, err, "cannot read %s of %s", rel, res.Name)
	}
	return fsutil.ChecksumBytes(content), nil
}

// importCandidate is a resource reference discovered in an existing workspace.
type importCandidate struct {
	ref string
	// declared reports that workspace.json names the resource, as opposed to it having been
	// found while scanning the managed directories.
	declared  bool
	requested bool
}

// candidates lists what an existing workspace appears to contain: the resources
// workspace.json records, plus anything found in the managed directories.
func candidates(project string, descriptor *workspace.Descriptor) []importCandidate {
	seen := map[string]bool{}
	var out []importCandidate
	add := func(ref string, declared, requested bool) {
		ref = strings.TrimSpace(ref)
		if ref == "" || seen[ref] {
			return
		}
		seen[ref] = true
		out = append(out, importCandidate{ref: ref, declared: declared, requested: requested})
	}

	if descriptor.Pack != nil && descriptor.Pack.Name != "" &&
		!strings.HasPrefix(descriptor.Pack.Name, "custom") {
		add(descriptor.Pack.Name, true, true)
	}
	for _, entry := range descriptor.Skills {
		add(entry.ID, true, false)
	}
	for _, entry := range descriptor.Agents {
		add(entry.ID, true, false)
	}
	for _, dir := range dirEntries(project, adapter.WorkspaceDir+"/packs") {
		add(dir, false, true)
	}
	for _, name := range markdownStems(project, adapter.WorkspaceDir+"/agents") {
		add(name, false, false)
	}
	for _, name := range markdownStems(project, adapter.WorkspaceDir+"/workflows") {
		add(name, false, false)
	}
	for _, dir := range dirEntries(project, adapter.WorkspaceDir+"/skills") {
		add(dir, false, false)
	}
	return out
}

// declaredKits lists the kit names the workspace claims, used to disambiguate names that
// several kits own.
func declaredKits(descriptor *workspace.Descriptor) []string {
	var out []string
	if descriptor.Pack != nil && !strings.HasPrefix(descriptor.Pack.Name, "custom") {
		out = append(out, descriptor.Pack.Name)
	}
	return out
}

// disambiguate resolves a bare reference by trying each known kit as its owner.
func disambiguate(cat *catalog.Catalog, ref string, kits []string) (*model.Resource, error) {
	var matches []*model.Resource
	for _, kit := range kits {
		if res, ok := cat.Get(model.ID(kit + "/" + ref)); ok {
			matches = append(matches, res)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	return nil, errs.New(errs.CodeAmbiguousID,
		"%q could belong to more than one kit; install it explicitly by qualified id", ref)
}

// dirEntries lists the immediate subdirectory names of a workspace-relative directory.
func dirEntries(project, rel string) []string {
	abs, err := security.Contain(project, rel)
	if err != nil {
		return nil
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil
	}
	var out []string
	for _, entry := range entries {
		if entry.IsDir() {
			out = append(out, entry.Name())
		}
	}
	sort.Strings(out)
	return out
}

// markdownStems lists the Markdown filenames without extension in a directory.
func markdownStems(project, rel string) []string {
	abs, err := security.Contain(project, rel)
	if err != nil {
		return nil
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil
	}
	var out []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(name), ".md") {
			continue
		}
		out = append(out, strings.TrimSuffix(name, filepath.Ext(name)))
	}
	sort.Strings(out)
	return out
}
