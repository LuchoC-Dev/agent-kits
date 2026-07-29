package git

import (
	"strings"
	"testing"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
)

// D-004: the CLI must have no way to mutate a remote. This test guards the allowlist
// itself, so adding a writing subcommand cannot pass unnoticed.
func TestRunRejectsWritingSubcommands(t *testing.T) {
	forbidden := []string{
		"push", "commit", "tag", "branch", "remote", "merge", "rebase",
		"am", "apply", "cherry-pick", "config", "gc", "filter-branch", "update-ref",
	}
	for _, sub := range forbidden {
		_, err := Run("", sub)
		if err == nil {
			t.Errorf("git %s was allowed", sub)
			continue
		}
		if errs.CodeOf(err) != errs.CodeInternal {
			t.Errorf("git %s error code = %s", sub, errs.CodeOf(err))
		}
		if !strings.Contains(err.Error(), "never writes to a remote") {
			t.Errorf("git %s error = %q", sub, err.Error())
		}
	}
}

func TestAllowlistContainsOnlyReadOrCacheLocalSubcommands(t *testing.T) {
	// clone, fetch, checkout and reset write, but only inside the cache Agent Kits owns.
	expected := map[string]bool{
		"clone": true, "fetch": true, "checkout": true, "reset": true,
		"rev-parse": true, "ls-remote": true, "log": true, "status": true,
		"symbolic-ref": true,
	}
	got := AllowedSubcommands()
	if len(got) != len(expected) {
		t.Fatalf("allowlist = %v", got)
	}
	for _, sub := range got {
		if !expected[sub] {
			t.Errorf("unexpected subcommand in the allowlist: %s", sub)
		}
	}
}

func TestRunRejectsEmptyInvocation(t *testing.T) {
	if _, err := Run(""); err == nil {
		t.Error("an empty invocation was accepted")
	}
}

func TestHeadCommitOnNonRepository(t *testing.T) {
	if commit := HeadCommit(t.TempDir()); commit != "" {
		t.Errorf("HeadCommit on a plain directory = %q", commit)
	}
	if got := Describe(t.TempDir()); !strings.Contains(got, "no commit") {
		t.Errorf("Describe = %q", got)
	}
}
