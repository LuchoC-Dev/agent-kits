package migrate

import (
	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/journal"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
	"github.com/LuchoC-Dev/agent-kits/internal/workspace"
)

// Report summarises what applying a migration did.
type Report struct {
	Operation  string               `json:"operation"`
	Runtime    string               `json:"runtime"`
	Origin     string               `json:"origin"`
	FromSchema int                  `json:"from_schema"`
	ToSchema   int                  `json:"to_schema"`
	Changed    bool                 `json:"changed"`
	Adopted    []model.PlanResource `json:"adopted,omitempty"`
	Preserved  []string             `json:"preserved,omitempty"`
	Backup     string               `json:"backup,omitempty"`
	Retired    []string             `json:"retired,omitempty"`
	Warnings   []model.Diagnostic   `json:"warnings,omitempty"`
}

// Apply writes an approved migration.
//
// The three writes are one operation: the lockfile and the backup must both be on disk
// before workspace.json is retired, and any failure restores the project exactly as it
// was — lockfile, backup and descriptor included (D-031).
func Apply(project string, plan *Plan) (*Report, error) {
	if plan.Blocked() {
		return nil, blockedError(plan)
	}
	report := &Report{
		Operation: plan.Operation, Runtime: plan.Runtime, Origin: plan.Origin,
		FromSchema: plan.FromSchema, ToSchema: plan.ToSchema,
		Adopted: plan.Adopted, Preserved: plan.Preserved, Warnings: plan.Warnings,
	}
	if plan.Empty() {
		return report, nil
	}
	if plan.Lock == nil {
		return nil, errs.New(errs.CodeInternal, "the migration carries no proposed lockfile")
	}

	jrnl, err := journal.New(project)
	if err != nil {
		return nil, err
	}
	defer jrnl.Discard()

	if err := write(jrnl, plan); err != nil {
		if rollbackErr := jrnl.Rollback(); rollbackErr != nil {
			return nil, errs.Wrap(errs.CodeInternal, rollbackErr,
				"the migration failed (%s) and the rollback did not complete", err.Error())
		}
		return nil, err
	}

	report.Changed = true
	report.Backup = plan.Backup
	report.Retired = plan.Retired
	return report, nil
}

// write applies the changes in the order the plan lists them, which is the order that
// keeps the project recoverable at every intermediate point.
func write(jrnl *journal.Journal, plan *Plan) error {
	lockBytes, err := workspace.LockBytes(plan.Lock)
	if err != nil {
		return err
	}
	for _, change := range plan.Changes {
		if !change.Action.Writes() {
			continue
		}
		switch change.Path {
		case workspace.BackupPath:
			if len(plan.LegacyBytes) == 0 {
				return errs.New(errs.CodeInternal, "the migration carries no bytes to back up")
			}
			// The backup is a copy of the original bytes, never a re-serialisation.
			if err := jrnl.Write(change.Path, plan.LegacyBytes); err != nil {
				return err
			}
		case workspace.LegacyPath:
			if err := jrnl.Remove(change.Path); err != nil {
				return err
			}
		default:
			if err := jrnl.Write(change.Path, lockBytes); err != nil {
				return err
			}
		}
	}
	return nil
}

// blockedError turns a blocked migration into the coded failure the caller must handle.
func blockedError(plan *Plan) error {
	primary := plan.Blockers[0]
	location := primary.Path
	if location == "" {
		location = primary.Ref
	}
	err := errs.New(primary.Code, "%s: %s", location, primary.Message).
		With("blockers", plan.Blockers)
	switch primary.Code {
	case errs.CodeIntegrityMismatch:
		return err.Hint("move %s aside and run the migration again", workspace.BackupPath)
	case errs.CodeLocalDivergence:
		return err.Hint("restore the file from its source, or remove it, then migrate again")
	case errs.CodeWorkspaceInvalid:
		return err.Hint("reconcile %s with the lockfile before migrating", workspace.LegacyPath)
	}
	return err
}
