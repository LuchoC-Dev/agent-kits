package install

import (
	"os"
	"sort"

	"github.com/LuchoC-Dev/agent-kits/internal/adapter"
	"github.com/LuchoC-Dev/agent-kits/internal/catalog"
	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/fsutil"
	"github.com/LuchoC-Dev/agent-kits/internal/git"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
	"github.com/LuchoC-Dev/agent-kits/internal/security"
	"github.com/LuchoC-Dev/agent-kits/internal/source"
	"github.com/LuchoC-Dev/agent-kits/internal/workspace"
)

// SourceStatus is the reachability of one configured source.
type SourceStatus struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Access    string `json:"access"`
	Trust     string `json:"trust"`
	Reachable bool   `json:"reachable"`
	Commit    string `json:"commit,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

// DoctorReport is the outcome of a diagnosis (RF-12).
type DoctorReport struct {
	Project   string             `json:"project"`
	Runtime   string             `json:"runtime"`
	Sources   []SourceStatus     `json:"sources"`
	Installed int                `json:"installed"`
	Problems  []model.Diagnostic `json:"problems,omitempty"`
	Notes     []model.Diagnostic `json:"notes,omitempty"`
	Healthy   bool               `json:"healthy"`
}

// DoctorInput carries everything a diagnosis inspects.
type DoctorInput struct {
	Project string
	Adapter adapter.Adapter
	Store   *source.Store
	// Catalog is nil when the catalog could not be loaded, in which case CatalogErr says why.
	Catalog    *catalog.Catalog
	CatalogErr error
}

// Doctor inspects a project and its sources without changing anything.
func Doctor(in DoctorInput) (*DoctorReport, error) {
	report := &DoctorReport{
		Project: fsutil.ToSlash(in.Project),
		Runtime: in.Adapter.Name(),
	}
	problem := func(code errs.Code, ref, path, message string) {
		report.Problems = append(report.Problems,
			model.Diagnostic{Code: code, Ref: ref, Path: path, Message: message})
	}
	note := func(code errs.Code, ref, path, message string) {
		report.Notes = append(report.Notes,
			model.Diagnostic{Code: code, Ref: ref, Path: path, Message: message})
	}

	if !git.Available() {
		note(errs.CodeSourceUnavailable, "", "",
			"git is not on PATH; only local sources can be used")
	}

	for _, src := range in.Store.List() {
		status := SourceStatus{
			Name: src.Name, URL: src.URL,
			Access: string(src.Access), Trust: string(src.Trust),
		}
		checkout, err := in.Store.Resolve(src)
		if err != nil {
			status.Detail = err.Error()
			problem(errs.CodeOf(err), src.Name, "", err.Error())
		} else {
			status.Reachable = true
			status.Commit = checkout.Commit
		}
		report.Sources = append(report.Sources, status)
	}
	if len(report.Sources) == 0 {
		note(errs.CodeSourceUnknown, "", "", "no sources are configured")
	}

	if in.CatalogErr != nil {
		problem(errs.CodeOf(in.CatalogErr), "", "", in.CatalogErr.Error())
	}
	if in.Catalog != nil {
		for _, diagnostic := range in.Catalog.Diagnostics() {
			note(diagnostic.Code, diagnostic.Ref, diagnostic.Path, diagnostic.Message)
		}
	}

	lock, err := workspace.LoadLock(in.Project, in.Adapter)
	if err != nil {
		problem(errs.CodeOf(err), "", in.Adapter.LockPath(), err.Error())
		report.Healthy = false
		return report, nil
	}
	report.Installed = len(lock.Resources)

	owned := map[string]bool{}
	for _, record := range lock.Resources {
		for _, file := range record.Files {
			owned[file.Path] = true
			abs, containErr := security.Contain(in.Project, file.Path)
			if containErr != nil {
				problem(errs.CodeUnsafePath, string(record.ID), file.Path, containErr.Error())
				continue
			}
			if !fsutil.Exists(abs) {
				problem(errs.CodeIntegrityMismatch, string(record.ID), file.Path,
					"file recorded in the lockfile is missing")
				continue
			}
			sum, _, sumErr := fsutil.ChecksumFile(abs)
			if sumErr != nil {
				problem(errs.CodeInternal, string(record.ID), file.Path, sumErr.Error())
				continue
			}
			if sum != file.Checksum {
				problem(errs.CodeLocalDivergence, string(record.ID), file.Path,
					"file was modified after it was installed")
			}
		}
		if in.Catalog == nil {
			continue
		}
		current, found := in.Catalog.Get(record.ID)
		if !found {
			problem(errs.CodeNotFound, string(record.ID), "",
				"installed resource is no longer offered by any configured source")
			continue
		}
		if current.Version != record.Version {
			note(errs.CodeNotFound, string(record.ID), "",
				"an update is available: "+record.Version+" -> "+current.Version)
		}
	}

	// Managed directories may contain files no lockfile claims — typically a workspace
	// created by the retired conversational flow, which `migrate` can adopt.
	for _, orphan := range findOrphans(in.Project, owned) {
		note(errs.CodeWorkspaceInvalid, "", orphan,
			"file is not recorded in the lockfile; `agent-kits migrate` can adopt it")
	}

	// A project that still carries workspace.json has two files claiming to describe it.
	// That is reported with the existing vocabulary: the migration is the remedy, and no
	// new public error code is introduced for it (§7).
	if workspace.Pending(in.Project) {
		problem(errs.CodeWorkspaceInvalid, "", workspace.LegacyPath,
			"this project has not been migrated yet; run `agent-kits migrate --project <path>`")
	}

	sortDiagnostics(report.Problems)
	sortDiagnostics(report.Notes)
	report.Healthy = len(report.Problems) == 0
	return report, nil
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

// managedDirs are the workspace subdirectories Agent Kits installs into.
var managedDirs = []string{"skills", "agents", "workflows", "packs"}

// findOrphans lists files inside the managed directories that no lockfile entry claims.
func findOrphans(project string, owned map[string]bool) []string {
	var out []string
	for _, name := range managedDirs {
		rel := adapter.WorkspaceDir + "/" + name
		abs, err := security.Contain(project, rel)
		if err != nil {
			continue
		}
		info, statErr := os.Stat(abs)
		if statErr != nil || !info.IsDir() {
			continue
		}
		files, _, walkErr := fsutil.WalkFiles(abs)
		if walkErr != nil {
			continue
		}
		for _, file := range files {
			candidate := rel + "/" + file
			if !owned[candidate] {
				out = append(out, candidate)
			}
		}
	}
	sort.Strings(out)
	return out
}
