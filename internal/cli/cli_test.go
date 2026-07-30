package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/internaltest"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
	"github.com/LuchoC-Dev/agent-kits/internal/source"
)

// runner drives the CLI in memory against a throwaway home and source.
type runner struct {
	t         *testing.T
	sourceDir string
	project   string
}

func newRunner(t *testing.T, resources ...internaltest.Resource) *runner {
	t.Helper()
	t.Setenv(source.HomeEnv, t.TempDir())
	// A deterministic runtime keeps assertions independent of the host environment.
	t.Setenv("CLAUDECODE", "")
	t.Setenv("OPENCODE", "")

	sourceDir := t.TempDir()
	internaltest.WriteNativeSource(t, sourceDir, resources...)
	r := &runner{t: t, sourceDir: sourceDir, project: t.TempDir()}
	r.mustRun("source", "add", "public", sourceDir, "--trust", "trusted")
	return r
}

// run executes a command and returns its exit code and streams.
func (r *runner) run(args ...string) (int, string, string) {
	r.t.Helper()
	var stdout, stderr bytes.Buffer
	interactive := false
	app := &App{
		Stdout: &stdout, Stderr: &stderr, Stdin: strings.NewReader(""),
		Interactive: &interactive,
	}
	code := app.Run(args)
	return code, stdout.String(), stderr.String()
}

func (r *runner) mustRun(args ...string) string {
	r.t.Helper()
	code, stdout, stderr := r.run(args...)
	if code != errs.ExitOK {
		r.t.Fatalf("%v exited %d\nstdout: %s\nstderr: %s", args, code, stdout, stderr)
	}
	return stdout
}

// decode parses a JSON envelope from stdout.
func decode(t *testing.T, stdout string) envelope {
	t.Helper()
	var result envelope
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("stdout is not a single JSON document: %v\n%s", err, stdout)
	}
	return result
}

func sampleCatalog() []internaltest.Resource {
	return []internaltest.Resource{
		{
			Name: "problem-framing", Type: model.TypeSkill, Version: "1.0.0",
			Description: "Define the real problem",
			Files:       map[string]string{"SKILL.md": "# framing\n"},
		},
		{
			Name: "context-builder", Type: model.TypeAgent, Version: "1.0.0",
			Files: map[string]string{"context-builder.md": "# builder\n"},
		},
		{
			Name: "context", Type: model.TypeKit, Version: "1.0.0",
			Description: "Context kit",
			Files:       map[string]string{"pack.md": "# pack\n"},
			Dependencies: []model.Dependency{
				internaltest.Dep("problem-framing"),
				internaltest.Dep("context-builder"),
			},
		},
		{Name: "solo", Type: model.TypeWorkflow, Version: "1.0.0"},
	}
}

func TestVersionEmitsContracts(t *testing.T) {
	r := newRunner(t, sampleCatalog()...)
	result := decode(t, r.mustRun("version", "--json"))
	if !result.OK || result.Command != "version" {
		t.Fatalf("envelope = %+v", result)
	}
	data, ok := result.Data.(map[string]any)
	if !ok {
		t.Fatalf("data = %T", result.Data)
	}
	for _, key := range []string{"version", "runtimes", "types", "error_codes", "lock_schema"} {
		if _, present := data[key]; !present {
			t.Errorf("version data is missing %q", key)
		}
	}
	if data["remote_writes"] != false {
		t.Error("version must advertise that remote writes are impossible")
	}
}

func TestSourceLifecycle(t *testing.T) {
	r := newRunner(t, sampleCatalog()...)

	listed := decode(t, r.mustRun("source", "list", "--json"))
	sources := listed.Data.(map[string]any)["sources"].([]any)
	if len(sources) != 1 {
		t.Fatalf("sources = %v", sources)
	}
	first := sources[0].(map[string]any)
	if first["name"] != "public" || first["local"] != true {
		t.Errorf("source = %+v", first)
	}
	if first["resources"].(float64) != float64(len(sampleCatalog())) {
		t.Errorf("resource count = %v", first["resources"])
	}

	// A duplicate name is refused.
	code, _, _ := r.run("source", "add", "public", r.sourceDir)
	if code != errs.ExitSource {
		t.Errorf("adding a duplicate source exited %d", code)
	}

	r.mustRun("source", "remove", "public")
	emptied := decode(t, r.mustRun("source", "list", "--json"))
	if len(emptied.Data.(map[string]any)["sources"].([]any)) != 0 {
		t.Error("the source was not removed")
	}
}

func TestSearchAndInfo(t *testing.T) {
	r := newRunner(t, sampleCatalog()...)

	found := decode(t, r.mustRun("search", "problem", "--json"))
	results := found.Data.(map[string]any)["results"].([]any)
	if len(results) != 1 || results[0].(map[string]any)["name"] != "problem-framing" {
		t.Fatalf("results = %v", results)
	}

	typed := decode(t, r.mustRun("search", "--type", "kit", "--json"))
	if len(typed.Data.(map[string]any)["results"].([]any)) != 1 {
		t.Errorf("type filter results = %v", typed.Data)
	}

	code, _, _ := r.run("search", "--type", "tool", "--json")
	if code != errs.ExitUsage {
		t.Errorf("an unsupported type exited %d", code)
	}

	info := decode(t, r.mustRun("info", "context", "--json"))
	data := info.Data.(map[string]any)
	if data["type"] != "kit" || len(data["dependencies"].([]any)) != 2 {
		t.Errorf("info = %+v", data)
	}

	// Human output must mention the resource without needing JSON.
	human := r.mustRun("info", "problem-framing")
	if !strings.Contains(human, "problem-framing") || !strings.Contains(human, "Define the real problem") {
		t.Errorf("human info output = %q", human)
	}
}

// D-010: plan writes nothing.
func TestPlanDoesNotWrite(t *testing.T) {
	r := newRunner(t, sampleCatalog()...)
	result := decode(t, r.mustRun("plan", "context", "--project", r.project, "--json"))
	if !result.OK {
		t.Fatalf("envelope = %+v", result)
	}
	changes := result.Data.(map[string]any)["changes"].([]any)
	if len(changes) != 3 {
		t.Errorf("changes = %v", changes)
	}
	if internaltest.Exists(r.project, ".agents") {
		t.Error("plan created files in the project")
	}
}

// D-009 and D-010: a non-interactive session must pass --yes to write.
func TestInstallRequiresConfirmation(t *testing.T) {
	r := newRunner(t, sampleCatalog()...)
	code, _, _ := r.run("install", "context", "--project", r.project)
	if code != errs.ExitConflict {
		t.Fatalf("exit = %d, want %d", code, errs.ExitConflict)
	}
	if internaltest.Exists(r.project, ".agents/workspace.json") {
		t.Error("an unconfirmed install wrote to the project")
	}
}

func TestInstallListRepeatAndRemove(t *testing.T) {
	r := newRunner(t, sampleCatalog()...)

	// Flags after operands must parse, the way an agent tends to compose a command.
	installed := decode(t, r.mustRun("install", "context", "--project", r.project, "--yes", "--json"))
	data := installed.Data.(map[string]any)
	if data["changed"] != true {
		t.Fatalf("install data = %+v", data)
	}
	for _, path := range []string{
		".agents/skills/problem-framing/SKILL.md",
		".agents/agents/context-builder.md",
		".agents/packs/context/pack.md",
		".agents/agent-kits.lock.json",
	} {
		if !internaltest.Exists(r.project, path) {
			t.Errorf("%s was not installed", path)
		}
	}
	// No normal command creates workspace.json any more (D-030).
	if internaltest.Exists(r.project, ".agents/workspace.json") {
		t.Error("install created a second state file")
	}

	listed := decode(t, r.mustRun("list", "--project", r.project, "--json"))
	if listed.Data.(map[string]any)["count"].(float64) != 3 {
		t.Errorf("list count = %v", listed.Data)
	}

	// RF-08: the second install reports no change.
	repeated := decode(t, r.mustRun("install", "context", "--project", r.project, "--yes", "--json"))
	if repeated.Data.(map[string]any)["changed"] != false {
		t.Errorf("a repeated install reported a change: %+v", repeated.Data)
	}

	r.mustRun("doctor", "--project", r.project, "--json")

	removed := decode(t, r.mustRun("remove", "context", "--project", r.project, "--yes", "--json"))
	if removed.Data.(map[string]any)["changed"] != true {
		t.Errorf("remove data = %+v", removed.Data)
	}
	if internaltest.Exists(r.project, ".agents/skills/problem-framing/SKILL.md") {
		t.Error("removal left files behind")
	}
}

func TestUpdatePicksUpNewVersions(t *testing.T) {
	r := newRunner(t, sampleCatalog()...)
	r.mustRun("install", "problem-framing", "--project", r.project, "--yes")

	// Publish a new version of the installed skill.
	internaltest.WriteNativeSource(t, r.sourceDir, internaltest.Resource{
		Name: "problem-framing", Type: model.TypeSkill, Version: "1.1.0",
		Files: map[string]string{"SKILL.md": "# framing v2\n"},
	})

	listed := decode(t, r.mustRun("list", "--project", r.project, "--json"))
	managed := listed.Data.(map[string]any)["managed"].([]any)
	if managed[0].(map[string]any)["available"] != "1.1.0" {
		t.Errorf("list should advertise the update: %+v", managed[0])
	}

	updated := decode(t, r.mustRun("update", "--project", r.project, "--yes", "--json"))
	if updated.Data.(map[string]any)["changed"] != true {
		t.Fatalf("update data = %+v", updated.Data)
	}
	if got := internaltest.ReadFile(t, r.project, ".agents/skills/problem-framing/SKILL.md"); got != "# framing v2\n" {
		t.Errorf("content after update = %q", got)
	}
}

func TestDoctorExitsNonZeroWithASingleEnvelope(t *testing.T) {
	r := newRunner(t, sampleCatalog()...)
	r.mustRun("install", "context", "--project", r.project, "--yes")
	internaltest.WriteFile(t, r.project, ".agents/skills/problem-framing/SKILL.md", "edited\n")

	code, stdout, _ := r.run("doctor", "--project", r.project, "--json")
	if code != errs.ExitConflict {
		t.Errorf("exit = %d, want %d", code, errs.ExitConflict)
	}
	result := decode(t, stdout)
	if result.OK {
		t.Error("the envelope should report failure")
	}
	if result.Error == nil || len(result.Data.(map[string]any)["problems"].([]any)) == 0 {
		t.Errorf("envelope = %+v", result)
	}
}

// D-036: a name two sources offer is ambiguous, and the CLI refuses to choose.
func TestAmbiguousReferenceFailsClosed(t *testing.T) {
	r := newRunner(t, sampleCatalog()...)

	// A second source offering the same name as the first.
	other := t.TempDir()
	internaltest.WriteNativeSource(t, other, internaltest.Resource{
		Name: "problem-framing", ID: "3f1c2b7a-9d84-4e11-b6f2-77c1a9e0d512",
		Type: model.TypeSkill, Version: "2.0.0",
		Files: map[string]string{"SKILL.md": "# other framing\n"},
	})
	r.mustRun("source", "add", "acme", other, "--trust", "trusted")

	code, stdout, _ := r.run("plan", "problem-framing", "--project", r.project, "--json")
	if code != errs.ExitIntegrity {
		t.Fatalf("exit = %d, want %d\n%s", code, errs.ExitIntegrity, stdout)
	}
	result := decode(t, stdout)
	if result.OK || result.Error.Code != errs.CodeAmbiguousID {
		t.Fatalf("envelope = %+v", result)
	}
	candidates, ok := result.Error.Details["candidates"].([]any)
	if !ok || len(candidates) != 2 {
		t.Fatalf("details = %+v", result.Error.Details)
	}
	if candidates[0] != "acme:problem-framing" {
		t.Errorf("candidates must be qualified by source: %v", candidates)
	}

	// Qualifying the reference resolves it.
	resolved := decode(t, r.mustRun("plan", "acme:problem-framing", "--project", r.project, "--json"))
	if !resolved.OK {
		t.Fatalf("a qualified reference failed: %+v", resolved)
	}
}

func TestUnknownCommandsAndFlags(t *testing.T) {
	r := newRunner(t, sampleCatalog()...)
	if code, _, _ := r.run("frobnicate"); code != errs.ExitUsage {
		t.Errorf("unknown command exited %d", code)
	}
	if code, _, _ := r.run("search", "--nope"); code != errs.ExitUsage {
		t.Errorf("unknown flag exited %d", code)
	}
	if code, _, _ := r.run("install", "--project", r.project); code != errs.ExitUsage {
		t.Errorf("install without an id exited %d", code)
	}
	if code, _, _ := r.run(); code != errs.ExitUsage {
		t.Errorf("no arguments exited %d", code)
	}
	if code, stdout, _ := r.run("help"); code != errs.ExitOK || !strings.Contains(stdout, "usage:") {
		t.Errorf("help exited %d with %q", code, stdout)
	}
}

func TestRuntimeSelection(t *testing.T) {
	r := newRunner(t, sampleCatalog()...)
	if code, _, _ := r.run("plan", "context", "--project", r.project, "--runtime", "nope"); code != errs.ExitFailure {
		t.Errorf("an unknown runtime exited %d", code)
	}
	result := decode(t, r.mustRun("plan", "context", "--project", r.project,
		"--runtime", "opencode", "--json"))
	if result.Data.(map[string]any)["runtime"] != "opencode" {
		t.Errorf("runtime = %+v", result.Data)
	}
}

func TestProjectMustExist(t *testing.T) {
	r := newRunner(t, sampleCatalog()...)
	if code, _, _ := r.run("plan", "context", "--project", r.project+"/missing"); code != errs.ExitUsage {
		t.Error("a nonexistent project directory was accepted")
	}
}

// The migration window is closed: `migrate` and `import` no longer exist (D-041).
func TestTheMigrationSurfaceIsGone(t *testing.T) {
	r := newRunner(t, sampleCatalog()...)
	for _, command := range []string{"migrate", "import"} {
		code, _, stderr := r.run(command, "--project", r.project)
		if code != errs.ExitUsage {
			t.Errorf("%s exited %d, want %d", command, code, errs.ExitUsage)
		}
		if !strings.Contains(stderr, "unknown command") {
			t.Errorf("%s stderr = %q", command, stderr)
		}
	}
	if strings.Contains(r.mustRun("help"), "migrate") {
		t.Error("the usage text still offers migrate")
	}
}

// A workspace.json left behind is a foreign file now: nothing reads it, and nothing that
// writes to the project is blocked by it.
func TestALeftoverWorkspaceDescriptorIsIgnored(t *testing.T) {
	r := newRunner(t, sampleCatalog()...)
	internaltest.WriteFile(t, r.project, ".agents/workspace.json", `{"$schema_version": 2}`)

	installed := decode(t, r.mustRun("install", "context", "--project", r.project, "--yes", "--json"))
	if installed.Data.(map[string]any)["changed"] != true {
		t.Fatalf("install data = %+v", installed.Data)
	}
	if got := internaltest.ReadFile(t, r.project, ".agents/workspace.json"); got != `{"$schema_version": 2}` {
		t.Errorf("a foreign file was modified: %q", got)
	}
	// And doctor no longer reports it as a problem.
	r.mustRun("doctor", "--project", r.project, "--json")
}

func TestEmptyCatalogGivesAnActionableHint(t *testing.T) {
	t.Setenv(source.HomeEnv, t.TempDir())
	var stdout, stderr bytes.Buffer
	interactive := false
	app := &App{Stdout: &stdout, Stderr: &stderr, Stdin: strings.NewReader(""), Interactive: &interactive}

	code := app.Run([]string{"info", "anything"})
	if code != errs.ExitFailure {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stderr.String(), "source add") {
		t.Errorf("stderr = %q", stderr.String())
	}
}
