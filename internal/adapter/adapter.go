// Package adapter maps canonical resources onto a concrete runtime's filesystem layout.
//
// An adapter decides *where* a resource lands, never *what* it means: it performs no
// content transformation, so a resource installed through any adapter is byte-identical
// to the catalog (02-architecture-direction.md §9).
package adapter

import (
	"os"
	"sort"
	"strings"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
)

// WorkspaceDir is the portable workspace root shared by every supported runtime.
const WorkspaceDir = ".agents"

// LockName is the lockfile's name inside the workspace.
const LockName = "agent-kits.lock.json"

// Auto is the runtime selector that asks the adapter registry to detect the environment.
const Auto = "auto"

// Adapter places resources for one runtime.
type Adapter interface {
	// Name is the runtime identifier recorded in the lockfile.
	Name() string
	// Destination returns the project-relative, slash-separated path for one file of a
	// resource, given that file's path relative to the resource root.
	Destination(res *model.Resource, rel string) (string, error)
	// LockPath is the only metadata file a runtime carries (D-030).
	LockPath() string
}

// workspaceAdapter implements the .agents layout. The three supported runtimes share it
// and differ only in the runtime they declare, which is what keeps a workspace portable
// between Claude Code and OpenCode.
type workspaceAdapter struct{ name string }

func (a workspaceAdapter) Name() string { return a.name }

func (a workspaceAdapter) LockPath() string { return WorkspaceDir + "/" + LockName }

// Destination keeps the portable .agents layout the inherited flow established. The layout
// is now the CLI's own: a project migrated onto lockfile v2 keeps every path it had, so
// retiring workspace.json moves no file (07-cli-only-transition-plan.md §7).
func (a workspaceAdapter) Destination(res *model.Resource, rel string) (string, error) {
	clean := strings.TrimPrefix(strings.ReplaceAll(rel, `\`, "/"), "./")
	if clean == "" {
		return "", errs.New(errs.CodeUnsafePath, "resource %s declares an empty file path", res.Name)
	}
	// The destination comes from the install name, never from the identity: a UUID would
	// make a workspace unreadable, and the name is exactly what the user asked for (D-036).
	switch res.Type {
	case model.TypeSkill:
		return join(WorkspaceDir, "skills", res.Name, clean), nil
	case model.TypeAgent:
		return join(WorkspaceDir, "agents", clean), nil
	case model.TypeWorkflow:
		return join(WorkspaceDir, "workflows", clean), nil
	case model.TypeKit:
		return join(WorkspaceDir, "packs", res.Name, clean), nil
	}
	return "", errs.New(errs.CodeRuntimeUnsupported,
		"runtime %s cannot place a resource of type %q", a.name, res.Type)
}

func join(parts ...string) string { return strings.Join(parts, "/") }

// registry holds the adapters this build supports (D-021).
var registry = map[string]Adapter{
	"agents":      workspaceAdapter{name: "agents"},
	"claude-code": workspaceAdapter{name: "claude-code"},
	"opencode":    workspaceAdapter{name: "opencode"},
}

// Names lists the supported runtimes.
func Names() []string {
	out := make([]string, 0, len(registry))
	for name := range registry {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Get returns the adapter for a runtime name. "auto" and "" detect the environment.
func Get(name string) (Adapter, error) {
	trimmed := strings.TrimSpace(strings.ToLower(name))
	if trimmed == "" || trimmed == Auto {
		return registry[Detect()], nil
	}
	if a, ok := registry[trimmed]; ok {
		return a, nil
	}
	return nil, errs.New(errs.CodeRuntimeUnsupported,
		"unknown runtime %q (supported: %s)", name, strings.Join(Names(), ", "))
}

// Detect infers the host runtime from the environment, using the same signals as the
// inherited kits-init flow. It falls back to the generic layout, which is always valid.
func Detect() string {
	if os.Getenv("CLAUDECODE") == "1" {
		return "claude-code"
	}
	if strings.TrimSpace(os.Getenv("OPENCODE")) != "" {
		return "opencode"
	}
	return "agents"
}
