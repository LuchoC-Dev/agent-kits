package security

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
)

func TestCheckRelPathRejectsUnsafePaths(t *testing.T) {
	unsafe := []string{
		"",
		"/etc/passwd",
		"../escape.md",
		"nested/../../escape.md",
		"./same.md",
		"a//b.md",
		"C:/Windows/system32",
		"c:relative",
		"\\\\server\\share\\file.md",
		"con.md",
		"nested/NUL.txt",
		"lpt1",
		"has\x00null",
	}
	for _, rel := range unsafe {
		err := CheckRelPath(rel)
		if err == nil {
			t.Errorf("CheckRelPath(%q) accepted an unsafe path", rel)
			continue
		}
		if code := errs.CodeOf(err); code != errs.CodeUnsafePath {
			t.Errorf("CheckRelPath(%q) code = %s", rel, code)
		}
	}
}

func TestCheckRelPathAcceptsOrdinaryPaths(t *testing.T) {
	safe := []string{"SKILL.md", "references/guide.md", "a/b/c/d.json", "pack.md", "console.md"}
	for _, rel := range safe {
		if err := CheckRelPath(rel); err != nil {
			t.Errorf("CheckRelPath(%q) returned %v", rel, err)
		}
	}
}

func TestContainKeepsPathsInsideRoot(t *testing.T) {
	root := t.TempDir()
	got, err := Contain(root, ".agents/skills/x/SKILL.md")
	if err != nil {
		t.Fatalf("Contain returned %v", err)
	}
	want := filepath.Join(root, ".agents", "skills", "x", "SKILL.md")
	if got != want {
		t.Errorf("Contain = %q, want %q", got, want)
	}

	for _, rel := range []string{"../outside.md", "a/../../outside.md"} {
		if _, err := Contain(root, rel); err == nil {
			t.Errorf("Contain(%q) escaped the root", rel)
		}
	}
}

func TestContainRejectsSiblingPrefixEscape(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "project")
	// "project-evil" shares a string prefix with "project"; containment must be decided
	// by path structure, not by prefix comparison.
	if _, err := Contain(root, "../project-evil/file.md"); err == nil {
		t.Error("Contain accepted a sibling directory that shares a prefix")
	}
}

func TestLimits(t *testing.T) {
	limits := Limits{MaxFileBytes: 10, MaxResourceFiles: 2}
	if err := limits.CheckSize("a.md", 10); err != nil {
		t.Errorf("a file at the limit was rejected: %v", err)
	}
	err := limits.CheckSize("a.md", 11)
	if err == nil || errs.CodeOf(err) != errs.CodeUnsafeContent {
		t.Errorf("an oversized file was not rejected: %v", err)
	}
	if err := limits.CheckFileCount("x", 2); err != nil {
		t.Errorf("a resource at the file limit was rejected: %v", err)
	}
	if err := limits.CheckFileCount("x", 3); err == nil {
		t.Error("a resource over the file limit was accepted")
	}

	unlimited := Limits{}
	if err := unlimited.CheckSize("a.md", 1<<40); err != nil {
		t.Errorf("zero means unlimited, got %v", err)
	}
}

func TestScanSecretsBlocksRealCredentials(t *testing.T) {
	cases := map[string]string{
		"private key":   "-----BEGIN RSA PRIVATE KEY-----\nMIIEvQ\n",
		"aws key":       "aws_access_key_id = AKIAIOSFODNN7EXAMPLE\n",
		"github token":  "token: ghp_" + strings.Repeat("a", 36) + "\n",
		"github pat":    "GITHUB_PAT=github_pat_" + strings.Repeat("b", 30) + "\n",
		"slack token":   "xoxb-1234567890-abcdefghij\n",
		"anthropic key": "ANTHROPIC_API_KEY=sk-ant-" + strings.Repeat("c", 30) + "\n",
	}
	for name, content := range cases {
		findings := Blocking(ScanSecrets("doc.md", []byte(content)))
		if len(findings) == 0 {
			t.Errorf("%s: no blocking finding for %q", name, strings.TrimSpace(content))
		}
	}
}

func TestScanSecretsIgnoresDocumentationPlaceholders(t *testing.T) {
	documentation := strings.Join([]string{
		"Set `API_KEY=<your-api-key-here>` in the environment.",
		"export SECRET=your-secret-value-goes-here",
		"password: changeme-please-use-a-real-one",
		"token: xxxxxxxxxxxxxxxxxxxx",
		"api_key: ${SOME_ENV_VARIABLE_NAME}",
	}, "\n")
	if findings := ScanSecrets("doc.md", []byte(documentation)); len(findings) > 0 {
		t.Errorf("placeholders produced findings: %+v", findings)
	}
}

func TestScanSecretsWarnsOnAssignedLiteral(t *testing.T) {
	findings := ScanSecrets("doc.md", []byte("api_key = 8f3a9b2c1d4e5f6a7b8c9d0e\n"))
	if len(findings) != 1 {
		t.Fatalf("findings = %+v", findings)
	}
	if findings[0].Severity != SeverityWarn {
		t.Errorf("severity = %s, want %s", findings[0].Severity, SeverityWarn)
	}
	if len(Blocking(findings)) != 0 {
		t.Error("a warning must not block")
	}
}

func TestScanSecretsSkipsBinaryContent(t *testing.T) {
	binary := append([]byte{0x00, 0x01, 0x02}, []byte("AKIAIOSFODNN7EXAMPLE")...)
	if findings := ScanSecrets("blob.bin", binary); len(findings) != 0 {
		t.Errorf("binary content produced findings: %+v", findings)
	}
}

func TestScanSecretsReportsLineNumbers(t *testing.T) {
	content := "line one\nline two\n-----BEGIN PRIVATE KEY-----\n"
	findings := ScanSecrets("doc.md", []byte(content))
	if len(findings) != 1 || findings[0].Line != 3 {
		t.Fatalf("findings = %+v", findings)
	}
	if !strings.Contains(findings[0].Message(), "doc.md:3") {
		t.Errorf("message = %q", findings[0].Message())
	}
}

// The reserved-device check matters most on Windows but must behave identically
// everywhere, so a catalog cannot pass validation on Linux and break on Windows.
func TestReservedNamesRejectedOnEveryPlatform(t *testing.T) {
	if err := CheckRelPath("aux.md"); err == nil {
		t.Errorf("aux.md was accepted on %s", runtime.GOOS)
	}
}
