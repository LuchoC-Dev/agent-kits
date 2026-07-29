// Package errs defines the stable error vocabulary of Agent Kits.
//
// Every failure surfaced to a caller carries a Code. Codes are part of the public
// contract: agents branch on them, so they are never renamed once released. Each code
// maps to a documented process exit code (see 04-specification.md).
package errs

import (
	"errors"
	"fmt"
	"sort"
)

// Code is a stable, machine-readable failure identifier.
type Code string

const (
	// Registry and resolution.
	CodeNotFound            Code = "not_found"
	CodeAmbiguousID         Code = "ambiguous_id"
	CodeRegistryIntegrity   Code = "registry_integrity_error"
	CodeInvalidManifest     Code = "invalid_manifest"
	CodeDependencyCycle     Code = "dependency_cycle"
	CodeDependencyMissing   Code = "dependency_unresolved"
	CodeVersionConflict     Code = "version_conflict"
	CodeVisibilityViolation Code = "visibility_violation"

	// Planning and installation.
	CodeLocalDivergence      Code = "local_divergence"
	CodeDestinationConflict  Code = "destination_conflict"
	CodeConfirmationRequired Code = "confirmation_required"
	CodeIntegrityMismatch    Code = "integrity_mismatch"
	CodeWorkspaceInvalid     Code = "workspace_invalid"
	CodeNotInstalled         Code = "not_installed"

	// Sources.
	CodeSourceUnavailable Code = "source_unavailable"
	CodeSourceExists      Code = "source_exists"
	CodeSourceUnknown     Code = "source_unknown"

	// Security.
	CodeUnsafePath    Code = "unsafe_path"
	CodeUnsafeContent Code = "unsafe_content"
	CodeUntrusted     Code = "untrusted_source"

	// Runtime and usage.
	CodeRuntimeUnsupported Code = "runtime_unsupported"
	CodeUsage              Code = "usage_error"
	CodeInternal           Code = "internal_error"
)

// Exit codes documented in 04-specification.md §3.
const (
	ExitOK        = 0
	ExitFailure   = 1
	ExitUsage     = 2
	ExitIntegrity = 3
	ExitConflict  = 4
	ExitSource    = 5
	ExitSecurity  = 6
)

var exitByCode = map[Code]int{
	CodeNotFound:            ExitFailure,
	CodeNotInstalled:        ExitFailure,
	CodeAmbiguousID:         ExitIntegrity,
	CodeRegistryIntegrity:   ExitIntegrity,
	CodeInvalidManifest:     ExitIntegrity,
	CodeDependencyCycle:     ExitIntegrity,
	CodeDependencyMissing:   ExitIntegrity,
	CodeVersionConflict:     ExitIntegrity,
	CodeVisibilityViolation: ExitIntegrity,

	CodeLocalDivergence:      ExitConflict,
	CodeDestinationConflict:  ExitConflict,
	CodeConfirmationRequired: ExitConflict,
	CodeIntegrityMismatch:    ExitConflict,
	CodeWorkspaceInvalid:     ExitConflict,

	CodeSourceUnavailable: ExitSource,
	CodeSourceExists:      ExitSource,
	CodeSourceUnknown:     ExitSource,

	CodeUnsafePath:    ExitSecurity,
	CodeUnsafeContent: ExitSecurity,
	CodeUntrusted:     ExitSecurity,

	CodeRuntimeUnsupported: ExitFailure,
	CodeUsage:              ExitUsage,
	CodeInternal:           ExitFailure,
}

// Error is a coded, structured failure.
type Error struct {
	Code    Code           `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
	wrapped error
}

func (e *Error) Error() string { return string(e.Code) + ": " + e.Message }

func (e *Error) Unwrap() error { return e.wrapped }

// With attaches a detail field and returns the same error for chaining.
func (e *Error) With(key string, value any) *Error {
	if e.Details == nil {
		e.Details = map[string]any{}
	}
	e.Details[key] = value
	return e
}

// Hint attaches an actionable remediation message.
func (e *Error) Hint(format string, args ...any) *Error {
	return e.With("hint", fmt.Sprintf(format, args...))
}

// New builds a coded error.
func New(code Code, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// Wrap builds a coded error that preserves an underlying cause.
func Wrap(code Code, cause error, format string, args ...any) *Error {
	if cause == nil {
		return New(code, format, args...)
	}
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...) + ": " + cause.Error(),
		wrapped: cause,
	}
}

// CodeOf reports the code carried by err, or CodeInternal when err is not coded.
func CodeOf(err error) Code {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return CodeInternal
}

// ExitCode maps err to its documented process exit code.
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	if code, ok := exitByCode[CodeOf(err)]; ok {
		return code
	}
	return ExitFailure
}

// Is reports whether err carries the given code.
func Is(err error, code Code) bool { return CodeOf(err) == code }

// Codes lists every defined code, sorted. Used by `agent-kits version --json` so callers
// can discover the vocabulary they are expected to handle.
func Codes() []string {
	out := make([]string, 0, len(exitByCode))
	for code := range exitByCode {
		out = append(out, string(code))
	}
	sort.Strings(out)
	return out
}
