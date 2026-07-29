// Package workspace reads and writes the two bookkeeping files a project carries: the
// Agent Kits lockfile and the inherited workspace.json.
//
// workspace.json is not owned by Agent Kits — the conversational kits-init flow writes it
// too — so this package preserves its documented field order and every field it does not
// manage (D-022).
package workspace

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/LuchoC-Dev/agent-kits/internal/adapter"
	"github.com/LuchoC-Dev/agent-kits/internal/errs"
	"github.com/LuchoC-Dev/agent-kits/internal/fsutil"
	"github.com/LuchoC-Dev/agent-kits/internal/model"
	"github.com/LuchoC-Dev/agent-kits/internal/security"
)

// SchemaVersion is the workspace.json schema this build reads and writes.
const SchemaVersion = 2

// SystemVersion is recorded in workspace.json as the version of the system that wrote it.
const SystemVersion = "0.1.0"

// LoadLock reads the lockfile, returning an empty lock when the project has none.
func LoadLock(project string, a adapter.Adapter) (*model.Lock, error) {
	path, err := security.Contain(project, a.LockPath())
	if err != nil {
		return nil, err
	}
	if !fsutil.Exists(path) {
		return model.NewLock(a.Name()), nil
	}
	var lock model.Lock
	if err := fsutil.ReadJSON(path, &lock); err != nil {
		return nil, errs.Wrap(errs.CodeWorkspaceInvalid, err, "cannot read %s", a.LockPath())
	}
	if err := lock.Validate(); err != nil {
		return nil, err
	}
	return &lock, nil
}

// LockBytes renders a lockfile exactly as it would be written.
func LockBytes(lock *model.Lock) ([]byte, error) {
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, err, "cannot encode the lockfile")
	}
	return append(data, '\n'), nil
}

// Entry is an installed resource recorded in workspace.json.
type Entry struct {
	ID          string `json:"id"`
	Class       int    `json:"class,omitempty"`
	Source      string `json:"source"`
	InstalledAt string `json:"installed_at"`
}

// Pack is the composition record of workspace.json.
type Pack struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	InstalledAt string `json:"installed_at"`
}

// Stack is the detected project stack.
type Stack struct {
	Detected   []string `json:"detected"`
	Source     string   `json:"source"`
	Confidence string   `json:"confidence"`
}

// Flags are the workspace lifecycle markers.
type Flags struct {
	Initialized bool    `json:"initialized"`
	RepairedAt  *string `json:"repaired_at"`
	UpgradedAt  *string `json:"upgraded_at"`
}

// Descriptor is workspace.json.
type Descriptor struct {
	SchemaVersion int      `json:"$schema_version"`
	ID            string   `json:"id"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
	SystemVersion string   `json:"system_version"`
	Runtime       string   `json:"runtime"`
	Pack          *Pack    `json:"pack"`
	Stack         *Stack   `json:"stack,omitempty"`
	Skills        []Entry  `json:"skills"`
	Agents        []Entry  `json:"agents"`
	Disciplines   []string `json:"disciplines"`
	Flags         Flags    `json:"flags"`
	Structure     []string `json:"structure"`

	// extra keeps fields written by another tool so a round trip loses nothing.
	extra map[string]json.RawMessage
}

// fieldOrder is the documented key order of workspace-schema.md. Marshalling follows it
// so a workspace edited by the CLI stays diff-friendly against one written by kits-init.
var fieldOrder = []string{
	"$schema_version", "id", "created_at", "updated_at", "system_version", "runtime",
	"pack", "stack", "skills", "agents", "disciplines", "flags", "structure",
}

// managedFields are the keys Descriptor itself encodes; anything else is passed through.
var managedFields = func() map[string]bool {
	out := map[string]bool{}
	for _, name := range fieldOrder {
		out[name] = true
	}
	return out
}()

// UnmarshalJSON decodes the descriptor while retaining unknown fields.
func (d *Descriptor) UnmarshalJSON(data []byte) error {
	type alias Descriptor
	var base alias
	if err := json.Unmarshal(data, &base); err != nil {
		return err
	}
	*d = Descriptor(base)

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	d.extra = map[string]json.RawMessage{}
	for key, value := range raw {
		if !managedFields[key] {
			d.extra[key] = value
		}
	}
	return nil
}

// MarshalJSON encodes the descriptor in documented field order, followed by any
// unmanaged fields sorted by name.
func (d Descriptor) MarshalJSON() ([]byte, error) {
	type alias Descriptor
	encoded, err := json.Marshal(alias(d))
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.WriteByte('{')
	first := true
	write := func(key string, value json.RawMessage) {
		if !first {
			buf.WriteByte(',')
		}
		first = false
		keyJSON, _ := json.Marshal(key)
		buf.Write(keyJSON)
		buf.WriteByte(':')
		buf.Write(value)
	}
	for _, key := range fieldOrder {
		if value, ok := fields[key]; ok {
			write(key, value)
		}
	}
	extraKeys := make([]string, 0, len(d.extra))
	for key := range d.extra {
		extraKeys = append(extraKeys, key)
	}
	sort.Strings(extraKeys)
	for _, key := range extraKeys {
		write(key, d.extra[key])
	}
	buf.WriteByte('}')

	// Deliberately compact: encoding/json compacts whatever a custom marshaler returns,
	// so indentation is applied once, at the point of writing, by DescriptorBytes.
	return buf.Bytes(), nil
}

// LoadDescriptor reads workspace.json, or reports that the project has none.
func LoadDescriptor(project string, a adapter.Adapter) (*Descriptor, bool, error) {
	path, err := security.Contain(project, a.WorkspacePath())
	if err != nil {
		return nil, false, err
	}
	if !fsutil.Exists(path) {
		return nil, false, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, errs.Wrap(errs.CodeWorkspaceInvalid, err, "cannot read %s", a.WorkspacePath())
	}
	var descriptor Descriptor
	if err := json.Unmarshal(data, &descriptor); err != nil {
		return nil, false, errs.Wrap(errs.CodeWorkspaceInvalid, err,
			"cannot parse %s", a.WorkspacePath())
	}
	if descriptor.SchemaVersion != SchemaVersion && descriptor.SchemaVersion != 1 {
		return nil, false, errs.New(errs.CodeWorkspaceInvalid,
			"unsupported workspace.json $schema_version %d (expected 1 or %d)",
			descriptor.SchemaVersion, SchemaVersion)
	}
	return &descriptor, true, nil
}

// Sync rebuilds the descriptor from a lockfile, preserving everything Agent Kits does not
// own. A v1 workspace is upgraded in place by gaining the disciplines field.
func Sync(
	existing *Descriptor, lock *model.Lock, runtime string, resources map[model.ID]*model.Resource,
	now time.Time,
) (*Descriptor, error) {
	stamp := now.UTC().Format(time.RFC3339)

	descriptor := &Descriptor{}
	if existing != nil {
		*descriptor = *existing
	} else {
		id, err := uuidV4()
		if err != nil {
			return nil, err
		}
		descriptor.ID = id
		descriptor.CreatedAt = stamp
		descriptor.Stack = &Stack{Detected: []string{}, Source: "none", Confidence: "low"}
	}
	descriptor.SchemaVersion = SchemaVersion
	descriptor.SystemVersion = SystemVersion
	descriptor.UpdatedAt = stamp
	descriptor.Runtime = runtime
	descriptor.Flags.Initialized = true

	var (
		skills      []Entry
		agents      []Entry
		disciplines []string
		kits        []string
		hasWorkflow bool
	)
	for _, record := range lock.Resources {
		entry := Entry{
			ID:          string(record.ID),
			Source:      record.Source,
			InstalledAt: stamp,
		}
		if previous, ok := findEntry(existing, record.ID); ok {
			entry.InstalledAt = previous.InstalledAt
			entry.Class = previous.Class
		}
		switch record.Type {
		case model.TypeSkill:
			skills = append(skills, entry)
			if res, ok := resources[record.ID]; ok && res.Traits["discipline"] {
				disciplines = append(disciplines, string(record.ID))
			}
		case model.TypeAgent:
			if entry.Class == 0 {
				entry.Class = agentClass(record.ID, resources[record.ID])
			}
			agents = append(agents, entry)
		case model.TypeWorkflow:
			hasWorkflow = true
		case model.TypeKit:
			kits = append(kits, string(record.ID))
		}
	}
	sortEntries(skills)
	sortEntries(agents)
	sort.Strings(disciplines)
	sort.Strings(kits)

	descriptor.Skills = orEmptyEntries(skills)
	descriptor.Agents = orEmptyEntries(agents)
	descriptor.Disciplines = orEmpty(disciplines)
	descriptor.Pack = packRecord(existing, kits, stamp)
	descriptor.Structure = structure(len(skills) > 0, len(agents) > 0, hasWorkflow, len(kits) > 0)
	return descriptor, nil
}

// packRecord keeps the single-composition field of the inherited schema meaningful: one
// installed kit names itself, several become "custom", none keeps whatever was there.
func packRecord(existing *Descriptor, kits []string, stamp string) *Pack {
	switch len(kits) {
	case 0:
		if existing != nil && existing.Pack != nil {
			return existing.Pack
		}
		return &Pack{Name: "custom", InstalledAt: stamp}
	case 1:
		installedAt := stamp
		if existing != nil && existing.Pack != nil && existing.Pack.Name == kits[0] {
			installedAt = existing.Pack.InstalledAt
		}
		return &Pack{Name: kits[0], Source: "packs/" + kits[0], InstalledAt: installedAt}
	}
	return &Pack{Name: "custom", InstalledAt: stamp}
}

func structure(skills, agents, workflows, packs bool) []string {
	var out []string
	for _, candidate := range []struct {
		name    string
		present bool
	}{
		{"agents", agents}, {"packs", packs}, {"skills", skills}, {"workflows", workflows},
	} {
		if candidate.present {
			out = append(out, candidate.name)
		}
	}
	return orEmpty(out)
}

// agentClass reproduces the inherited class taxonomy: an agent that orchestrates a kit's
// workflow is class 1, a reusable agent from the shared pool is class 2.
//
// A declared class always wins. Otherwise ownership decides, which is derivable from the
// id alone — so the classification still works when the catalog is unavailable.
func agentClass(id model.ID, res *model.Resource) int {
	if res != nil {
		if raw, ok := res.Labels["class"]; ok {
			var class int
			if _, err := fmt.Sscanf(raw, "%d", &class); err == nil && class > 0 {
				return class
			}
		}
	}
	if id.Qualified() {
		return 1
	}
	return 2
}

func findEntry(existing *Descriptor, id model.ID) (Entry, bool) {
	if existing == nil {
		return Entry{}, false
	}
	for _, group := range [][]Entry{existing.Skills, existing.Agents} {
		for _, entry := range group {
			if entry.ID == string(id) {
				return entry, true
			}
		}
	}
	return Entry{}, false
}

func sortEntries(entries []Entry) {
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
}

func orEmpty(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

func orEmptyEntries(in []Entry) []Entry {
	if in == nil {
		return []Entry{}
	}
	return in
}

// DescriptorBytes renders workspace.json as it would be written, indented for review.
func DescriptorBytes(descriptor *Descriptor) ([]byte, error) {
	data, err := json.Marshal(descriptor)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, err, "cannot encode workspace.json")
	}
	var indented bytes.Buffer
	if err := json.Indent(&indented, data, "", "  "); err != nil {
		return nil, errs.Wrap(errs.CodeInternal, err, "cannot format workspace.json")
	}
	return append(indented.Bytes(), '\n'), nil
}

// uuidV4 generates a random identifier for a new workspace.
func uuidV4() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", errs.Wrap(errs.CodeInternal, err, "cannot generate a workspace id")
	}
	buf[6] = (buf[6] & 0x0f) | 0x40 // version 4
	buf[8] = (buf[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16]), nil
}
