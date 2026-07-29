// Package git wraps the system git binary with a read-only subcommand allowlist.
//
// The "no remote writes" invariant (D-004) is enforced here and is verifiable by
// inspection: Run refuses any subcommand outside allowedSubcommands, and that set
// contains nothing that can mutate a remote.
package git

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
)

// allowedSubcommands is the complete set of git subcommands Agent Kits may invoke.
// Every entry either reads, or writes only inside the local cache directory that Agent
// Kits itself owns. `push`, `remote`, `commit`, `tag` and `branch` are absent by design.
var allowedSubcommands = map[string]bool{
	"clone":        true,
	"fetch":        true,
	"checkout":     true, // local working tree of the cache only
	"reset":        true, // local working tree of the cache only
	"rev-parse":    true,
	"ls-remote":    true,
	"log":          true,
	"status":       true,
	"symbolic-ref": true,
}

// AllowedSubcommands lists the allowlist, for documentation and doctor output.
func AllowedSubcommands() []string {
	out := make([]string, 0, len(allowedSubcommands))
	for name := range allowedSubcommands {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Available reports whether a git binary is on PATH.
func Available() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// Run executes a git subcommand in dir and returns its trimmed stdout.
//
// The child environment disables every interactive prompt, so a private source the
// caller is not authorised for fails immediately with source_unavailable instead of
// blocking on a credential dialog.
func Run(dir string, args ...string) (string, error) {
	if len(args) == 0 {
		return "", errs.New(errs.CodeInternal, "git invoked with no subcommand")
	}
	sub := args[0]
	if !allowedSubcommands[sub] {
		return "", errs.New(errs.CodeInternal,
			"git subcommand %q is not allowed; Agent Kits never writes to a remote", sub)
	}
	if !Available() {
		return "", errs.New(errs.CodeSourceUnavailable, "git is not installed or not on PATH")
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=",
		"SSH_ASKPASS=",
		"GCM_INTERACTIVE=never",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		if errors.As(err, &exitErr) {
			return "", errs.New(errs.CodeSourceUnavailable,
				"git %s failed: %s", sub, firstLine(detail))
		}
		return "", errs.Wrap(errs.CodeSourceUnavailable, err, "git %s failed", sub)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

// Clone makes a shallow checkout of ref from url into dir.
func Clone(url, ref, dir string) error {
	args := []string{"clone", "--depth", "1", "--single-branch"}
	if ref != "" {
		args = append(args, "--branch", ref)
	}
	args = append(args, "--", url, dir)
	_, err := Run("", args...)
	return err
}

// Sync fast-forwards an existing shallow checkout to the tip of ref.
//
// reset --hard targets the cache working tree, which Agent Kits owns exclusively; it
// never touches a user project or a remote.
func Sync(dir, ref string) error {
	target := ref
	if target == "" {
		var err error
		target, err = defaultBranch(dir)
		if err != nil {
			return err
		}
	}
	if _, err := Run(dir, "fetch", "--depth", "1", "origin", target); err != nil {
		return err
	}
	_, err := Run(dir, "reset", "--hard", "FETCH_HEAD")
	return err
}

func defaultBranch(dir string) (string, error) {
	out, err := Run(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	if out == "" || out == "HEAD" {
		return "", errs.New(errs.CodeSourceUnavailable,
			"cannot determine the checked-out branch of %s; declare an explicit ref", dir)
	}
	return out, nil
}

// HeadCommit returns the full commit hash checked out in dir, or "" when dir is not a
// Git repository. A source that is a plain directory legitimately has no commit.
func HeadCommit(dir string) string {
	out, err := Run(dir, "rev-parse", "HEAD")
	if err != nil {
		return ""
	}
	return out
}

// IsRepo reports whether dir is inside a Git working tree.
func IsRepo(dir string) bool {
	out, err := Run(dir, "rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}

// Describe renders a short provenance string for messages.
func Describe(dir string) string {
	commit := HeadCommit(dir)
	if commit == "" {
		return "no commit (plain directory)"
	}
	return fmt.Sprintf("commit %s", commit[:min(len(commit), 12)])
}
