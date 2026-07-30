// Package cli implements the command surface.
//
// Every command is non-interactive when given sufficient flags, returns a stable JSON
// envelope under --json, and exits with the documented code for its failure (D-009).
package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/LuchoC-Dev/agent-kits/internal/adapter"
	"github.com/LuchoC-Dev/agent-kits/internal/catalog"
	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/source"
)

// Version is the CLI version, reported by `agent-kits version`.
const Version = "0.1.0"

// App holds the process-level dependencies, so tests can drive the CLI in-memory.
type App struct {
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
	// Interactive forces confirmation prompts on or off. When nil, the terminal decides.
	Interactive *bool

	// wantJSON is set while parsing flags so a failure is reported in the same format the
	// caller asked for, even when the failure is the flag parsing itself.
	wantJSON bool
}

// New returns an App bound to the process streams.
func New() *App {
	return &App{Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin}
}

// Run dispatches a command line and returns the process exit code.
func (a *App) Run(args []string) int {
	if len(args) == 0 {
		a.usage()
		return errs.ExitUsage
	}
	name := args[0]
	rest := args[1:]

	switch name {
	case "-h", "--help", "help":
		a.usage()
		return errs.ExitOK
	case "-v", "--version":
		name = "version"
	}

	handler, ok := commands[name]
	if !ok {
		fmt.Fprintf(a.Stderr, "unknown command %q\n\n", name)
		a.usage()
		return errs.ExitUsage
	}

	err := handler(a, rest)
	if err == nil {
		return errs.ExitOK
	}
	var quiet quietError
	if errors.As(err, &quiet) {
		// The command already reported its outcome in full; only the exit code is left.
		return quiet.code
	}
	return a.fail(name, err)
}

// quietError carries an exit code for a command that has already emitted its complete
// output, so a diagnosis is never reported twice or as two JSON documents.
type quietError struct{ code int }

func (quietError) Error() string { return "" }

// handler runs one command.
type handler func(*App, []string) error

var commands map[string]handler

func init() {
	commands = map[string]handler{
		"source":  (*App).cmdSource,
		"search":  (*App).cmdSearch,
		"info":    (*App).cmdInfo,
		"plan":    (*App).cmdPlan,
		"install": (*App).cmdInstall,
		"update":  (*App).cmdUpdate,
		"remove":  (*App).cmdRemove,
		"list":    (*App).cmdList,
		"doctor":  (*App).cmdDoctor,
		"migrate": (*App).cmdMigrate,
		"import":  (*App).cmdImport,
		"version": (*App).cmdVersion,
	}
}

func (a *App) usage() {
	fmt.Fprint(a.Stdout, `agent-kits — discover, plan and install agent capabilities

usage: agent-kits <command> [flags]

catalog
  source list|add|remove|sync   manage the configured sources
  search [query]                find resources
  info <id>                     inspect one resource

project
  plan <id>...                  preview an installation, writing nothing
  install <id>...               install resources and their dependencies
  update [<id>...]              re-install requested resources at their current version
  remove <id>...                uninstall resources and orphaned dependencies
  list                          list what this project has installed
  doctor                        diagnose sources and the project
  migrate                       move an inherited workspace onto the lockfile

  version                       print version, runtimes and error codes

common flags
  --project <path>   destination project (default ".")
  --runtime <name>   auto|agents|claude-code|opencode (default "auto")
  --json             emit a stable JSON envelope
  --yes              apply without confirmation
  --force            overwrite locally modified files

Agent Kits never writes to a source and never publishes.
`)
}

// envelope is the stable JSON contract of every command.
type envelope struct {
	OK      bool           `json:"ok"`
	Command string         `json:"command"`
	Data    any            `json:"data,omitempty"`
	Error   *envelopeError `json:"error,omitempty"`
}

type envelopeError struct {
	Code    errs.Code      `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// jsonRequested reports whether --json appeared anywhere in the arguments. It is checked
// before flag parsing so that a usage error can still answer in JSON.
func jsonRequested(args []string) bool {
	for _, arg := range args {
		if arg == "--json" || arg == "-json" {
			return true
		}
	}
	return false
}

// fail reports an error in the requested format and maps it to an exit code.
func (a *App) fail(command string, err error) int {
	if a.wantJSON {
		payload := &envelopeError{Code: errs.CodeOf(err), Message: err.Error()}
		var coded *errs.Error
		if errors.As(err, &coded) {
			payload.Message = coded.Message
			payload.Details = coded.Details
		}
		a.emitJSON(envelope{OK: false, Command: command, Error: payload})
		return errs.ExitCode(err)
	}

	var coded *errs.Error
	if errors.As(err, &coded) {
		fmt.Fprintf(a.Stderr, "error [%s]: %s\n", coded.Code, coded.Message)
		if hint, ok := coded.Details["hint"].(string); ok {
			fmt.Fprintf(a.Stderr, "  hint: %s\n", hint)
		}
	} else {
		fmt.Fprintf(a.Stderr, "error: %s\n", err.Error())
	}
	return errs.ExitCode(err)
}

func (a *App) emitJSON(value any) {
	encoder := json.NewEncoder(a.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintf(a.Stderr, "error: cannot encode output: %v\n", err)
	}
}

// succeed writes a successful result in the requested format.
func (a *App) succeed(command string, data any, render func()) error {
	if a.wantJSON {
		a.emitJSON(envelope{OK: true, Command: command, Data: data})
		return nil
	}
	render()
	return nil
}

// options are the flags shared by the project-facing commands.
type options struct {
	project string
	runtime string
	json    bool
	yes     bool
	force   bool
}

func (a *App) newFlagSet(name string) *flag.FlagSet {
	set := flag.NewFlagSet("agent-kits "+name, flag.ContinueOnError)
	set.SetOutput(a.Stderr)
	return set
}

// bindProjectFlags registers the flags used by project-facing commands.
func bindProjectFlags(set *flag.FlagSet, opts *options) {
	set.StringVar(&opts.project, "project", ".", "destination project")
	set.StringVar(&opts.runtime, "runtime", adapter.Auto, "target runtime")
	set.BoolVar(&opts.json, "json", false, "emit JSON")
	set.BoolVar(&opts.yes, "yes", false, "apply without confirmation")
	set.BoolVar(&opts.force, "force", false, "overwrite locally modified files")
}

// parse runs a flag set and returns the positional operands.
//
// Flags and operands may be interleaved — `install frontend-design --project .` is as
// valid as `install --project . frontend-design` — because an agent composing a command
// should not have to know that Go's flag package stops at the first operand.
func (a *App) parse(set *flag.FlagSet, args []string, opts *options) ([]string, error) {
	a.wantJSON = jsonRequested(args)
	flags, operands, err := partition(set, args)
	if err != nil {
		return nil, err
	}
	if err := set.Parse(flags); err != nil {
		return nil, errs.New(errs.CodeUsage, "%s", err.Error())
	}
	if opts != nil {
		a.wantJSON = opts.json
	}
	return operands, nil
}

// partition splits args into flag tokens and operands, consulting set to learn which
// flags consume a following value.
func partition(set *flag.FlagSet, args []string) (flags []string, operands []string, err error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			operands = append(operands, args[i+1:]...)
			return flags, operands, nil
		}
		if len(arg) < 2 || arg[0] != '-' {
			operands = append(operands, arg)
			continue
		}
		name := strings.TrimLeft(arg, "-")
		value, hasInlineValue := "", false
		if idx := strings.IndexByte(name, '='); idx >= 0 {
			name, value, hasInlineValue = name[:idx], name[idx+1:], true
		}
		_ = value

		declared := set.Lookup(name)
		if declared == nil {
			return nil, nil, errs.New(errs.CodeUsage, "unknown flag %q", arg).
				Hint("run `agent-kits help` for the supported flags")
		}
		flags = append(flags, arg)
		if hasInlineValue || isBoolFlag(declared) {
			continue
		}
		if i+1 >= len(args) {
			return nil, nil, errs.New(errs.CodeUsage, "flag %q needs a value", arg)
		}
		i++
		flags = append(flags, args[i])
	}
	return flags, operands, nil
}

// isBoolFlag reports whether a flag may appear without a value.
func isBoolFlag(f *flag.Flag) bool {
	boolean, ok := f.Value.(interface{ IsBoolFlag() bool })
	return ok && boolean.IsBoolFlag()
}

// resolveProject validates the destination directory.
func resolveProject(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		path = "."
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", errs.Wrap(errs.CodeUsage, err, "invalid --project %q", path)
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", errs.New(errs.CodeUsage, "--project %q is not an existing directory", path)
	}
	return abs, nil
}

// environment bundles the catalog-side dependencies a command needs.
type environment struct {
	store   *source.Store
	catalog *catalog.Catalog
	// catalogErr records a load failure that a command may tolerate (doctor) or not.
	catalogErr error
}

// openEnvironment loads sources and the aggregated catalog.
func openEnvironment() (*environment, error) {
	store, err := source.Open()
	if err != nil {
		return nil, err
	}
	env := &environment{store: store}
	env.catalog, env.catalogErr = catalog.NewLoader().Load(store)
	return env, nil
}

// requireCatalog returns the catalog or the failure that prevented loading it.
func (e *environment) requireCatalog() (*catalog.Catalog, error) {
	if e.catalogErr != nil {
		return nil, e.catalogErr
	}
	if e.catalog.Len() == 0 {
		return e.catalog, nil
	}
	return e.catalog, nil
}

// emptyCatalogHint augments a not-found failure when nothing is configured yet.
func emptyCatalogHint(env *environment, err error) error {
	if err == nil || len(env.store.List()) > 0 {
		return err
	}
	var coded *errs.Error
	if errors.As(err, &coded) && coded.Code == errs.CodeNotFound {
		return coded.Hint("no sources are configured; add one with " +
			"`agent-kits source add <name> <url>`")
	}
	return err
}

// confirm asks the user to approve an operation.
//
// When the session is not interactive — an agent, a pipe, or --json — approval must come
// from --yes, so a plan is never applied on a guess.
func (a *App) confirm(opts *options, prompt string) error {
	if opts.yes {
		return nil
	}
	if !a.isInteractive(opts) {
		return errs.New(errs.CodeConfirmationRequired,
			"this plan writes to the project").
			Hint("re-run with --yes to apply it")
	}
	fmt.Fprintf(a.Stdout, "\n%s [y/N]: ", prompt)
	reader := bufio.NewReader(a.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil && answer == "" {
		return errs.New(errs.CodeConfirmationRequired, "no answer was given")
	}
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return nil
	}
	return errs.New(errs.CodeConfirmationRequired, "cancelled")
}

func (a *App) isInteractive(opts *options) bool {
	if a.Interactive != nil {
		return *a.Interactive
	}
	if opts != nil && opts.json {
		return false
	}
	file, ok := a.Stdin.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
