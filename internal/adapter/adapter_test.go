package adapter

import (
	"testing"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
)

func resource(id string, typ model.Type) *model.Resource {
	return &model.Resource{Manifest: model.Manifest{ID: model.ID(id), Type: typ}}
}

// The destination layout must reproduce what the inherited kits-init flow creates, so a
// workspace stays readable by both surfaces (D-022).
func TestDestinationReproducesLegacyLayout(t *testing.T) {
	a, err := Get("claude-code")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		id   string
		typ  model.Type
		rel  string
		want string
	}{
		{"problem-framing", model.TypeSkill, "SKILL.md", ".agents/skills/problem-framing/SKILL.md"},
		{"problem-framing", model.TypeSkill, "references/a.md", ".agents/skills/problem-framing/references/a.md"},
		{"design-critic", model.TypeAgent, "design-critic.md", ".agents/agents/design-critic.md"},
		{"context/context-builder", model.TypeAgent, "context-builder.md", ".agents/agents/context-builder.md"},
		{"context/context-building", model.TypeWorkflow, "context-building.md", ".agents/workflows/context-building.md"},
		{"context", model.TypeKit, "pack.md", ".agents/packs/context/pack.md"},
	}
	for _, tc := range cases {
		got, err := a.Destination(resource(tc.id, tc.typ), tc.rel)
		if err != nil {
			t.Errorf("Destination(%s, %s) returned %v", tc.id, tc.rel, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Destination(%s, %s) = %q, want %q", tc.id, tc.rel, got, tc.want)
		}
	}
}

func TestDestinationNormalisesSeparatorsAndRejectsEmpty(t *testing.T) {
	a, _ := Get("agents")
	got, err := a.Destination(resource("x", model.TypeSkill), `references\nested\a.md`)
	if err != nil {
		t.Fatalf("Destination returned %v", err)
	}
	if got != ".agents/skills/x/references/nested/a.md" {
		t.Errorf("Destination = %q", got)
	}
	if _, err := a.Destination(resource("x", model.TypeSkill), ""); err == nil {
		t.Error("an empty relative path was accepted")
	}
	if _, err := a.Destination(resource("x", "tool"), "a.md"); errs.CodeOf(err) != errs.CodeRuntimeUnsupported {
		t.Errorf("an unsupported type gave %v", err)
	}
}

// Every supported runtime shares the layout, which is what makes a workspace portable.
func TestAllRuntimesShareTheLayout(t *testing.T) {
	var reference string
	for _, name := range Names() {
		a, err := Get(name)
		if err != nil {
			t.Fatalf("Get(%s) returned %v", name, err)
		}
		got, err := a.Destination(resource("x", model.TypeSkill), "SKILL.md")
		if err != nil {
			t.Fatal(err)
		}
		if reference == "" {
			reference = got
			continue
		}
		if got != reference {
			t.Errorf("%s places a skill at %q, want %q", name, got, reference)
		}
	}
}

func TestGetAndDetect(t *testing.T) {
	if _, err := Get("nope"); errs.CodeOf(err) != errs.CodeRuntimeUnsupported {
		t.Error("an unknown runtime was accepted")
	}

	t.Setenv("CLAUDECODE", "1")
	t.Setenv("OPENCODE", "")
	if got := Detect(); got != "claude-code" {
		t.Errorf("Detect with CLAUDECODE = %q", got)
	}
	auto, err := Get(Auto)
	if err != nil || auto.Name() != "claude-code" {
		t.Errorf("Get(auto) = %v, %v", auto, err)
	}
	if empty, err := Get(""); err != nil || empty.Name() != "claude-code" {
		t.Errorf(`Get("") = %v, %v`, empty, err)
	}

	t.Setenv("CLAUDECODE", "")
	t.Setenv("OPENCODE", "yes")
	if got := Detect(); got != "opencode" {
		t.Errorf("Detect with OPENCODE = %q", got)
	}

	t.Setenv("OPENCODE", "")
	if got := Detect(); got != "agents" {
		t.Errorf("Detect with no signal = %q, want the generic layout", got)
	}
}

func TestMetadataPaths(t *testing.T) {
	a, _ := Get("agents")
	if a.LockPath() != ".agents/agent-kits.lock.json" {
		t.Errorf("LockPath = %q", a.LockPath())
	}
	if a.WorkspacePath() != ".agents/workspace.json" {
		t.Errorf("WorkspacePath = %q", a.WorkspacePath())
	}
}
