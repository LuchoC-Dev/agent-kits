package catalog

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/frontmatter"
	"github.com/LuchoC-Dev/agent-kits/internal/fsutil"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
	"github.com/LuchoC-Dev/agent-kits/internal/security"
	"github.com/LuchoC-Dev/agent-kits/internal/semver"
	"github.com/LuchoC-Dev/agent-kits/internal/source"
)

// legacyVersion is the version assigned to inherited resources that declare none.
// A synthetic 0.0.0 keeps them orderable without pretending they are released.
const legacyVersion = "0.0.0"

// legacyEntry is a resource discovered in the inherited layout, before its dependencies
// have been wired. Wiring needs the full index, so discovery and wiring are two passes.
type legacyEntry struct {
	id    model.ID
	typ   model.Type
	root  string
	files []string
	fm    *frontmatter.Value
	owner string
}

// legacyIndex maps discovered ids and bare names, so textual references in the inherited
// frontmatter can be turned into canonical dependencies.
type legacyIndex struct {
	byID   map[model.ID]*legacyEntry
	byName map[string][]model.ID
}

func newLegacyIndex() *legacyIndex {
	return &legacyIndex{byID: map[model.ID]*legacyEntry{}, byName: map[string][]model.ID{}}
}

func (idx *legacyIndex) put(entry *legacyEntry) {
	idx.byID[entry.id] = entry
	name := entry.id.Name()
	idx.byName[name] = append(idx.byName[name], entry.id)
}

// resolve turns a textual reference into a canonical id.
//
// The global pool is consulted first and the owning kit second, which is the order the
// inherited kits-init flow uses: an agent id is looked up in the shared pool, and only
// then in the kit that references it.
func (idx *legacyIndex) resolve(ref, owner string, want model.Type) (model.ID, bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", false
	}
	candidates := []model.ID{model.ID(ref)}
	if owner != "" {
		candidates = append(candidates, model.ID(owner+"/"+ref))
	}
	for _, candidate := range candidates {
		if entry, ok := idx.byID[candidate]; ok && (want == "" || entry.typ == want) {
			return entry.id, true
		}
	}
	// Fall back to a unique bare-name match of the wanted type.
	var matches []model.ID
	for _, id := range idx.byName[ref] {
		if want == "" || idx.byID[id].typ == want {
			matches = append(matches, id)
		}
	}
	if len(matches) == 1 {
		return matches[0], true
	}
	return "", false
}

// loadLegacy reads the inherited Markdown catalog described in 06-legacy-baseline.md.
func (l *Loader) loadLegacy(checkout source.Checkout, cat *Catalog) (bool, error) {
	root := checkout.Root
	entries, err := l.discoverLegacy(root)
	if err != nil {
		return false, err
	}
	if len(entries) == 0 {
		return false, nil
	}

	idx := newLegacyIndex()
	for _, entry := range entries {
		idx.put(entry)
	}
	ids := make([]model.ID, 0, len(idx.byID))
	for id := range idx.byID {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	for _, id := range ids {
		entry := idx.byID[id]
		res, err := l.buildLegacyResource(checkout, entry, idx, cat)
		if err != nil {
			return true, err
		}
		if err := cat.add(res); err != nil {
			return true, err
		}
	}
	return true, nil
}

// discoverLegacy walks the inherited layout and returns one entry per resource.
func (l *Loader) discoverLegacy(root string) ([]*legacyEntry, error) {
	var entries []*legacyEntry

	// Skills: skills/<id>/SKILL.md, with the directory name as identity.
	skillDirs, err := subdirs(filepath.Join(root, "skills"))
	if err != nil {
		return nil, err
	}
	for _, dir := range skillDirs {
		manifestPath := filepath.Join(dir, "SKILL.md")
		if !fsutil.IsRegular(manifestPath) || hasNativeManifest(dir) {
			continue
		}
		fm, err := readFrontmatter(manifestPath)
		if err != nil {
			return nil, err
		}
		files, err := l.legacyFiles(dir, filepath.Base(dir))
		if err != nil {
			return nil, err
		}
		id, err := model.ParseID(filepath.Base(dir))
		if err != nil {
			return nil, err
		}
		entries = append(entries, &legacyEntry{
			id: id, typ: model.TypeSkill, root: dir, files: files, fm: fm,
		})
	}

	// Global agents: agents/<id>.md.
	agentEntries, err := l.discoverLegacyAgents(filepath.Join(root, "agents"), "")
	if err != nil {
		return nil, err
	}
	entries = append(entries, agentEntries...)

	// Kits: packs/<id>/pack.md, plus the agents and workflows the kit owns.
	packDirs, err := subdirs(filepath.Join(root, "packs"))
	if err != nil {
		return nil, err
	}
	for _, dir := range packDirs {
		manifestPath := filepath.Join(dir, "pack.md")
		if !fsutil.IsRegular(manifestPath) || hasNativeManifest(dir) {
			continue
		}
		fm, err := readFrontmatter(manifestPath)
		if err != nil {
			return nil, err
		}
		name := fm.Get("id").String()
		if name == "" {
			name = filepath.Base(dir)
		}
		id, err := model.ParseID(name)
		if err != nil {
			return nil, err
		}
		// A kit owns its own namespace, so its references resolve against it: this is how
		// `packs/backend` finds `backend/feature-development` rather than colliding with
		// the identically named workflow in `packs/frontend`.
		entries = append(entries, &legacyEntry{
			id: id, typ: model.TypeKit, root: dir, files: []string{"pack.md"}, fm: fm,
			owner: string(id),
		})

		owned, err := l.discoverLegacyAgents(filepath.Join(dir, "agents"), string(id))
		if err != nil {
			return nil, err
		}
		entries = append(entries, owned...)

		workflows, err := l.discoverLegacyWorkflows(filepath.Join(dir, "workflows"), string(id))
		if err != nil {
			return nil, err
		}
		entries = append(entries, workflows...)
	}
	return entries, nil
}

func (l *Loader) discoverLegacyAgents(dir, owner string) ([]*legacyEntry, error) {
	return l.discoverMarkdownEntries(dir, owner, model.TypeAgent)
}

func (l *Loader) discoverLegacyWorkflows(dir, owner string) ([]*legacyEntry, error) {
	return l.discoverMarkdownEntries(dir, owner, model.TypeWorkflow)
}

// discoverMarkdownEntries reads a flat directory of single-file resources.
func (l *Loader) discoverMarkdownEntries(dir, owner string, typ model.Type) ([]*legacyEntry, error) {
	names, err := markdownFiles(dir)
	if err != nil {
		return nil, err
	}
	var entries []*legacyEntry
	for _, name := range names {
		path := filepath.Join(dir, name)
		fm, err := readFrontmatter(path)
		if err != nil {
			return nil, err
		}
		local := fm.Get("id").String()
		if local == "" {
			local = strings.TrimSuffix(name, filepath.Ext(name))
		}
		canonical := local
		if owner != "" {
			canonical = owner + "/" + local
		}
		id, err := model.ParseID(canonical)
		if err != nil {
			return nil, err
		}
		info, statErr := os.Lstat(path)
		if statErr != nil {
			return nil, errs.Wrap(errs.CodeSourceUnavailable, statErr, "cannot stat %s", path)
		}
		if err := l.Limits.CheckSize(name, info.Size()); err != nil {
			return nil, err
		}
		entries = append(entries, &legacyEntry{
			id: id, typ: typ, root: dir, files: []string{name}, fm: fm, owner: owner,
		})
	}
	return entries, nil
}

// buildLegacyResource turns a discovered entry into a canonical resource.
func (l *Loader) buildLegacyResource(
	checkout source.Checkout, entry *legacyEntry, idx *legacyIndex, cat *Catalog,
) (*model.Resource, error) {
	fm := entry.fm
	manifest := model.Manifest{
		SchemaVersion: model.ManifestSchemaVersion,
		ID:            entry.id,
		Type:          entry.typ,
		Name:          displayName(fm, entry.id),
		Version:       versionOf(fm),
		Description:   fm.Get("description").Text(),
		Files:         entry.files,
		Traits:        legacyTraits(fm),
		Labels:        legacyLabels(fm),
		Produces:      artifacts(fm.Get("produces")),
		Consumes:      artifacts(fm.Get("consumes")),
	}
	manifest.Dependencies = l.legacyDependencies(entry, idx, cat)

	if err := manifest.Validate(); err != nil {
		return nil, err
	}
	return &model.Resource{
		Manifest: manifest,
		Source:   checkout.Source.Name,
		Root:     entry.root,
		Commit:   checkout.Commit,
		Access:   checkout.Source.Access,
		Trust:    checkout.Source.Trust,
		Legacy:   true,
	}, nil
}

// legacyDependencies derives the dependency graph from the inherited frontmatter.
//
// A reference that cannot be resolved is recorded as a diagnostic instead of a hard
// failure: the inherited catalog documents a few resources it does not ship, and refusing
// to load the whole source over one dangling name would make the catalog unusable.
func (l *Loader) legacyDependencies(
	entry *legacyEntry, idx *legacyIndex, cat *Catalog,
) []model.Dependency {
	fm := entry.fm
	seen := map[model.ID]bool{}
	var deps []model.Dependency

	addRef := func(ref string, want model.Type) {
		id, ok := idx.resolve(ref, entry.owner, want)
		if !ok {
			cat.warn(errs.CodeDependencyMissing, string(entry.id),
				"references unknown "+string(want)+" "+ref)
			return
		}
		if id == entry.id || seen[id] {
			return
		}
		seen[id] = true
		deps = append(deps, model.Dependency{ID: id})
	}

	switch entry.typ {
	case model.TypeSkill:
		// A discipline skill composes its sub-skills.
		for _, ref := range fm.Get("composes").Strings() {
			addRef(ref, model.TypeSkill)
		}

	case model.TypeAgent:
		for _, ref := range fm.Get("skills").StringsOf("id") {
			addRef(ref, model.TypeSkill)
		}
		for _, ref := range fm.Get("uses_agents").Strings() {
			addRef(ref, model.TypeAgent)
		}
		for _, ref := range fm.Get("workflows").StringsOf("id") {
			addRef(ref, model.TypeWorkflow)
		}

	case model.TypeWorkflow:
		for _, ref := range fm.Get("agent").Strings() {
			addRef(ref, model.TypeAgent)
		}
		for _, ref := range fm.Get("skills").StringsOf("id") {
			addRef(ref, model.TypeSkill)
		}
		for _, group := range []string{"steps", "phases", "shared_phases", "rapido_phases"} {
			for _, ref := range fm.Get(group).StringsOf("skill") {
				addRef(ref, model.TypeSkill)
			}
		}

	case model.TypeKit:
		for _, ref := range fm.Get("depends_on").Strings() {
			addRef(ref, model.TypeKit)
		}
		for _, ref := range fm.Get("skills").StringsOf("id") {
			addRef(ref, model.TypeSkill)
		}
		for _, ref := range fm.Get("agents").StringsOf("id") {
			addRef(ref, model.TypeAgent)
		}
		for _, ref := range fm.Get("workflows").StringsOf("id") {
			addRef(ref, model.TypeWorkflow)
		}
	}
	sort.SliceStable(deps, func(i, j int) bool { return deps[i].ID < deps[j].ID })
	return deps
}

// legacyFiles lists every file of a multi-file resource such as a skill directory.
func (l *Loader) legacyFiles(dir, id string) ([]string, error) {
	files, symlinks, err := fsutil.WalkFiles(dir)
	if err != nil {
		return nil, errs.Wrap(errs.CodeSourceUnavailable, err, "cannot read %s", dir)
	}
	if len(symlinks) > 0 {
		return nil, errs.New(errs.CodeUnsafePath,
			"resource %s contains symlinks or special files: %s", id, strings.Join(symlinks, ", "))
	}
	if err := l.Limits.CheckFileCount(id, len(files)); err != nil {
		return nil, err
	}
	for _, rel := range files {
		if err := security.CheckRelPath(rel); err != nil {
			return nil, err
		}
		info, statErr := os.Lstat(filepath.Join(dir, fsutil.FromSlash(rel)))
		if statErr != nil {
			return nil, errs.Wrap(errs.CodeSourceUnavailable, statErr, "cannot stat %s in %s", rel, dir)
		}
		if err := l.Limits.CheckSize(rel, info.Size()); err != nil {
			return nil, err
		}
	}
	return files, nil
}

func displayName(fm *frontmatter.Value, id model.ID) string {
	for _, key := range []string{"name", "title"} {
		if value := fm.Get(key).String(); value != "" && value != string(id) {
			return value
		}
	}
	return ""
}

// versionOf reads a declared version, tolerating the two-component form some inherited
// resources use, and falls back to the synthetic legacy version.
func versionOf(fm *frontmatter.Value) string {
	raw := strings.TrimSpace(fm.Get("version").String())
	if raw == "" {
		if meta := fm.Get("metadata"); meta != nil {
			raw = strings.TrimSpace(meta.Get("version").String())
		}
	}
	if raw == "" {
		return legacyVersion
	}
	if _, err := semver.Parse(raw); err == nil {
		return strings.TrimPrefix(raw, "v")
	}
	if parts := strings.Split(strings.TrimPrefix(raw, "v"), "."); len(parts) == 2 {
		if _, errMajor := strconv.Atoi(parts[0]); errMajor == nil {
			if _, errMinor := strconv.Atoi(parts[1]); errMinor == nil {
				candidate := raw + ".0"
				if _, err := semver.Parse(candidate); err == nil {
					return strings.TrimPrefix(candidate, "v")
				}
			}
		}
	}
	return legacyVersion
}

func legacyTraits(fm *frontmatter.Value) map[string]bool {
	traits := map[string]bool{}
	for _, key := range []string{"discipline", "combinable"} {
		if value := fm.Get(key); value != nil && value.Bool() {
			traits[key] = true
		}
	}
	if len(traits) == 0 {
		return nil
	}
	return traits
}

func legacyLabels(fm *frontmatter.Value) map[string]string {
	labels := map[string]string{}
	for _, key := range []string{"class", "license", "category", "risk", "date_added", "source"} {
		if value := fm.Get(key).String(); value != "" {
			labels[key] = value
		}
	}
	if meta := fm.Get("metadata"); meta != nil {
		for _, key := range meta.Keys() {
			if value := meta.Get(key).String(); value != "" {
				labels["metadata."+key] = value
			}
		}
	}
	if invocation := fm.Get("invocation"); invocation != nil {
		for _, key := range invocation.Keys() {
			if value := invocation.Get(key).String(); value != "" {
				labels["invocation."+key] = value
			}
		}
	}
	if len(labels) == 0 {
		return nil
	}
	return labels
}

// artifacts converts a produces/consumes block into canonical artifact contracts.
func artifacts(value *frontmatter.Value) []model.Artifact {
	var out []model.Artifact
	for _, item := range value.Items() {
		switch item.Kind {
		case frontmatter.KindScalar:
			if item.Str != "" {
				out = append(out, model.Artifact{Artifact: item.Str})
			}
		case frontmatter.KindMap:
			entry := model.Artifact{
				Artifact:    item.Get("artifact").String(),
				Path:        item.Get("path").String(),
				Description: item.Get("description").Text(),
			}
			if optional := item.Get("optional"); optional != nil {
				entry.Optional = optional.Bool()
			}
			if entry.Artifact == "" {
				entry.Artifact = item.Get("id").String()
			}
			if entry.Artifact != "" {
				out = append(out, entry)
			}
		}
	}
	return out
}

func readFrontmatter(path string) (*frontmatter.Value, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errs.Wrap(errs.CodeSourceUnavailable, err, "cannot read %s", path)
	}
	block, _, ok := frontmatter.Split(data)
	if !ok {
		return nil, errs.New(errs.CodeInvalidManifest, "%s has no frontmatter block", path)
	}
	value, err := frontmatter.Parse(block)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInvalidManifest, err, "cannot parse the frontmatter of %s", path)
	}
	return value, nil
}

func hasNativeManifest(dir string) bool {
	return fsutil.IsRegular(filepath.Join(dir, model.ManifestFilename))
}

// subdirs lists the immediate subdirectories of dir, tolerating a missing directory.
func subdirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errs.Wrap(errs.CodeSourceUnavailable, err, "cannot read %s", dir)
	}
	var out []string
	for _, entry := range entries {
		if entry.IsDir() && !skipDir(entry.Name()) {
			out = append(out, filepath.Join(dir, entry.Name()))
		}
	}
	sort.Strings(out)
	return out, nil
}

// markdownFiles lists the Markdown files directly inside dir, tolerating its absence.
func markdownFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errs.Wrap(errs.CodeSourceUnavailable, err, "cannot read %s", dir)
	}
	var out []string
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".md" {
			continue
		}
		out = append(out, entry.Name())
	}
	sort.Strings(out)
	return out, nil
}
