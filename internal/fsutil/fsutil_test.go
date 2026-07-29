package fsutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChecksumBytesAndFileAgree(t *testing.T) {
	content := []byte("hello\n")
	path := filepath.Join(t.TempDir(), "a.md")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	fromFile, size, err := ChecksumFile(path)
	if err != nil {
		t.Fatalf("ChecksumFile returned %v", err)
	}
	if fromFile != ChecksumBytes(content) {
		t.Errorf("checksums differ: %s vs %s", fromFile, ChecksumBytes(content))
	}
	if size != int64(len(content)) {
		t.Errorf("size = %d", size)
	}
	if !strings.HasPrefix(fromFile, ChecksumPrefix) {
		t.Errorf("checksum is missing its algorithm label: %s", fromFile)
	}
}

// The tree checksum must not depend on iteration order, or a plan would differ between
// runs on identical inputs.
func TestChecksumTreeIsOrderIndependent(t *testing.T) {
	first := ChecksumTree(map[string]string{
		"a.md": "sha256:1", "b/c.md": "sha256:2", "z.md": "sha256:3",
	})
	second := ChecksumTree(map[string]string{
		"z.md": "sha256:3", "a.md": "sha256:1", "b/c.md": "sha256:2",
	})
	if first != second {
		t.Errorf("tree checksums differ: %s vs %s", first, second)
	}

	// A different path with the same content must change the result.
	third := ChecksumTree(map[string]string{
		"a.md": "sha256:1", "b/d.md": "sha256:2", "z.md": "sha256:3",
	})
	if third == first {
		t.Error("renaming a file did not change the tree checksum")
	}
}

func TestWriteFileAtomicCreatesAndReplaces(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "a.md")

	if err := WriteFileAtomic(path, []byte("one\n"), 0o644); err != nil {
		t.Fatalf("WriteFileAtomic returned %v", err)
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != "one\n" {
		t.Fatalf("content = %q, err = %v", got, err)
	}

	// Overwriting an existing file must work on every platform.
	if err := WriteFileAtomic(path, []byte("two\n"), 0o644); err != nil {
		t.Fatalf("overwrite returned %v", err)
	}
	if got, _ := os.ReadFile(path); string(got) != "two\n" {
		t.Errorf("content after overwrite = %q", got)
	}

	// No temporary files may survive a successful write.
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".agent-kits-") {
			t.Errorf("a temporary file survived: %s", entry.Name())
		}
	}
}

func TestReadJSONRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.json")
	if err := os.WriteFile(path, []byte(`{"known":1,"surprise":2}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var target struct {
		Known int `json:"known"`
	}
	if err := ReadJSON(path, &target); err == nil {
		t.Error("an unknown field was accepted")
	}
}

func TestWriteAndReadJSONRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.json")
	type payload struct {
		Name string `json:"name"`
	}
	if err := WriteJSON(path, payload{Name: "x"}); err != nil {
		t.Fatalf("WriteJSON returned %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(raw), "\n") {
		t.Error("JSON files must end with a newline")
	}
	var got payload
	if err := ReadJSON(path, &got); err != nil || got.Name != "x" {
		t.Errorf("round trip = %+v, err = %v", got, err)
	}
}

func TestRemoveEmptyDirsStopsAtTheRoot(t *testing.T) {
	root := t.TempDir()
	stop := filepath.Join(root, ".agents")
	deep := filepath.Join(stop, "skills", "x", "nested")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := RemoveEmptyDirs(deep, stop); err != nil {
		t.Fatalf("RemoveEmptyDirs returned %v", err)
	}
	if Exists(filepath.Join(stop, "skills")) {
		t.Error("empty parents were not removed")
	}
	if !Exists(stop) {
		t.Error("the stop directory must survive")
	}
}

func TestRemoveEmptyDirsKeepsNonEmptyDirectories(t *testing.T) {
	root := t.TempDir()
	stop := filepath.Join(root, ".agents")
	dir := filepath.Join(stop, "skills", "x")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "keep.md"), []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveEmptyDirs(dir, stop); err != nil {
		t.Fatalf("RemoveEmptyDirs returned %v", err)
	}
	if !Exists(dir) {
		t.Error("a directory with content was removed")
	}
}

func TestWalkFilesReportsFilesAndFlagsSymlinks(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"a.md", "nested/b.md", "nested/deep/c.md"} {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	files, symlinks, err := WalkFiles(root)
	if err != nil {
		t.Fatalf("WalkFiles returned %v", err)
	}
	if len(files) != 3 || len(symlinks) != 0 {
		t.Fatalf("files = %v symlinks = %v", files, symlinks)
	}
	// Paths are reported in slash form regardless of platform, and sorted.
	want := []string{"a.md", "nested/b.md", "nested/deep/c.md"}
	for i := range want {
		if files[i] != want[i] {
			t.Errorf("files = %v, want %v", files, want)
			break
		}
	}

	// A symlink is reported separately so callers can refuse it.
	link := filepath.Join(root, "link.md")
	if err := os.Symlink(filepath.Join(root, "a.md"), link); err != nil {
		t.Skipf("symlinks are unavailable in this environment: %v", err)
	}
	files, symlinks, err = WalkFiles(root)
	if err != nil {
		t.Fatalf("WalkFiles returned %v", err)
	}
	if len(symlinks) != 1 || symlinks[0] != "link.md" {
		t.Errorf("symlinks = %v", symlinks)
	}
	for _, file := range files {
		if file == "link.md" {
			t.Error("a symlink was reported as a regular file")
		}
	}
}

func TestIsRegularRejectsDirectories(t *testing.T) {
	dir := t.TempDir()
	if IsRegular(dir) {
		t.Error("a directory was reported as a regular file")
	}
	path := filepath.Join(dir, "a.md")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !IsRegular(path) {
		t.Error("a regular file was not recognised")
	}
}
