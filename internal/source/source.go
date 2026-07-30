// Package source manages the configured catalog sources and their local cache.
//
// A source is always read-only from the CLI's perspective: remote sources are mirrored
// into a cache directory Agent Kits owns, and local sources are read in place and never
// written to.
package source

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/fsutil"
	"github.com/LuchoC-Dev/agent-kits/internal/git"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
)

// ConfigSchemaVersion is the current version of sources.json.
const ConfigSchemaVersion = 1

// HomeEnv overrides the Agent Kits home directory.
const HomeEnv = "AGENT_KITS_HOME"

// Source is one configured catalog origin.
type Source struct {
	Name   string       `json:"name"`
	URL    string       `json:"url"`
	Ref    string       `json:"ref,omitempty"`
	Access model.Access `json:"access"`
	Trust  model.Trust  `json:"trust"`
	// Publishes names the source this one is the published mirror of (D-038).
	//
	// A resource that has been published exists in both, sharing one identity, and that is
	// the normal state rather than a duplicate. Declaring the relationship is what lets the
	// catalog tell "the same resource, published" apart from "two resources claiming one
	// identity", without ever breaking a tie by source order.
	Publishes string `json:"publishes,omitempty"`
}

// Config is the persisted source list.
type Config struct {
	SchemaVersion int      `json:"schema_version"`
	Sources       []Source `json:"sources"`
}

// Validate normalises and checks a source declaration.
//
// A source name is kebab-case for two reasons: it has to work as a directory name in the
// cache, and it is the qualifier of a reference — `acme:frontend-design` — so it must not
// contain the separator (D-036).
func (s *Source) Validate() error {
	s.Name = strings.TrimSpace(s.Name)
	s.URL = strings.TrimSpace(s.URL)
	if _, err := model.ParseName(s.Name); err != nil {
		return errs.New(errs.CodeUsage,
			"invalid source name %q: use lower-case kebab-case", s.Name)
	}
	if s.URL == "" {
		return errs.New(errs.CodeUsage, "source %s declares no url", s.Name)
	}
	switch s.Access {
	case "":
		s.Access = model.AccessPublic
	case model.AccessPublic, model.AccessPrivate:
	default:
		return errs.New(errs.CodeUsage,
			"source %s declares unknown access %q (public|private)", s.Name, s.Access)
	}
	switch s.Trust {
	case "":
		s.Trust = model.TrustReview
	case model.TrustTrusted, model.TrustReview:
	default:
		return errs.New(errs.CodeUsage,
			"source %s declares unknown trust %q (trusted|review)", s.Name, s.Trust)
	}
	s.Publishes = strings.TrimSpace(s.Publishes)
	if s.Publishes != "" {
		if _, err := model.ParseName(s.Publishes); err != nil {
			return errs.New(errs.CodeUsage,
				"source %s declares an invalid origin %q", s.Name, s.Publishes)
		}
		if s.Publishes == s.Name {
			return errs.New(errs.CodeUsage, "source %s cannot publish itself", s.Name)
		}
	}
	return nil
}

// Mirrors reports whether a and b are the two ends of one publication relationship, in
// which case sharing an identity is expected rather than a conflict (D-038).
func Mirrors(a, b Source) bool {
	return (a.Publishes != "" && a.Publishes == b.Name) ||
		(b.Publishes != "" && b.Publishes == a.Name)
}

// IsLocal reports whether the source is a directory on this machine rather than a remote
// repository to mirror.
func (s *Source) IsLocal() bool { return LocalPath(s.URL) != "" }

// LocalPath returns the filesystem path a source URL refers to, or "" when the URL is
// remote. Both `file://` URLs and plain paths are accepted.
func LocalPath(url string) string {
	trimmed := strings.TrimSpace(url)
	if trimmed == "" {
		return ""
	}
	if rest, ok := strings.CutPrefix(trimmed, "file://"); ok {
		// file:///C:/x and file://C:/x both appear in practice.
		rest = strings.TrimPrefix(rest, "/")
		if filepath.VolumeName(rest) == "" && !strings.HasPrefix(trimmed, "file:///") {
			rest = "/" + rest
		}
		return filepath.Clean(fsutil.FromSlash(rest))
	}
	for _, prefix := range []string{"http://", "https://", "ssh://", "git://", "git@"} {
		if strings.HasPrefix(trimmed, prefix) {
			return ""
		}
	}
	if filepath.IsAbs(trimmed) || filepath.VolumeName(trimmed) != "" ||
		strings.HasPrefix(trimmed, ".") || strings.HasPrefix(trimmed, "~") {
		expanded := trimmed
		if strings.HasPrefix(expanded, "~") {
			if home, err := os.UserHomeDir(); err == nil {
				expanded = filepath.Join(home, strings.TrimPrefix(expanded, "~"))
			}
		}
		abs, err := filepath.Abs(fsutil.FromSlash(expanded))
		if err != nil {
			return ""
		}
		return abs
	}
	return ""
}

// Store loads and persists the source configuration and owns the cache layout.
type Store struct {
	home   string
	config Config
}

// Home returns the Agent Kits home directory, honouring AGENT_KITS_HOME.
func Home() (string, error) {
	if custom := strings.TrimSpace(os.Getenv(HomeEnv)); custom != "" {
		abs, err := filepath.Abs(custom)
		if err != nil {
			return "", errs.Wrap(errs.CodeUsage, err, "invalid %s", HomeEnv)
		}
		return abs, nil
	}
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", errs.Wrap(errs.CodeInternal, err, "cannot determine the home directory")
	}
	return filepath.Join(dir, ".agent-kits"), nil
}

// Open loads the store, creating an empty configuration when none exists yet.
func Open() (*Store, error) {
	home, err := Home()
	if err != nil {
		return nil, err
	}
	s := &Store{home: home, config: Config{SchemaVersion: ConfigSchemaVersion}}
	path := s.ConfigPath()
	if !fsutil.Exists(path) {
		return s, nil
	}
	var cfg Config
	if err := fsutil.ReadJSON(path, &cfg); err != nil {
		return nil, errs.Wrap(errs.CodeUsage, err, "cannot read %s", path)
	}
	if cfg.SchemaVersion != ConfigSchemaVersion {
		return nil, errs.New(errs.CodeUsage,
			"unsupported sources.json schema_version %d (expected %d)",
			cfg.SchemaVersion, ConfigSchemaVersion)
	}
	for i := range cfg.Sources {
		if err := cfg.Sources[i].Validate(); err != nil {
			return nil, err
		}
	}
	if err := checkDuplicates(cfg.Sources); err != nil {
		return nil, err
	}
	s.config = cfg
	return s, nil
}

func checkDuplicates(sources []Source) error {
	seen := map[string]bool{}
	for _, src := range sources {
		if seen[src.Name] {
			return errs.New(errs.CodeUsage, "source %s is declared twice", src.Name)
		}
		seen[src.Name] = true
	}
	return nil
}

// ConfigPath is the location of sources.json.
func (s *Store) ConfigPath() string { return filepath.Join(s.home, "sources.json") }

// CacheDir is the root of the local mirror of remote sources.
func (s *Store) CacheDir() string { return filepath.Join(s.home, "cache") }

// HomeDir returns the Agent Kits home directory.
func (s *Store) HomeDir() string { return s.home }

// List returns the configured sources, sorted by name.
func (s *Store) List() []Source {
	out := make([]Source, len(s.config.Sources))
	copy(out, s.config.Sources)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Get returns a source by name.
func (s *Store) Get(name string) (Source, error) {
	for _, src := range s.config.Sources {
		if src.Name == name {
			return src, nil
		}
	}
	return Source{}, errs.New(errs.CodeSourceUnknown, "no source named %q", name).
		Hint("run `agent-kits source list` to see the configured sources")
}

// Add registers a new source.
func (s *Store) Add(src Source) error {
	if err := src.Validate(); err != nil {
		return err
	}
	if _, err := s.Get(src.Name); err == nil {
		return errs.New(errs.CodeSourceExists, "source %s already exists", src.Name)
	}
	if local := LocalPath(src.URL); local != "" {
		info, err := os.Stat(local)
		if err != nil || !info.IsDir() {
			return errs.New(errs.CodeSourceUnavailable,
				"local source path %s is not an existing directory", local)
		}
	}
	s.config.Sources = append(s.config.Sources, src)
	return s.save()
}

// Remove deletes a source and its cache.
func (s *Store) Remove(name string) error {
	src, err := s.Get(name)
	if err != nil {
		return err
	}
	for i := range s.config.Sources {
		if s.config.Sources[i].Name == name {
			s.config.Sources = append(s.config.Sources[:i], s.config.Sources[i+1:]...)
			break
		}
	}
	if !src.IsLocal() {
		if err := os.RemoveAll(s.cachePathFor(name)); err != nil {
			return errs.Wrap(errs.CodeInternal, err, "cannot remove the cache of source %s", name)
		}
	}
	return s.save()
}

func (s *Store) save() error {
	s.config.SchemaVersion = ConfigSchemaVersion
	if err := fsutil.EnsureDir(s.home); err != nil {
		return errs.Wrap(errs.CodeInternal, err, "cannot create %s", s.home)
	}
	if err := fsutil.WriteJSON(s.ConfigPath(), s.config); err != nil {
		return errs.Wrap(errs.CodeInternal, err, "cannot write %s", s.ConfigPath())
	}
	return nil
}

func (s *Store) cachePathFor(name string) string { return filepath.Join(s.CacheDir(), name) }

// Checkout is a source resolved to a readable directory on disk.
type Checkout struct {
	Source Source
	// Root is the directory the catalog is read from.
	Root string
	// Commit is the revision, when the checkout is a Git working tree.
	Commit string
	// Local reports whether Root is the user's own directory rather than a cache mirror.
	Local bool
}

// Resolve returns the readable checkout of a source without contacting the network.
// A remote source that has never been synced is reported as unavailable.
func (s *Store) Resolve(src Source) (Checkout, error) {
	if local := LocalPath(src.URL); local != "" {
		info, err := os.Stat(local)
		if err != nil || !info.IsDir() {
			return Checkout{}, errs.New(errs.CodeSourceUnavailable,
				"source %s points at %s, which is not a readable directory", src.Name, local)
		}
		return Checkout{Source: src, Root: local, Commit: git.HeadCommit(local), Local: true}, nil
	}
	dir := s.cachePathFor(src.Name)
	if !fsutil.Exists(dir) {
		return Checkout{}, errs.New(errs.CodeSourceUnavailable,
			"source %s has never been synced", src.Name).
			Hint("run `agent-kits source sync %s`", src.Name)
	}
	return Checkout{Source: src, Root: dir, Commit: git.HeadCommit(dir)}, nil
}

// ResolveAll resolves every configured source, returning the failures separately so a
// single unreachable private source does not hide the rest of the catalog.
func (s *Store) ResolveAll() (checkouts []Checkout, failures []error) {
	for _, src := range s.List() {
		checkout, err := s.Resolve(src)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		checkouts = append(checkouts, checkout)
	}
	return checkouts, failures
}

// Sync mirrors a remote source into the cache, or verifies a local one. It is the only
// operation that touches the network.
func (s *Store) Sync(src Source) (Checkout, error) {
	if src.IsLocal() {
		return s.Resolve(src)
	}
	dir := s.cachePathFor(src.Name)
	if err := fsutil.EnsureDir(s.CacheDir()); err != nil {
		return Checkout{}, errs.Wrap(errs.CodeInternal, err, "cannot create the cache directory")
	}
	if fsutil.Exists(filepath.Join(dir, ".git")) {
		if err := git.Sync(dir, src.Ref); err != nil {
			return Checkout{}, err
		}
	} else {
		if err := os.RemoveAll(dir); err != nil {
			return Checkout{}, errs.Wrap(errs.CodeInternal, err, "cannot clear the stale cache of %s", src.Name)
		}
		if err := git.Clone(src.URL, src.Ref, dir); err != nil {
			return Checkout{}, err
		}
	}
	return Checkout{Source: src, Root: dir, Commit: git.HeadCommit(dir)}, nil
}
