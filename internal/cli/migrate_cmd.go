package cli

import (
	"fmt"
	"strings"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/migrate"
	"github.com/LuchoC-Dev/agent-kits/internal/workspace"
)

// cmdMigrate moves a project onto the current lockfile schema and retires workspace.json
// (D-031).
//
// It is a temporary command: it exists for the length of the transition window and is
// removed by a separate, approved change.
func (a *App) cmdMigrate(args []string) error {
	return a.migrateWith("migrate", args, false)
}

// cmdImport is the deprecated spelling of migrate. It shares the implementation rather
// than keeping a second adoption path alive, so the two can never disagree.
func (a *App) cmdImport(args []string) error {
	return a.migrateWith("import", args, true)
}

func (a *App) migrateWith(command string, args []string, deprecated bool) error {
	set := a.newFlagSet(command)
	opts := &options{}
	bindProjectFlags(set, opts)
	if _, err := a.parse(set, args, opts); err != nil {
		return err
	}
	if opts.force {
		// Migrating state is not a content conflict: discarding data to continue is exactly
		// what this operation must never do.
		return errs.New(errs.CodeUsage, "--force is not accepted by %s", command).
			Hint("resolve the reported conflict instead; a migration never discards data")
	}
	ctx, err := a.openProject(opts, false)
	if err != nil {
		return err
	}
	if deprecated && !a.wantJSON {
		fmt.Fprintf(a.Stderr,
			"warning: `agent-kits import` is deprecated; use `agent-kits migrate` instead\n")
	}

	plan, err := migrate.Gather(migrate.Input{
		Project: ctx.project,
		Adapter: ctx.adapter,
		Catalog: ctx.catalog,
	})
	if err != nil {
		return err
	}
	if plan.Blocked() {
		if !a.wantJSON {
			a.renderMigration(plan)
		}
		return migrationFailure(plan)
	}
	if plan.Empty() {
		return a.succeed(command, migrationData(command, plan, nil, deprecated), func() {
			fmt.Fprintf(a.Stdout, "%s · nothing to migrate (%s)\n", command, settled(plan))
			renderDiagnostics(a.Stdout, "warning", plan.Warnings)
		})
	}

	if !a.wantJSON {
		a.renderMigration(plan)
	}
	if err := a.confirm(opts, "Apply this migration?"); err != nil {
		return err
	}
	report, err := migrate.Apply(ctx.project, plan)
	if err != nil {
		return err
	}
	return a.succeed(command, migrationData(command, plan, report, deprecated), func() {
		fmt.Fprintf(a.Stdout, "\n✓ %s · lockfile schema %s → %d · %d resource(s) adopted\n",
			command, fromSchema(plan), plan.ToSchema, len(report.Adopted))
		if report.Backup != "" {
			fmt.Fprintf(a.Stdout, "  backup %s (yours to keep or delete)\n", report.Backup)
		}
		for _, path := range report.Retired {
			fmt.Fprintf(a.Stdout, "  retired %s\n", path)
		}
	})
}

// fromSchema labels the starting point. A project with no lockfile has no schema to name,
// which is a different thing from having schema zero.
func fromSchema(plan *migrate.Plan) string {
	if plan.FromSchema == 0 {
		return "none"
	}
	return fmt.Sprint(plan.FromSchema)
}

// settled explains why there is nothing to do, which is the difference between a project
// that is already migrated and one that has no state at all.
func settled(plan *migrate.Plan) string {
	if plan.Origin == migrate.OriginNone {
		return "this project has no Agent Kits state"
	}
	return "already on lockfile schema " + fmt.Sprint(plan.ToSchema)
}

// migrationData is the JSON payload. It is a single envelope for both spellings of the
// command; the deprecated one only adds a field.
func migrationData(
	command string, plan *migrate.Plan, report *migrate.Report, deprecated bool,
) map[string]any {
	data := map[string]any{
		"operation":   "migrate",
		"origin":      plan.Origin,
		"runtime":     plan.Runtime,
		"from_schema": plan.FromSchema,
		"to_schema":   plan.ToSchema,
		"changed":     report != nil && report.Changed,
		"plan":        plan,
	}
	if report != nil {
		data["report"] = report
	}
	if deprecated {
		data["deprecated"] = map[string]any{
			"command":     command,
			"replacement": "migrate",
			"message":     "`import` is deprecated; use `agent-kits migrate`",
		}
	}
	return data
}

// renderMigration prints what a migration would do.
func (a *App) renderMigration(plan *migrate.Plan) {
	fmt.Fprintf(a.Stdout, "migrate · runtime %s · lockfile schema %s → %d\n",
		plan.Runtime, fromSchema(plan), plan.ToSchema)
	fmt.Fprintf(a.Stdout, "  source %s\n", plan.Origin)

	var changes []string
	for _, change := range plan.Changes {
		if change.Action.Writes() {
			changes = append(changes, fmt.Sprintf("%s %s", change.Action, change.Path))
		}
	}
	if len(changes) == 0 {
		changes = []string{"no file changes"}
	}
	fmt.Fprintf(a.Stdout, "  %s\n", strings.Join(changes, " · "))

	if len(plan.Adopted) > 0 {
		ids := make([]string, 0, len(plan.Adopted))
		for _, res := range plan.Adopted {
			ids = append(ids, string(res.ID))
		}
		fmt.Fprintf(a.Stdout, "  adopting %s\n", strings.Join(ids, ", "))
	}
	if len(plan.Preserved) > 0 {
		fmt.Fprintf(a.Stdout, "  preserving %s\n", strings.Join(plan.Preserved, ", "))
	}
	if plan.Backup != "" {
		fmt.Fprintf(a.Stdout, "  backup %s\n", plan.Backup)
	}

	renderDiagnostics(a.Stdout, "blocked", plan.Blockers)
	renderDiagnostics(a.Stdout, "warning", plan.Warnings)
}

// migrationFailure converts a blocked migration into the coded error a caller handles.
func migrationFailure(plan *migrate.Plan) error {
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
		return err.Hint("restore the listed files from their source, or remove them, then migrate again")
	}
	return err.Hint("reconcile %s with the lockfile before migrating", workspace.LegacyPath)
}
