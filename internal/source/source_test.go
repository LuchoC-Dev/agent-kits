package source

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
)

func TestLocalPathRecognisesLocalAndRemoteURLs(t *testing.T) {
	remote := []string{
		"https://github.com/example/agent-kits.git",
		"http://example.com/repo.git",
		"git@github.com:example/repo.git",
		"ssh://git@example.com/repo.git",
		"git://example.com/repo.git",
	}
	for _, url := range remote {
		if got := LocalPath(url); got != "" {
			t.Errorf("LocalPath(%q) = %q, want a remote", url, got)
		}
	}

	dir := t.TempDir()
	for _, url := range []string{dir, "file://" + filepath.ToSlash(dir)} {
		if got := LocalPath(url); got == "" {
			t.Errorf("LocalPath(%q) reported a remote", url)
		}
	}
	if runtime.GOOS != "windows" {
		if got := LocalPath("file:///etc"); got != "/etc" {
			t.Errorf("LocalPath(file:///etc) = %q", got)
		}
	}
	if got := LocalPath(""); got != "" {
		t.Errorf("LocalPath(\"\") = %q", got)
	}
}

func TestSourceValidateDefaultsAndRejections(t *testing.T) {
	src := Source{Name: "public", URL: "https://example.com/repo.git"}
	if err := src.Validate(); err != nil {
		t.Fatalf("Validate returned %v", err)
	}
	if src.Access != model.AccessPublic {
		t.Errorf("access default = %q", src.Access)
	}
	// The safe default is to require review, not to trust an unknown source.
	if src.Trust != model.TrustReview {
		t.Errorf("trust default = %q, want review", src.Trust)
	}

	invalid := []Source{
		{Name: "", URL: "x"},
		{Name: "Public", URL: "x"},
		{Name: "with/slash", URL: "x"},
		{Name: "ok", URL: ""},
		{Name: "ok", URL: "x", Access: "secret"},
		{Name: "ok", URL: "x", Trust: "absolute"},
	}
	for _, candidate := range invalid {
		if err := candidate.Validate(); err == nil {
			t.Errorf("Validate accepted %+v", candidate)
		}
	}
}

func TestStoreRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv(HomeEnv, home)

	store, err := Open()
	if err != nil {
		t.Fatalf("Open returned %v", err)
	}
	if len(store.List()) != 0 {
		t.Fatal("a fresh store must be empty")
	}
	if filepath.Dir(store.ConfigPath()) != home {
		t.Errorf("ConfigPath = %q", store.ConfigPath())
	}

	catalogDir := t.TempDir()
	src := Source{Name: "local", URL: catalogDir, Access: model.AccessPrivate, Trust: model.TrustTrusted}
	if err := store.Add(src); err != nil {
		t.Fatalf("Add returned %v", err)
	}
	if err := store.Add(src); errs.CodeOf(err) != errs.CodeSourceExists {
		t.Errorf("adding twice gave %v", err)
	}

	// A new store must read what the previous one persisted.
	reopened, err := Open()
	if err != nil {
		t.Fatalf("Open returned %v", err)
	}
	got, err := reopened.Get("local")
	if err != nil {
		t.Fatalf("Get returned %v", err)
	}
	if got.Access != model.AccessPrivate || got.Trust != model.TrustTrusted {
		t.Errorf("persisted source = %+v", got)
	}

	if _, err := reopened.Get("absent"); errs.CodeOf(err) != errs.CodeSourceUnknown {
		t.Errorf("Get(absent) = %v", err)
	}
	if err := reopened.Remove("local"); err != nil {
		t.Fatalf("Remove returned %v", err)
	}
	if len(reopened.List()) != 0 {
		t.Error("Remove did not persist")
	}
}

func TestAddRejectsMissingLocalDirectory(t *testing.T) {
	t.Setenv(HomeEnv, t.TempDir())
	store, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(t.TempDir(), "not-here")
	err = store.Add(Source{Name: "local", URL: missing})
	if errs.CodeOf(err) != errs.CodeSourceUnavailable {
		t.Fatalf("err = %v, want source_unavailable", err)
	}
}

func TestResolveLocalAndUnsyncedRemote(t *testing.T) {
	t.Setenv(HomeEnv, t.TempDir())
	store, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	catalogDir := t.TempDir()
	if err := store.Add(Source{Name: "local", URL: catalogDir}); err != nil {
		t.Fatal(err)
	}
	checkout, err := store.Resolve(mustGet(t, store, "local"))
	if err != nil {
		t.Fatalf("Resolve returned %v", err)
	}
	if !checkout.Local || checkout.Root != catalogDir {
		t.Errorf("checkout = %+v", checkout)
	}

	// A remote that has never been synced is unavailable rather than silently empty.
	if err := store.Add(Source{Name: "remote", URL: "https://example.com/repo.git"}); err != nil {
		t.Fatal(err)
	}
	_, err = store.Resolve(mustGet(t, store, "remote"))
	if errs.CodeOf(err) != errs.CodeSourceUnavailable {
		t.Fatalf("err = %v, want source_unavailable", err)
	}

	// ResolveAll must return the readable sources and the failures separately, so one
	// unreachable source cannot hide the rest of the catalog.
	checkouts, failures := store.ResolveAll()
	if len(checkouts) != 1 || len(failures) != 1 {
		t.Errorf("checkouts = %d failures = %d", len(checkouts), len(failures))
	}
}

func TestOpenRejectsUnsupportedSchemaVersion(t *testing.T) {
	home := t.TempDir()
	t.Setenv(HomeEnv, home)
	path := filepath.Join(home, "sources.json")
	if err := writeFile(path, `{"schema_version":99,"sources":[]}`); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(); err == nil {
		t.Error("an unsupported schema_version was accepted")
	}
}

func mustGet(t *testing.T, store *Store, name string) Source {
	t.Helper()
	src, err := store.Get(name)
	if err != nil {
		t.Fatalf("Get(%s) returned %v", name, err)
	}
	return src
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
