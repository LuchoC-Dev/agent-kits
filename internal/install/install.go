// Package install applies approved plans to a project.
//
// Application is journalled (see internal/journal): any failure restores the project to
// its previous state, so a partially applied install is never left behind.
package install

import (
	"os"
	"path/filepath"
	"time"

	"github.com/LuchoC-Dev/agent-kits/internal/adapter"
	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/fsutil"
	"github.com/LuchoC-Dev/agent-kits/internal/journal"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
	"github.com/LuchoC-Dev/agent-kits/internal/security"
	"github.com/LuchoC-Dev/agent-kits/internal/workspace"
)

// Installer applies plans for one project and runtime.
type Installer struct {
	Adapter adapter.Adapter
	Project string
	Now     func() time.Time
}

// New returns an installer.
func New(a adapter.Adapter, project string) *Installer {
	return &Installer{Adapter: a, Project: project, Now: time.Now}
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

	jrnl, err := journal.New(i.Project)
	if err != nil {
		return nil, err
	}
	defer jrnl.Discard()

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
				if err := jrnl.Write(change.Path, content); err != nil {
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
				if err := jrnl.Remove(change.Path); err != nil {
					return err
				}
				report.Removed++
			}
		}
		return i.writeMetadata(jrnl, p)
	}

	if err := apply(); err != nil {
		if rollbackErr := jrnl.Rollback(); rollbackErr != nil {
			return nil, errs.Wrap(errs.CodeInternal, rollbackErr,
				"the operation failed (%s) and the rollback did not complete", err.Error())
		}
		return nil, err
	}

	i.pruneEmptyDirs(p)
	return report, nil
}

// writeMetadata rewrites the lockfile, which is the only state file Agent Kits owns.
func (i *Installer) writeMetadata(jrnl *journal.Journal, p *model.Plan) error {
	if p.Lock == nil {
		return errs.New(errs.CodeInternal, "plan carries no proposed lockfile")
	}
	// A project keeps one identity for its whole life, assigned on its first write.
	if p.Lock.Project == nil {
		id, err := model.NewProjectID()
		if err != nil {
			return err
		}
		p.Lock.EnsureProject(id, i.now().UTC().Format(time.RFC3339))
	}
	lockBytes, err := workspace.LockBytes(p.Lock)
	if err != nil {
		return err
	}
	return jrnl.Write(i.Adapter.LockPath(), lockBytes)
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
