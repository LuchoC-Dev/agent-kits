---
name: build-progress
description: >
  Creates and maintains a docs/build.md progress tracker for any software project.
  Use this skill proactively after completing any implementation task, phase, or feature —
  even if the user doesn't explicitly ask. Also trigger when the user says things like
  "actualiza build.md", "update build.md", "mark X as done", "marcar X como completo",
  "actualiza el progreso", "update progress", "create build.md", or "crea el build.md".
  At project start, use it to scaffold the full build.md from the project description or codebase.
  After any task completes, use it to mark that task/stage as done and keep the file in sync.
---

## Purpose

Maintain a single source of truth for project build progress at `docs/build.md`.
The file has two parts: a summary table and a detailed checklist. Both must always be in sync.

## Two modes

### Mode A — Create (docs/build.md does not exist)

1. Check if `docs/build.md` exists. If not, this is Mode A.
2. Understand the project's planned phases and tasks from:
   - The current conversation (user description, prior work done)
   - Existing docs (README, CLAUDE.md, any planning docs)
   - The codebase structure (folders, files already present)
3. Infer which stages are already complete based on what exists in the codebase.
4. Create `docs/` if it doesn't exist.
5. Write `docs/build.md` using the format below.

### Mode B — Update (docs/build.md exists)

1. Read the current `docs/build.md`.
2. Determine what was just completed from the conversation context.
3. For each completed item:
   - In the table: change ⚪ → ✅, update % to 100%
   - In the detail section: change `[ ]` → `[x]` for each completed checkbox
4. If a phase has all stages complete, the phase row itself should show ✅ 100%.
5. Write the updated file.

## Format

```markdown
# Build Progress

## Estados

- ⚪ Pendiente
- 🔵 En progreso
- ✅ Completado

---

| Fase                     | N°  | Etapa                  | Estado | %    | Descripción                          |
| ------------------------ | --- | ---------------------- | :----: | :--: | ------------------------------------ |
| **1 — Initial Setup**    | 1.1 | Project scaffold       |   ✅   | 100% | Folder structure, config, entry point |
|                          | 1.2 | Dependencies           |   ✅   | 100% | Package manager, core libs            |
| **2 — Domain**           | 2.1 | Entities               |   ⚪   | 0%   | Core domain models                    |
|                          | 2.2 | Business rules         |   ⚪   | 0%   | Validations and invariants            |

---

## Detalle por etapa

---

### Etapa 1 — Initial Setup

#### 1.1 Project scaffold

- [x] Create folder structure
- [x] Configure build tool (pom.xml / package.json / pyproject.toml / etc.)
- [x] Create entry point

#### 1.2 Dependencies

- [x] Add core dependencies
- [x] Add dev/test dependencies

---

### Etapa 2 — Domain

#### 2.1 Entities

- [ ] Define core entities
- [ ] Add fields and getters

#### 2.2 Business rules

- [ ] Add validation logic
- [ ] Add invariants
```

## Rules

- Adapt language to the project: if the project uses Spanish (like the current one), keep headings in Spanish ("Etapa", "Fase", "Estados", etc.). If English, use English.
- The table and the detail section must always be in sync — if the table says ✅ 100%, all checkboxes in that section must be `[x]`.
- Phase rows (e.g., `**1 — Initial Setup**`) don't have their own N° or status — the status lives at the stage level (1.1, 1.2, etc.).
- Use `⚪` for not started, `🔵` for in progress (some checkboxes done but not all), `✅` for complete.
- % should be `0%`, `50%`, or `100%` — avoid intermediate values unless you have specific evidence.
- Keep descriptions short (≤ 8 words).
- Checkboxes in the detail section should be concrete and verifiable, not vague.
- Don't add phases or tasks that haven't been planned — only document what's real.
- After writing the file, confirm to the user with a one-line summary: "✅ build.md updated — etapa X.Y marcada como completa." or similar.
