package cli

import (
	"fmt"

	"github.com/LuchoC-Dev/agent-kits/internal/adapter"
	"github.com/LuchoC-Dev/agent-kits/internal/catalog"
	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/install"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
	"github.com/LuchoC-Dev/agent-kits/internal/plan"
	"github.com/LuchoC-Dev/agent-kits/internal/resolve"
	"github.com/LuchoC-Dev/agent-kits/internal/workspace"
)

// projectContext is everything a project-facing command needs after flag parsing.
type projectContext struct {
	opts    *options
	project string
	adapter adapter.Adapter
	env     *environment
	catalog *catalog.Catalog
	lock    *model.Lock
	planner *plan.Planner
}

// openProject prepares a project-facing command.
func (a *App) openProject(opts *options, needCatalog bool) (*projectContext, error) {
	project, err := resolveProject(opts.project)
	if err != nil {
		return nil, err
	}
	runtime, err := adapter.Get(opts.runtime)
	if err != nil {
		return nil, err
	}
	env, err := openEnvironment()
	if err != nil {
		return nil, err
	}
	ctx := &projectContext{opts: opts, project: project, adapter: runtime, env: env}
	if needCatalog {
		ctx.catalog, err = env.requireCatalog()
		if err != nil {
			return nil, err
		}
	} else {
		ctx.catalog = env.catalog
	}
	ctx.lock, err = workspace.LoadLock(project, runtime)
	if err != nil {
		return nil, err
	}
	ctx.planner = plan.New(runtime, project, ctx.lock)
	ctx.planner.Force = opts.force
	return ctx, nil
}

// buildInstallPlan resolves refs and plans their installation.
func (ctx *projectContext) buildInstallPlan(refs []string) (*model.Plan, *resolve.Result, error) {
	resolver := resolve.New(ctx.catalog, ctx.adapter.Name())
	resolution, err := resolver.Resolve(refs)
	if err != nil {
		return nil, nil, err
	}
	built, err := ctx.planner.Install(resolution)
	if err != nil {
		return nil, nil, err
	}
	return built, resolution, nil
}

// cmdPlan previews an installation without writing (RF-06).
func (a *App) cmdPlan(args []string) error {
	set := a.newFlagSet("plan")
	opts := &options{}
	bindProjectFlags(set, opts)
	operands, err := a.parse(set, args, opts)
	if err != nil {
		return err
	}
	if len(operands) == 0 {
		return errs.New(errs.CodeUsage, "usage: agent-kits plan <id>... --project <path>")
	}
	ctx, err := a.openProject(opts, true)
	if err != nil {
		return err
	}
	built, _, err := ctx.buildInstallPlan(operands)
	if err != nil {
		return emptyCatalogHint(ctx.env, err)
	}
	return a.succeed("plan", built, func() { a.renderPlan(built) })
}

// cmdInstall installs resources and their dependencies.
func (a *App) cmdInstall(args []string) error {
	set := a.newFlagSet("install")
	opts := &options{}
	bindProjectFlags(set, opts)
	operands, err := a.parse(set, args, opts)
	if err != nil {
		return err
	}
	if len(operands) == 0 {
		return errs.New(errs.CodeUsage, "usage: agent-kits install <id>... --project <path>")
	}
	ctx, err := a.openProject(opts, true)
	if err != nil {
		return err
	}
	built, _, err := ctx.buildInstallPlan(operands)
	if err != nil {
		return emptyCatalogHint(ctx.env, err)
	}
	return a.applyPlan("install", ctx, built)
}

// cmdUpdate re-plans the resources the project requested, picking up catalog changes.
func (a *App) cmdUpdate(args []string) error {
	set := a.newFlagSet("update")
	opts := &options{}
	bindProjectFlags(set, opts)
	operands, err := a.parse(set, args, opts)
	if err != nil {
		return err
	}
	ctx, err := a.openProject(opts, true)
	if err != nil {
		return err
	}
	refs := operands
	if len(refs) == 0 {
		for _, id := range ctx.lock.RequestedIDs() {
			refs = append(refs, string(id))
		}
	}
	if len(refs) == 0 {
		return errs.New(errs.CodeNotInstalled, "this project has nothing to update").
			Hint("install something first, or run `agent-kits import` to adopt an existing workspace")
	}
	built, _, err := ctx.buildInstallPlan(refs)
	if err != nil {
		return err
	}
	built.Operation = "update"
	return a.applyPlan("update", ctx, built)
}

// cmdRemove uninstalls resources and the dependencies nothing else needs.
func (a *App) cmdRemove(args []string) error {
	set := a.newFlagSet("remove")
	opts := &options{}
	bindProjectFlags(set, opts)
	operands, err := a.parse(set, args, opts)
	if err != nil {
		return err
	}
	if len(operands) == 0 {
		return errs.New(errs.CodeUsage, "usage: agent-kits remove <id>... --project <path>")
	}
	ctx, err := a.openProject(opts, false)
	if err != nil {
		return err
	}
	built, err := ctx.planner.Remove(operands, ctx.catalog)
	if err != nil {
		return err
	}
	return a.applyPlan("remove", ctx, built)
}

// applyPlan renders a plan, asks for approval when needed, and applies it.
func (a *App) applyPlan(command string, ctx *projectContext, built *model.Plan) error {
	if built.Blocked() {
		if !a.wantJSON {
			a.renderPlan(built)
		}
		return blockedFailure(built)
	}
	if built.Empty() {
		return a.succeed(command, map[string]any{
			"operation": built.Operation,
			"runtime":   built.Runtime,
			"changed":   false,
			"plan":      built,
		}, func() {
			fmt.Fprintf(a.Stdout, "%s · nothing to do (already up to date)\n", built.Operation)
			renderDiagnostics(a.Stdout, "warning", built.Warnings)
		})
	}

	if !a.wantJSON {
		a.renderPlan(built)
	}
	if err := a.confirm(ctx.opts, "Apply this plan?"); err != nil {
		return err
	}

	installer := install.New(ctx.adapter, ctx.project, resourceIndex(ctx.catalog))
	report, err := installer.Apply(built)
	if err != nil {
		return err
	}
	return a.succeed(command, map[string]any{
		"operation": report.Operation,
		"runtime":   report.Runtime,
		"changed":   true,
		"report":    report,
	}, func() {
		fmt.Fprintf(a.Stdout, "\n✓ %s · create %d · update %d · adopt %d · remove %d\n",
			report.Operation, report.Created, report.Updated, report.Adopted, report.Removed)
		fmt.Fprintf(a.Stdout, "  lockfile %s\n", ctx.adapter.LockPath())
	})
}

// blockedFailure converts a blocked plan into the coded error the caller must handle.
func blockedFailure(built *model.Plan) error {
	primary := built.Blockers[0]
	location := primary.Path
	if location == "" {
		location = primary.Ref
	}
	err := errs.New(primary.Code, "%s: %s", location, primary.Message).
		With("blockers", built.Blockers)
	if primary.Code == errs.CodeLocalDivergence {
		return err.Hint("inspect the listed files, then re-run with --force to overwrite them")
	}
	return err
}

// cmdList reports what a project has installed.
func (a *App) cmdList(args []string) error {
	set := a.newFlagSet("list")
	opts := &options{}
	bindProjectFlags(set, opts)
	_, err := a.parse(set, args, opts)
	if err != nil {
		return err
	}
	ctx, err := a.openProject(opts, false)
	if err != nil {
		return err
	}
	type row struct {
		ID        model.ID   `json:"id"`
		Type      model.Type `json:"type"`
		Version   string     `json:"version"`
		Source    string     `json:"source"`
		Requested bool       `json:"requested"`
		Files     int        `json:"files"`
		Available string     `json:"available,omitempty"`
	}
	rows := make([]row, 0, len(ctx.lock.Resources))
	for _, record := range ctx.lock.Resources {
		entry := row{
			ID: record.ID, Type: record.Type, Version: record.Version,
			Source: record.Source, Requested: record.Requested, Files: len(record.Files),
		}
		if ctx.catalog != nil {
			if current, ok := ctx.catalog.Get(record.ID); ok && current.Version != record.Version {
				entry.Available = current.Version
			}
		}
		rows = append(rows, entry)
	}
	return a.succeed("list", map[string]any{
		"project": ctx.project,
		"runtime": ctx.lock.Runtime,
		"count":   len(rows),
		"lock":    ctx.adapter.LockPath(),
		"managed": rows,
	}, func() {
		if len(rows) == 0 {
			fmt.Fprintln(a.Stdout, "nothing installed")
			return
		}
		fmt.Fprintf(a.Stdout, "%d resource(s) · runtime %s\n\n", len(rows), ctx.lock.Runtime)
		table := newTable("ID", "TYPE", "VERSION", "SOURCE", "REQUESTED", "UPDATE")
		for _, entry := range rows {
			requested := ""
			if entry.Requested {
				requested = "yes"
			}
			table.add(string(entry.ID), string(entry.Type), entry.Version,
				entry.Source, requested, entry.Available)
		}
		table.render(a.Stdout)
	})
}

// cmdDoctor diagnoses sources and the project (RF-12).
func (a *App) cmdDoctor(args []string) error {
	set := a.newFlagSet("doctor")
	opts := &options{}
	bindProjectFlags(set, opts)
	_, err := a.parse(set, args, opts)
	if err != nil {
		return err
	}
	ctx, err := a.openProject(opts, false)
	if err != nil {
		return err
	}
	report, err := install.Doctor(install.DoctorInput{
		Project:    ctx.project,
		Adapter:    ctx.adapter,
		Store:      ctx.env.store,
		Catalog:    ctx.env.catalog,
		CatalogErr: ctx.env.catalogErr,
	})
	if err != nil {
		return err
	}
	renderReport := func() {
		state := "healthy"
		if !report.Healthy {
			state = fmt.Sprintf("%d problem(s)", len(report.Problems))
		}
		fmt.Fprintf(a.Stdout, "%s · runtime %s · %d installed · %d source(s)\n",
			state, report.Runtime, report.Installed, len(report.Sources))
		if len(report.Sources) > 0 {
			fmt.Fprintln(a.Stdout)
			table := newTable("SOURCE", "ACCESS", "TRUST", "STATE")
			for _, src := range report.Sources {
				status := "unreachable"
				if src.Reachable {
					status = shortCommit(src.Commit)
				}
				table.add(src.Name, src.Access, src.Trust, status)
			}
			table.render(a.Stdout)
		}
		renderDiagnostics(a.Stdout, "problem", report.Problems)
		renderDiagnostics(a.Stdout, "note", report.Notes)
	}
	// A diagnosis is a result, not a failure: the report is emitted once, and an unhealthy
	// project is signalled only through the exit code so a caller can branch on it.
	if a.wantJSON {
		result := envelope{OK: report.Healthy, Command: "doctor", Data: report}
		if !report.Healthy {
			result.Error = &envelopeError{
				Code:    errs.CodeWorkspaceInvalid,
				Message: fmt.Sprintf("%d problem(s) found", len(report.Problems)),
			}
		}
		a.emitJSON(result)
	} else {
		renderReport()
		if !report.Healthy {
			fmt.Fprintf(a.Stderr, "\n%d problem(s) found\n", len(report.Problems))
		}
	}
	if !report.Healthy {
		return quietError{code: errs.ExitConflict}
	}
	return nil
}

// cmdImport adopts a workspace created by the conversational kits-init flow.
func (a *App) cmdImport(args []string) error {
	set := a.newFlagSet("import")
	opts := &options{}
	bindProjectFlags(set, opts)
	_, err := a.parse(set, args, opts)
	if err != nil {
		return err
	}
	ctx, err := a.openProject(opts, true)
	if err != nil {
		return err
	}
	built, err := install.Import(install.ImportInput{
		Project: ctx.project,
		Adapter: ctx.adapter,
		Catalog: ctx.catalog,
		Force:   opts.force,
	})
	if err != nil {
		return err
	}
	if len(built.Changes) == 0 {
		return a.succeed("import", map[string]any{
			"operation": "import",
			"changed":   false,
			"plan":      built,
		}, func() {
			fmt.Fprintln(a.Stdout, "import · nothing to adopt")
			renderDiagnostics(a.Stdout, "warning", built.Warnings)
		})
	}
	if !a.wantJSON {
		a.renderPlan(built)
	}
	if err := a.confirm(opts, "Record these resources in the lockfile?"); err != nil {
		return err
	}
	installer := install.New(ctx.adapter, ctx.project, resourceIndex(ctx.catalog))
	report, err := installer.Apply(built)
	if err != nil {
		return err
	}
	return a.succeed("import", map[string]any{
		"operation": "import",
		"changed":   true,
		"report":    report,
	}, func() {
		fmt.Fprintf(a.Stdout, "\n✓ import · adopted %d resource(s)\n", len(built.Resources))
		fmt.Fprintf(a.Stdout, "  lockfile %s\n", ctx.adapter.LockPath())
		if len(built.Warnings) > 0 {
			fmt.Fprintf(a.Stdout, "  %d item(s) were not adopted; run `agent-kits doctor` for detail\n",
				len(built.Warnings))
		}
	})
}
