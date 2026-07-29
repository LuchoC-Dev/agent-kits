package install

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

// ImportInput describes an adoption of an existing workspace.
type ImportInput struct {
	Project string
	Adapter adapter.Adapter
	Catalog *catalog.Catalog
	// Force adopts resources whose files diverge from the catalog, recording what is
	// actually on disk instead of refusing.
	Force bool
	Now   func() time.Time
}

// Import builds a plan that adopts a workspace created by the conversational kits-init
// flow, so the CLI can manage it from then on (D-022).
//
// Adoption is deliberately conservative: a resource is adopted only when every one of its
// files matches the catalog byte for byte. Recording a lockfile entry for content that
// does not match would make a later update silently overwrite the difference.
func Import(in ImportInput) (*model.Plan, error) {
	descriptor, present, err := workspace.LoadDescriptor(in.Project, in.Adapter)
	if err != nil {
		return nil, err
	}
	if !present {
		return nil, errs.New(errs.CodeWorkspaceInvalid,
			"no %s to import", in.Adapter.WorkspacePath()).
			Hint("run `agent-kits install` to create a workspace instead")
	}
	if in.Catalog == nil {
		return nil, errs.New(errs.CodeSourceUnavailable,
			"importing needs a readable catalog to identify installed resources")
	}

	now := in.Now
	if now == nil {
		now = time.Now
	}
	out := &model.Plan{
		Operation: "import",
		Project:   fsutil.ToSlash(in.Project),
		Runtime:   in.Adapter.Name(),
	}
	existing, err := workspace.LoadLock(in.Project, in.Adapter)
	if err != nil {
		return nil, err
	}
	proposed := existing.Clone()
	proposed.Runtime = in.Adapter.Name()
	proposed.GeneratedAt = now().UTC().Format(time.RFC3339)

	requested := requestedKits(descriptor)
	for _, candidate := range candidates(in.Project, descriptor) {
		res, err := in.Catalog.Lookup(candidate.ref)
		if err != nil {
			if errs.Is(err, errs.CodeAmbiguousID) {
				// A bare workflow name can belong to more than one kit; the workspace's
				// own composition record is the tie-breaker.
				res, err = disambiguate(in.Catalog, candidate.ref, requested)
			}
			if err != nil {
				out.Warnings = append(out.Warnings, model.Diagnostic{
					Code: errs.CodeOf(err), Ref: candidate.ref, Message: err.Error(),
				})
				continue
			}
		}
		if _, already := proposed.Find(res.ID); already {
			continue
		}
		record, changes, diagnostics := adopt(in, res, candidate.requested)
		out.Warnings = append(out.Warnings, diagnostics...)
		if record == nil {
			continue
		}
		proposed.Upsert(*record)
		out.Changes = append(out.Changes, changes...)
		out.Resources = append(out.Resources, model.PlanResource{
			ID: res.ID, Type: res.Type, Version: res.Version, Source: res.Source,
			Requested: candidate.requested, State: "adopt",
		})
	}

	out.Lock = proposed
	if len(out.Changes) > 0 {
		for _, path := range []string{in.Adapter.LockPath(), in.Adapter.WorkspacePath()} {
			action := model.ActionCreate
			if abs, err := security.Contain(in.Project, path); err == nil && fsutil.Exists(abs) {
				action = model.ActionUpdate
			}
			out.Metadata = append(out.Metadata, model.FileChange{Path: path, Action: action})
		}
	}
	out.Sort()
	return out, nil
}

// adopt verifies a resource's files against the project and builds its lock record.
func adopt(
	in ImportInput, res *model.Resource, requested bool,
) (*model.LockResource, []model.FileChange, []model.Diagnostic) {
	var (
		changes     []model.FileChange
		diagnostics []model.Diagnostic
	)
	record := model.LockResource{
		ID: res.ID, Type: res.Type, Source: res.Source,
		Version: res.Version, Commit: res.Commit, Requested: requested,
	}
	perFile := map[string]string{}

	for _, rel := range res.Files {
		dest, err := in.Adapter.Destination(res, rel)
		if err != nil {
			diagnostics = append(diagnostics, model.Diagnostic{
				Code: errs.CodeOf(err), Ref: string(res.ID), Message: err.Error(),
			})
			return nil, nil, diagnostics
		}
		abs, err := security.Contain(in.Project, dest)
		if err != nil {
			diagnostics = append(diagnostics, model.Diagnostic{
				Code: errs.CodeUnsafePath, Ref: string(res.ID), Path: dest, Message: err.Error(),
			})
			return nil, nil, diagnostics
		}
		if !fsutil.IsRegular(abs) {
			diagnostics = append(diagnostics, model.Diagnostic{
				Code: errs.CodeNotInstalled, Ref: string(res.ID), Path: dest,
				Message: "not adopted: the workspace does not contain this file",
			})
			return nil, nil, diagnostics
		}
		diskSum, _, err := fsutil.ChecksumFile(abs)
		if err != nil {
			diagnostics = append(diagnostics, model.Diagnostic{
				Code: errs.CodeInternal, Ref: string(res.ID), Path: dest, Message: err.Error(),
			})
			return nil, nil, diagnostics
		}
		catalogSum, err := checksumSource(res, rel)
		if err != nil {
			diagnostics = append(diagnostics, model.Diagnostic{
				Code: errs.CodeOf(err), Ref: string(res.ID), Path: dest, Message: err.Error(),
			})
			return nil, nil, diagnostics
		}
		if diskSum != catalogSum && !in.Force {
			diagnostics = append(diagnostics, model.Diagnostic{
				Code: errs.CodeLocalDivergence, Ref: string(res.ID), Path: dest,
				Message: "not adopted: the installed file differs from the catalog",
			})
			return nil, nil, diagnostics
		}
		if diskSum != catalogSum {
			diagnostics = append(diagnostics, model.Diagnostic{
				Code: errs.CodeLocalDivergence, Ref: string(res.ID), Path: dest,
				Message: "adopted with --force: recording the file as it is on disk",
			})
		}
		record.Files = append(record.Files, model.LockFile{Path: dest, Checksum: diskSum})
		perFile[rel] = diskSum
		changes = append(changes, model.FileChange{
			Path: dest, Action: model.ActionAdopt, ResourceID: res.ID, Checksum: diskSum,
		})
	}
	if len(record.Files) == 0 {
		return nil, nil, diagnostics
	}
	record.Checksum = fsutil.ChecksumTree(perFile)
	sort.SliceStable(record.Files, func(i, j int) bool {
		return record.Files[i].Path < record.Files[j].Path
	})
	return &record, changes, diagnostics
}

func checksumSource(res *model.Resource, rel string) (string, error) {
	if err := security.CheckRelPath(rel); err != nil {
		return "", err
	}
	content, err := os.ReadFile(filepath.Join(res.Root, fsutil.FromSlash(rel)))
	if err != nil {
		return "", errs.Wrap(errs.CodeSourceUnavailable, err, "cannot read %s of %s", rel, res.ID)
	}
	return fsutil.ChecksumBytes(content), nil
}

// importCandidate is a reference discovered in an existing workspace.
type importCandidate struct {
	ref       string
	requested bool
}

// candidates lists what an existing workspace appears to contain: the resources
// workspace.json records, plus anything found in the managed directories.
func candidates(project string, descriptor *workspace.Descriptor) []importCandidate {
	seen := map[string]bool{}
	var out []importCandidate
	add := func(ref string, requested bool) {
		ref = strings.TrimSpace(ref)
		if ref == "" || seen[ref] {
			return
		}
		seen[ref] = true
		out = append(out, importCandidate{ref: ref, requested: requested})
	}

	if descriptor.Pack != nil && descriptor.Pack.Name != "" &&
		!strings.HasPrefix(descriptor.Pack.Name, "custom") {
		add(descriptor.Pack.Name, true)
	}
	for _, dir := range dirEntries(project, adapter.WorkspaceDir+"/packs") {
		add(dir, true)
	}
	for _, entry := range descriptor.Skills {
		add(entry.ID, false)
	}
	for _, entry := range descriptor.Agents {
		add(entry.ID, false)
	}
	for _, name := range markdownStems(project, adapter.WorkspaceDir+"/agents") {
		add(name, false)
	}
	for _, name := range markdownStems(project, adapter.WorkspaceDir+"/workflows") {
		add(name, false)
	}
	for _, dir := range dirEntries(project, adapter.WorkspaceDir+"/skills") {
		add(dir, false)
	}
	return out
}

// requestedKits lists the kit names the workspace claims, used to disambiguate names that
// several kits own.
func requestedKits(descriptor *workspace.Descriptor) []string {
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
		if entry.IsDir() || !strings.EqualFold(strings.TrimPrefix(pathExt(name), "."), "md") {
			continue
		}
		out = append(out, strings.TrimSuffix(name, pathExt(name)))
	}
	sort.Strings(out)
	return out
}

func pathExt(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[i:]
	}
	return ""
}
