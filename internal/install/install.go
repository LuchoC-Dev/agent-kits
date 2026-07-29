// Package install applies approved plans to a project.
//
// Application is journalled: every file the operation overwrites or deletes is copied
// aside first, and any failure restores the project to its previous state. A partially
// applied install is never left behind (02-architecture-direction.md §7).
package install

import (
	"os"
	"path/filepath"
	"time"

	"github.com/LuchoC-Dev/agent-kits/internal/adapter"
	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/fsutil"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
	"github.com/LuchoC-Dev/agent-kits/internal/security"
	"github.com/LuchoC-Dev/agent-kits/internal/workspace"
)

// Installer applies plans for one project and runtime.
type Installer struct {
	Adapter adapter.Adapter
	Project string
	// Resources supplies manifest detail for workspace.json enrichment. It may be nil.
	Resources map[model.ID]*model.Resource
	Now       func() time.Time
}

// New returns an installer.
func New(a adapter.Adapter, project string, resources map[model.ID]*model.Resource) *Installer {
	return &Installer{Adapter: a, Project: project, Resources: resources, Now: time.Now}
}

func (i *Installer) now() time.Time {
	if i.Now == nil {
		return time.Now()
	}
	return i.Now()
}

// Report summarises what applying a plan did.
type Report struct {
	Operation string               `json:"operation"`
	Runtime   string               `json:"runtime"`
	Created   int                  `json:"created"`
	Updated   int                  `json:"updated"`
	Adopted   int                  `json:"adopted"`
	Removed   int                  `json:"removed"`
	Unchanged int                  `json:"unchanged"`
	Resources []model.PlanResource `json:"resources"`
	Warnings  []model.Diagnostic   `json:"warnings,omitempty"`
}

// Apply writes an approved plan.
func (i *Installer) Apply(p *model.Plan) (*Report, error) {
	if p.Blocked() {
		return nil, blockedError(p)
	}
	report := &Report{
		Operation: p.Operation,
		Runtime:   p.Runtime,
		Resources: p.Resources,
		Warnings:  p.Warnings,
	}
	for _, change := range p.Changes {
		if change.Action == model.ActionUnchanged {
			report.Unchanged++
		}
	}
	if p.Empty() {
		return report, nil
	}

	jrnl, err := newJournal(i.Project)
	if err != nil {
		return nil, err
	}
	defer jrnl.discard()

	apply := func() error {
		for _, change := range p.Changes {
			switch change.Action {
			case model.ActionCreate, model.ActionUpdate:
				content, readErr := os.ReadFile(change.SourcePath)
				if readErr != nil {
					return errs.Wrap(errs.CodeSourceUnavailable, readErr,
						"cannot read %s while installing %s", change.SourcePath, change.ResourceID)
				}
				if sum := fsutil.ChecksumBytes(content); sum != change.Checksum {
					return errs.New(errs.CodeIntegrityMismatch,
						"%s changed in its source between plan and apply", change.Path).
						With("path", change.Path).
						Hint("re-run the plan")
				}
				if err := jrnl.write(change.Path, content); err != nil {
					return err
				}
				if change.Action == model.ActionCreate {
					report.Created++
				} else {
					report.Updated++
				}

			case model.ActionAdopt:
				// The file on disk already matches; only the lockfile gains a record.
				report.Adopted++

			case model.ActionRemove:
				if err := jrnl.remove(change.Path); err != nil {
					return err
				}
				report.Removed++
			}
		}
		return i.writeMetadata(jrnl, p)
	}

	if err := apply(); err != nil {
		if rollbackErr := jrnl.rollback(); rollbackErr != nil {
			return nil, errs.Wrap(errs.CodeInternal, rollbackErr,
				"the operation failed (%s) and the rollback did not complete", err.Error())
		}
		return nil, err
	}

	i.pruneEmptyDirs(p)
	return report, nil
}

// writeMetadata rewrites the lockfile and workspace.json.
func (i *Installer) writeMetadata(jrnl *journal, p *model.Plan) error {
	if p.Lock == nil {
		return errs.New(errs.CodeInternal, "plan carries no proposed lockfile")
	}
	lockBytes, err := workspace.LockBytes(p.Lock)
	if err != nil {
		return err
	}
	if err := jrnl.write(i.Adapter.LockPath(), lockBytes); err != nil {
		return err
	}

	existing, _, err := workspace.LoadDescriptor(i.Project, i.Adapter)
	if err != nil {
		return err
	}
	descriptor, err := workspace.Sync(existing, p.Lock, i.Adapter.Name(), i.Resources, i.now())
	if err != nil {
		return err
	}
	descriptorBytes, err := workspace.DescriptorBytes(descriptor)
	if err != nil {
		return err
	}
	return jrnl.write(i.Adapter.WorkspacePath(), descriptorBytes)
}

// pruneEmptyDirs removes directories left empty by removals, so uninstalling a resource
// leaves no scaffolding behind.
func (i *Installer) pruneEmptyDirs(p *model.Plan) {
	root, err := security.Contain(i.Project, adapter.WorkspaceDir)
	if err != nil {
		return
	}
	for _, change := range p.Changes {
		if change.Action != model.ActionRemove {
			continue
		}
		abs, err := security.Contain(i.Project, change.Path)
		if err != nil {
			continue
		}
		_ = fsutil.RemoveEmptyDirs(filepath.Dir(abs), root)
	}
}

// blockedError turns a plan's blockers into a single coded failure.
func blockedError(p *model.Plan) error {
	primary := p.Blockers[0]
	err := errs.New(primary.Code, "%s: %s", primary.Path, primary.Message).
		With("blockers", p.Blockers)
	switch primary.Code {
	case errs.CodeLocalDivergence:
		return err.Hint("inspect the file, then re-run with --force to overwrite it")
	case errs.CodeDestinationConflict:
		return err.Hint("install the conflicting resources into separate projects")
	case errs.CodeUnsafeContent:
		return err.Hint("the source must remove the credential before it can be installed")
	}
	return err
}
