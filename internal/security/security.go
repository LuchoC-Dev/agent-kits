// Package security enforces the pre-write guarantees fixed by D-025: every installed
// path stays inside the destination project, no symlink or device file is ever
// materialised, files stay within a size budget, and credentials are detected before
// they reach disk.
//
// Nothing in Agent Kits executes catalog content. This package contains no code path
// that runs, evaluates or interprets a resource file.
package security

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/fsutil"
)

// DefaultMaxFileBytes is the per-file size budget. Catalog resources are documents; a
// file past this size is a signal that something else is being shipped.
const DefaultMaxFileBytes int64 = 2 << 20 // 2 MiB

// DefaultMaxResourceFiles caps how many files a single resource may install.
const DefaultMaxResourceFiles = 512

// Limits bounds what a single resource may write.
type Limits struct {
	MaxFileBytes     int64
	MaxResourceFiles int
}

// DefaultLimits returns the limits used unless a caller overrides them.
func DefaultLimits() Limits {
	return Limits{MaxFileBytes: DefaultMaxFileBytes, MaxResourceFiles: DefaultMaxResourceFiles}
}

// windowsReserved are device names that cannot be used as filenames on Windows. A
// catalog carrying one would make a workspace un-checkoutable on the primary platform.
var windowsReserved = map[string]bool{
	"con": true, "prn": true, "aux": true, "nul": true,
	"com1": true, "com2": true, "com3": true, "com4": true, "com5": true,
	"com6": true, "com7": true, "com8": true, "com9": true,
	"lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true, "lpt5": true,
	"lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true,
}

// CheckRelPath validates a catalog-declared relative path before it is joined to
// anything. It rejects absolute paths, traversal, volume references and NUL bytes.
func CheckRelPath(rel string) error {
	if rel == "" {
		return errs.New(errs.CodeUnsafePath, "empty path")
	}
	if strings.ContainsRune(rel, 0) {
		return errs.New(errs.CodeUnsafePath, "path contains a NUL byte")
	}
	normalised := strings.ReplaceAll(rel, `\`, "/")
	if strings.HasPrefix(normalised, "/") {
		return errs.New(errs.CodeUnsafePath, "absolute path is not allowed: %q", rel)
	}
	if filepath.IsAbs(rel) || filepath.VolumeName(rel) != "" || strings.Contains(normalised, ":") {
		return errs.New(errs.CodeUnsafePath, "volume-qualified path is not allowed: %q", rel)
	}
	for _, segment := range strings.Split(normalised, "/") {
		switch segment {
		case "":
			return errs.New(errs.CodeUnsafePath, "path has an empty segment: %q", rel)
		case ".", "..":
			return errs.New(errs.CodeUnsafePath, "path traversal is not allowed: %q", rel)
		}
		stem := segment
		if i := strings.IndexByte(stem, '.'); i > 0 {
			stem = stem[:i]
		}
		if windowsReserved[strings.ToLower(stem)] {
			return errs.New(errs.CodeUnsafePath,
				"path uses the reserved device name %q: %q", stem, rel)
		}
	}
	return nil
}

// Contain joins rel onto root and verifies the result cannot escape root. It is the only
// sanctioned way to turn a catalog path into an absolute destination.
func Contain(root, rel string) (string, error) {
	if err := CheckRelPath(rel); err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", errs.Wrap(errs.CodeUnsafePath, err, "cannot resolve root %q", root)
	}
	joined := filepath.Join(rootAbs, fsutil.FromSlash(rel))
	cleaned := filepath.Clean(joined)
	relCheck, err := filepath.Rel(rootAbs, cleaned)
	if err != nil {
		return "", errs.Wrap(errs.CodeUnsafePath, err, "cannot relativise %q", rel)
	}
	relCheck = fsutil.ToSlash(relCheck)
	if relCheck == ".." || strings.HasPrefix(relCheck, "../") {
		return "", errs.New(errs.CodeUnsafePath, "path escapes the destination: %q", rel)
	}
	return cleaned, nil
}

// CheckSize enforces the per-file budget.
func (l Limits) CheckSize(rel string, size int64) error {
	if l.MaxFileBytes > 0 && size > l.MaxFileBytes {
		return errs.New(errs.CodeUnsafeContent,
			"file %q is %d bytes, over the %d byte limit", rel, size, l.MaxFileBytes)
	}
	return nil
}

// CheckFileCount enforces the per-resource file budget.
func (l Limits) CheckFileCount(id string, count int) error {
	if l.MaxResourceFiles > 0 && count > l.MaxResourceFiles {
		return errs.New(errs.CodeUnsafeContent,
			"resource %s declares %d files, over the %d file limit",
			id, count, l.MaxResourceFiles)
	}
	return nil
}

// Severity separates findings that stop an operation from findings that are reported.
type Severity string

const (
	// SeverityBlock stops the operation: the match is credential-shaped with high
	// confidence.
	SeverityBlock Severity = "block"
	// SeverityWarn is reported and lets the operation continue.
	SeverityWarn Severity = "warn"
)

// Finding is one secret-scanner hit.
type Finding struct {
	Rule     string   `json:"rule"`
	Severity Severity `json:"severity"`
	Path     string   `json:"path"`
	Line     int      `json:"line"`
}

// Message renders the finding for a diagnostic.
func (f Finding) Message() string {
	return fmt.Sprintf("%s at %s:%d", f.Rule, f.Path, f.Line)
}

type secretRule struct {
	name     string
	severity Severity
	pattern  *regexp.Regexp
}

// secretRules are deterministic, prefix-anchored patterns. Entropy heuristics are
// deliberately excluded: a false positive that blocks an install is worse than a
// narrower ruleset, and every rule here corresponds to a credential format that cannot
// legitimately appear in a catalog document.
var secretRules = []secretRule{
	{"private key block", SeverityBlock, regexp.MustCompile(`-----BEGIN (RSA |EC |DSA |OPENSSH |PGP )?PRIVATE KEY-----`)},
	{"AWS access key id", SeverityBlock, regexp.MustCompile(`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`)},
	{"GitHub token", SeverityBlock, regexp.MustCompile(`\b(?:ghp|gho|ghu|ghs|ghr)_[0-9A-Za-z]{36,}\b`)},
	{"GitHub fine-grained token", SeverityBlock, regexp.MustCompile(`\bgithub_pat_[0-9A-Za-z_]{22,}\b`)},
	{"Slack token", SeverityBlock, regexp.MustCompile(`\bxox[abposr]-[0-9A-Za-z-]{10,}\b`)},
	{"Google API key", SeverityBlock, regexp.MustCompile(`\bAIza[0-9A-Za-z_\-]{35}\b`)},
	{"Anthropic API key", SeverityBlock, regexp.MustCompile(`\bsk-ant-[0-9A-Za-z_\-]{20,}\b`)},
	{"OpenAI API key", SeverityBlock, regexp.MustCompile(`\bsk-(?:proj-)?[0-9A-Za-z_\-]{32,}\b`)},
	{"assigned secret literal", SeverityWarn, regexp.MustCompile(`(?i)\b(?:api[_-]?key|secret|token|password|passwd)\b\s*[:=]\s*["']?[0-9A-Za-z/+_\-]{16,}["']?`)},
}

// placeholderPattern recognises documentation placeholders, which catalog resources use
// constantly when showing how to configure a credential.
var placeholderPattern = regexp.MustCompile(`(?i)(<[^>]*>|\$\{[^}]*\}|\bx{6,}\b|\byour[_-]|\bexample\b|\bplaceholder\b|\bchangeme\b|\bredacted\b|\bdummy\b|\bfake\b|\bsample\b|\.\.\.)`)

// ScanSecrets looks for credential-shaped content in a text file. Binary content is
// skipped: the scanner exists to catch documents that embed a key, and a binary blob is
// already refused elsewhere by the size and type checks.
func ScanSecrets(path string, content []byte) []Finding {
	if isBinary(content) {
		return nil
	}
	var findings []Finding
	for lineNo, line := range strings.Split(string(content), "\n") {
		if len(line) > 4096 {
			line = line[:4096]
		}
		for _, rule := range secretRules {
			if !rule.pattern.MatchString(line) {
				continue
			}
			if rule.severity == SeverityWarn && placeholderPattern.MatchString(line) {
				continue
			}
			findings = append(findings, Finding{
				Rule:     rule.name,
				Severity: rule.severity,
				Path:     path,
				Line:     lineNo + 1,
			})
		}
	}
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Line < findings[j].Line
	})
	return findings
}

// isBinary reports whether content looks non-textual.
func isBinary(content []byte) bool {
	probe := content
	if len(probe) > 8000 {
		probe = probe[:8000]
	}
	return bytes.IndexByte(probe, 0) >= 0
}

// Blocking filters findings that must stop an operation.
func Blocking(findings []Finding) []Finding {
	var out []Finding
	for _, f := range findings {
		if f.Severity == SeverityBlock {
			out = append(out, f)
		}
	}
	return out
}
