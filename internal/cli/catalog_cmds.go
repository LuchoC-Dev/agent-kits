package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/LuchoC-Dev/agent-kits/internal/adapter"
	"github.com/LuchoC-Dev/agent-kits/internal/catalog"
	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/git"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
	"github.com/LuchoC-Dev/agent-kits/internal/source"
)

// cmdVersion reports the build and the contracts a caller can rely on.
func (a *App) cmdVersion(args []string) error {
	set := a.newFlagSet("version")
	opts := &options{}
	set.BoolVar(&opts.json, "json", false, "emit JSON")
	_, err := a.parse(set, args, opts)
	if err != nil {
		return err
	}
	data := map[string]any{
		"version":         Version,
		"manifest_schema": model.ManifestSchemaVersion,
		"lock_schema":     model.LockSchemaVersion,
		// The lockfile a previous build wrote is still readable; workspace.json is only an
		// input to `migrate` and is never written (D-030).
		"lock_schema_read":  []int{model.LockSchemaVersionLegacy, model.LockSchemaVersion},
		"runtimes":          adapter.Names(),
		"types":             model.Types(),
		"error_codes":       errs.Codes(),
		"git_subcommands":   git.AllowedSubcommands(),
		"remote_writes":     false,
	}
	return a.succeed("version", data, func() {
		fmt.Fprintf(a.Stdout, "agent-kits %s\n", Version)
		fmt.Fprintf(a.Stdout, "  schemas    manifest v%d · lock v%d (reads v%d)\n",
			model.ManifestSchemaVersion, model.LockSchemaVersion, model.LockSchemaVersionLegacy)
		fmt.Fprintf(a.Stdout, "  runtimes   %s\n", strings.Join(adapter.Names(), ", "))
		fmt.Fprintf(a.Stdout, "  types      %s\n", joinTypes(model.Types()))
		fmt.Fprintf(a.Stdout, "  git        read-only (%s)\n",
			strings.Join(git.AllowedSubcommands(), ", "))
	})
}

func joinTypes(types []model.Type) string {
	out := make([]string, 0, len(types))
	for _, t := range types {
		out = append(out, string(t))
	}
	return strings.Join(out, ", ")
}

// cmdSource dispatches the source subcommands.
func (a *App) cmdSource(args []string) error {
	if len(args) == 0 {
		return errs.New(errs.CodeUsage, "usage: agent-kits source list|add|remove|sync")
	}
	switch args[0] {
	case "list":
		return a.sourceList(args[1:])
	case "add":
		return a.sourceAdd(args[1:])
	case "remove", "rm":
		return a.sourceRemove(args[1:])
	case "sync":
		return a.sourceSync(args[1:])
	}
	return errs.New(errs.CodeUsage, "unknown source subcommand %q", args[0])
}

func (a *App) sourceList(args []string) error {
	set := a.newFlagSet("source list")
	opts := &options{}
	set.BoolVar(&opts.json, "json", false, "emit JSON")
	_, err := a.parse(set, args, opts)
	if err != nil {
		return err
	}
	store, err := source.Open()
	if err != nil {
		return err
	}
	type row struct {
		Name      string `json:"name"`
		URL       string `json:"url"`
		Ref       string `json:"ref,omitempty"`
		Access    string `json:"access"`
		Trust     string `json:"trust"`
		Local     bool   `json:"local"`
		Synced    bool   `json:"synced"`
		Commit    string `json:"commit,omitempty"`
		Resources int    `json:"resources"`
	}
	loader := catalog.NewLoader()
	rows := []row{}
	for _, src := range store.List() {
		entry := row{
			Name: src.Name, URL: src.URL, Ref: src.Ref,
			Access: string(src.Access), Trust: string(src.Trust), Local: src.IsLocal(),
		}
		if checkout, resolveErr := store.Resolve(src); resolveErr == nil {
			entry.Synced = true
			entry.Commit = checkout.Commit
			if cat, loadErr := loader.LoadCheckout(checkout); loadErr == nil {
				entry.Resources = cat.Len()
			}
		}
		rows = append(rows, entry)
	}
	return a.succeed("source list", map[string]any{
		"config":  store.ConfigPath(),
		"cache":   store.CacheDir(),
		"sources": rows,
	}, func() {
		if len(rows) == 0 {
			fmt.Fprintf(a.Stdout, "no sources configured · %s\n", store.ConfigPath())
			fmt.Fprintln(a.Stdout, "  add one: agent-kits source add <name> <url>")
			return
		}
		fmt.Fprintf(a.Stdout, "%d source(s) · %s\n\n", len(rows), store.ConfigPath())
		table := newTable("NAME", "ACCESS", "TRUST", "STATE", "RESOURCES", "URL")
		for _, entry := range rows {
			state := "not synced"
			if entry.Local {
				state = "local"
			} else if entry.Synced {
				state = shortCommit(entry.Commit)
			}
			table.add(entry.Name, entry.Access, entry.Trust, state,
				fmt.Sprintf("%d", entry.Resources), entry.URL)
		}
		table.render(a.Stdout)
	})
}

func (a *App) sourceAdd(args []string) error {
	set := a.newFlagSet("source add")
	opts := &options{}
	access := set.String("access", string(model.AccessPublic), "public|private")
	trust := set.String("trust", string(model.TrustReview), "trusted|review")
	ref := set.String("ref", "", "branch or tag to track")
	publishes := set.String("publishes", "", "name of the source this one publishes")
	set.BoolVar(&opts.json, "json", false, "emit JSON")
	operands, err := a.parse(set, args, opts)
	if err != nil {
		return err
	}
	if len(operands) != 2 {
		return errs.New(errs.CodeUsage, "usage: agent-kits source add <name> <url>")
	}
	store, err := source.Open()
	if err != nil {
		return err
	}
	src := source.Source{
		Name:      operands[0],
		URL:       operands[1],
		Ref:       *ref,
		Access:    model.Access(*access),
		Trust:     model.Trust(*trust),
		Publishes: *publishes,
	}
	if err := store.Add(src); err != nil {
		return err
	}
	return a.succeed("source add", src, func() {
		fmt.Fprintf(a.Stdout, "added %s (%s, %s)\n", src.Name, src.Access, src.Trust)
		if !src.IsLocal() {
			fmt.Fprintf(a.Stdout, "  next: agent-kits source sync %s\n", src.Name)
		}
	})
}

func (a *App) sourceRemove(args []string) error {
	set := a.newFlagSet("source remove")
	opts := &options{}
	set.BoolVar(&opts.json, "json", false, "emit JSON")
	operands, err := a.parse(set, args, opts)
	if err != nil {
		return err
	}
	if len(operands) != 1 {
		return errs.New(errs.CodeUsage, "usage: agent-kits source remove <name>")
	}
	store, err := source.Open()
	if err != nil {
		return err
	}
	name := operands[0]
	if err := store.Remove(name); err != nil {
		return err
	}
	return a.succeed("source remove", map[string]string{"name": name}, func() {
		fmt.Fprintf(a.Stdout, "removed %s\n", name)
	})
}

func (a *App) sourceSync(args []string) error {
	set := a.newFlagSet("source sync")
	opts := &options{}
	set.BoolVar(&opts.json, "json", false, "emit JSON")
	operands, err := a.parse(set, args, opts)
	if err != nil {
		return err
	}
	store, err := source.Open()
	if err != nil {
		return err
	}
	targets := store.List()
	if len(operands) > 0 {
		targets = nil
		for _, name := range operands {
			src, getErr := store.Get(name)
			if getErr != nil {
				return getErr
			}
			targets = append(targets, src)
		}
	}
	if len(targets) == 0 {
		return errs.New(errs.CodeSourceUnknown, "no sources are configured").
			Hint("add one with `agent-kits source add <name> <url>`")
	}

	type result struct {
		Name      string `json:"name"`
		Commit    string `json:"commit,omitempty"`
		Resources int    `json:"resources"`
		Error     string `json:"error,omitempty"`
	}
	loader := catalog.NewLoader()
	results := make([]result, 0, len(targets))
	var failures int
	for _, src := range targets {
		entry := result{Name: src.Name}
		checkout, syncErr := store.Sync(src)
		if syncErr != nil {
			entry.Error = syncErr.Error()
			failures++
		} else {
			entry.Commit = checkout.Commit
			if cat, loadErr := loader.LoadCheckout(checkout); loadErr != nil {
				entry.Error = loadErr.Error()
				failures++
			} else {
				entry.Resources = cat.Len()
			}
		}
		results = append(results, entry)
	}
	if failures == len(results) {
		return errs.New(errs.CodeSourceUnavailable, "no source could be synced").
			With("results", results)
	}
	return a.succeed("source sync", map[string]any{"sources": results}, func() {
		for _, entry := range results {
			if entry.Error != "" {
				fmt.Fprintf(a.Stdout, "! %-16s %s\n", entry.Name, entry.Error)
				continue
			}
			fmt.Fprintf(a.Stdout, "✓ %-16s %d resources · %s\n",
				entry.Name, entry.Resources, shortCommit(entry.Commit))
		}
	})
}

// cmdSearch lists resources matching a query (RF-04).
func (a *App) cmdSearch(args []string) error {
	set := a.newFlagSet("search")
	opts := &options{}
	typeFilter := set.String("type", "", "filter by type")
	sourceFilter := set.String("source", "", "filter by source")
	limit := set.Int("limit", 0, "maximum results (0 = all)")
	set.BoolVar(&opts.json, "json", false, "emit JSON")
	operands, err := a.parse(set, args, opts)
	if err != nil {
		return err
	}
	if *typeFilter != "" && !model.Type(*typeFilter).Valid() {
		return errs.New(errs.CodeUsage, "unknown --type %q (%s)", *typeFilter, joinTypes(model.Types()))
	}
	env, err := openEnvironment()
	if err != nil {
		return err
	}
	cat, err := env.requireCatalog()
	if err != nil {
		return err
	}
	matches := cat.Search(catalog.Query{
		Text:   strings.Join(operands, " "),
		Type:   model.Type(*typeFilter),
		Source: *sourceFilter,
	})
	if *limit > 0 && len(matches) > *limit {
		matches = matches[:*limit]
	}
	return a.succeed("search", map[string]any{
		"count":   len(matches),
		"results": summarise(matches),
	}, func() {
		if len(matches) == 0 {
			fmt.Fprintln(a.Stdout, "no matches")
			if len(env.store.List()) == 0 {
				fmt.Fprintln(a.Stdout, "  no sources configured: agent-kits source add <name> <url>")
			}
			return
		}
		fmt.Fprintf(a.Stdout, "%d result(s)\n\n", len(matches))
		table := newTable("ID", "TYPE", "VERSION", "SOURCE", "DESCRIPTION")
		for _, res := range matches {
			table.add(res.Name, string(res.Type), res.Version, res.Source,
				truncate(res.Description, 60))
		}
		table.render(a.Stdout)
	})
}

// summary is the JSON shape of a search hit.
type summary struct {
	ID          model.ID   `json:"id"`
	Type        model.Type `json:"type"`
	Name        string     `json:"name,omitempty"`
	Version     string     `json:"version"`
	Source      string     `json:"source"`
	Access      string     `json:"access,omitempty"`
	Description string     `json:"description,omitempty"`
}

func summarise(list []*model.Resource) []summary {
	out := make([]summary, 0, len(list))
	for _, res := range list {
		out = append(out, summary{
			ID: res.ID, Type: res.Type, Name: res.Name, Version: res.Version,
			Source: res.Source, Access: string(res.Access), Description: res.Description,
		})
	}
	return out
}

// cmdInfo inspects one resource and the closure it would pull in.
func (a *App) cmdInfo(args []string) error {
	set := a.newFlagSet("info")
	opts := &options{}
	set.BoolVar(&opts.json, "json", false, "emit JSON")
	operands, err := a.parse(set, args, opts)
	if err != nil {
		return err
	}
	if len(operands) != 1 {
		return errs.New(errs.CodeUsage, "usage: agent-kits info <id>")
	}
	env, err := openEnvironment()
	if err != nil {
		return err
	}
	cat, err := env.requireCatalog()
	if err != nil {
		return err
	}
	res, err := cat.Lookup(operands[0])
	if err != nil {
		return emptyCatalogHint(env, err)
	}

	dependencies := make([]string, 0, len(res.Dependencies))
	for _, dep := range res.Dependencies {
		label := dep.Label()
		if dep.Version != "" {
			label += " " + dep.Version
		}
		dependencies = append(dependencies, label)
	}
	dependents := []string{}
	for _, candidate := range cat.All() {
		for _, dep := range candidate.Dependencies {
			if dep.ID == res.ID {
				dependents = append(dependents, candidate.Name)
				break
			}
		}
	}
	sort.Strings(dependents)

	data := map[string]any{
		"id":           res.ID,
		"type":         res.Type,
		"name":         res.Name,
		"title":        res.Title,
		"qualified":    res.Qualified(),
		"version":      res.Version,
		"description":  res.Description,
		"source":       res.Source,
		"access":       res.Access,
		"trust":        res.Trust,
		"commit":       res.Commit,
		"dependencies": dependencies,
		"dependents":   dependents,
		"files":        res.Files,
		"traits":       res.Traits,
		"labels":       res.Labels,
		"produces":     res.Produces,
		"consumes":     res.Consumes,
	}
	return a.succeed("info", data, func() {
		fmt.Fprintf(a.Stdout, "%s · %s %s · source %s\n",
			res.Name, res.Type, res.Version, res.Source)
		fmt.Fprintf(a.Stdout, "  id          %s\n", res.ID)
		if res.Description != "" {
			fmt.Fprintf(a.Stdout, "\n%s\n", wrap(res.Description, 88))
		}
		fmt.Fprintln(a.Stdout)
		fmt.Fprintf(a.Stdout, "  access      %s · trust %s\n", res.Access, res.Trust)
		if res.Commit != "" {
			fmt.Fprintf(a.Stdout, "  commit      %s\n", shortCommit(res.Commit))
		}
		fmt.Fprintf(a.Stdout, "  files       %d\n", len(res.Files))
		if len(dependencies) > 0 {
			fmt.Fprintf(a.Stdout, "  requires    %s\n", strings.Join(dependencies, ", "))
		}
		if len(dependents) > 0 {
			fmt.Fprintf(a.Stdout, "  required by %s\n", strings.Join(dependents, ", "))
		}
		for _, artifact := range res.Produces {
			fmt.Fprintf(a.Stdout, "  produces    %s → %s\n", artifact.Artifact, artifact.Path)
		}
		for _, artifact := range res.Consumes {
			fmt.Fprintf(a.Stdout, "  consumes    %s\n", artifact.Artifact)
		}
		if len(res.Traits) > 0 {
			keys := make([]string, 0, len(res.Traits))
			for key := range res.Traits {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			fmt.Fprintf(a.Stdout, "  traits      %s\n", strings.Join(keys, ", "))
		}
	})
}
