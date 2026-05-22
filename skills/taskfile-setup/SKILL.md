---
name: taskfile-setup
description: >
  Creates or updates a Taskfile.yml and a docs/tasks.md for any backend project that uses
  Docker Compose for infrastructure and any build tool (Maven, Gradle, npm, pnpm, Poetry, Go, etc.).
  Use this skill whenever the user wants to set up task automation, create a Taskfile, configure
  project tasks, or when starting any new project that needs a standard set of commands.
  Also trigger when the user says "crear taskfile", "setup taskfile", "configurar tasks",
  "crear tasks", "necesito un taskfile", or asks how to run the project with a task runner.
  When in doubt, offer to set this up — projects without a Taskfile benefit from having one.
---

# taskfile-setup

Creates a `Taskfile.yml` with standard tasks adapted to the project's stack using **includes**,
then writes `docs/tasks.md` documenting every task.

## Step 1 — Detect project conventions

Before writing anything, read the project to detect:

**Build tool** — check for these files in order:
- `pom.xml` → Maven (`mvn`)
- `build.gradle` / `build.gradle.kts` → Gradle (`./gradlew`)
- `package.json` → Node.js (`npm` or `pnpm` or `yarn` — check `packageManager` field or lockfiles)
- `pyproject.toml` / `requirements.txt` → Python (`poetry run` / `pip` / `uv`)
- `go.mod` → Go (`go`)
- `Cargo.toml` → Rust (`cargo`)
- anything else → ask the user

**Dev/test infra** — check for:
- `docker-compose.yml` or `compose.yml` → include infra tasks
- `docker-compose.test.yml` → include separate test infra tasks
- `.env.dev` vs `.env` → use the right filename in `--env-file`

**Test layout** — check the test directory structure to infer test commands:
- Maven/Gradle Java: look for `unit/`, `integration/`, `api/` subdirectories under `src/test/`
- Node: check for `jest`, `vitest`, `mocha` in `package.json`
- Python: check for `pytest`
- Go: `go test ./...`
- Rust: `cargo test`

**Existing Taskfile** — if `Taskfile.yml` already exists, read it first and merge rather than overwrite. Preserve any custom tasks the user already has.

## Step 2 — Build the task map

Based on detected conventions, decide which groups to include. The standard groups are:

### Infrastructure (include if Docker Compose is present)

Dev and test infra use nested includes — `taskfiles/infra.yml` includes `taskfiles/infra-test.yml` under the `test` namespace. This produces `task infra:test:up`, `task infra:test:down`, etc.

| Task | What it does |
|------|-------------|
| `infra:up` | Start dev database/services |
| `infra:down` | Stop dev database/services |
| `infra:clean-db` | Wipe dev DB volume and recreate (fresh migrations) |
| `infra:test:up` | Start test database/services (if `docker-compose.test.yml` exists) |
| `infra:test:down` | Stop test services |
| `infra:test:clean-db` | Wipe test DB volume and recreate |

### Application (always include — tasks defined directly in root Taskfile.yml, not in an include)

| Task | What it does |
|------|-------------|
| `build` | Compile / package without running tests |
| `dev` | Start the app in development mode (dep: infra:up if applicable) |
| `run` | Run the compiled binary (if applicable) |

Common build commands by stack:
- Maven: `mvn package -DskipTests` / `mvn spring-boot:run -Dspring-boot.run.profiles=dev`
- Gradle: `./gradlew build -x test` / `./gradlew bootRun`
- Node: see Windows note below
- Python: `poetry build` / `poetry run python -m app`
- Go: `go build -o bin/...` / `go run ./cmd/...` / `air` for live reload
- Rust: `cargo build --release` / `cargo run`

> ⚠️ **Windows + Node.js — Ctrl+C terminal bug**: On Windows, `npm` is a `.cmd` batch file. When a Taskfile task runs `npm run dev` (or any `npm run <script>`) and the user presses Ctrl+C, Windows shows a "¿Desea terminar el trabajo por lotes?" prompt and leaves the terminal in a broken state.
>
> **Fix**: invoke the Node.js entry point directly instead of going through `npm run`:
>
> ```yaml
> dev:
>   cmds:
>     - node node_modules/next/dist/bin/next dev --port 5000   # Next.js
> ```
>
> Common entry points by framework:
> | Framework | Command |
> |-----------|---------|
> | Next.js | `node node_modules/next/dist/bin/next dev` |
> | NestJS | `node node_modules/@nestjs/cli/bin/nest.js start --watch` |
> | ts-node | `node node_modules/ts-node/dist/bin.js src/main.ts` |
> | Vite | `node node_modules/vite/bin/vite.js` |
> | Vitest | `node node_modules/vitest/vitest.mjs` |
>
> To find the entry point for any package: check what `npm run <script>` executes in `package.json`, then find the corresponding `.js` file under `node_modules/<package>/`.
>
> This applies **only** to long-running dev/watch tasks. Short-lived commands (`npm run build`, `npm run lint`) are fine with `npm run` since Ctrl+C isn't an issue.

### Tests (always include)

Include only the test tasks that match the detected test layout. For Java projects with layered tests:

```yaml
# taskfiles/test.yml
tasks:
  unit:
    desc: ...
    cmds:
      - mvn test -Dtest="unit/**/*Test"

  integration:
    desc: ...
    deps:
      - task: :infra:test:up
    cmds:
      - mvn test -Dtest="integration/**/*Test" -Dsurefire.failIfNoSpecifiedTests=false
    finally:
      - task: :infra:test:down

  api:
    desc: ...
    deps:
      - task: :infra:test:up
    cmds:
      - mvn test -Dtest="api/**/*Test"
    finally:
      - task: :infra:test:down

  all:
    desc: ...
    deps:
      - task: :infra:test:up
    cmds:
      - mvn test
    finally:
      - task: :infra:test:down
```

For simpler stacks (Node, Go, Python) include `test:unit`, `test:api`, and `test:all` as needed.

> ⚠️ **Cross-namespace deps in included files**: When a task inside an included file (e.g. `taskfiles/test.yml`) needs to depend on a task in another namespace (e.g. `infra:test:up`), the reference MUST be prefixed with `:` to resolve from the root namespace. Without it, Taskfile prepends the current namespace, producing `test:infra:test:up` (broken).
>
> ```yaml
> # ✅ Correct — resolves from root
> deps:
>   - task: :infra:test:up
> finally:
>   - task: :infra:test:down
>
> # ❌ Wrong — resolves as test:infra:test:up
> deps:
>   - task: infra:test:up
> ```

## Step 3 — Write the files using includes

**Always use the includes pattern.** The root `Taskfile.yml` contains only the `includes` block.
Each group lives in its own file under `taskfiles/`.

### Root Taskfile.yml

Application tasks (`build`, `dev`, `run`) live directly in the root `Taskfile.yml` — no namespace, no include. Only `test` and `infra` use includes.

```yaml
version: "3"

includes:
  test: ./taskfiles/test.yml
  # infra: ./taskfiles/infra.yml  ← only if Docker Compose is present

tasks:
  build:
    desc: ...
    cmds:
      - ...

  dev:
    desc: ...
    cmds:
      - ...

  run:  # if applicable (compiled languages)
    desc: ...
    cmds:
      - ...
```

### taskfiles/infra.yml (only if Docker Compose is present)

If `docker-compose.test.yml` exists, `infra.yml` includes `infra-test.yml` under the `test` namespace — producing `task infra:test:up`, `task infra:test:clean-db`, etc.

```yaml
version: "3"

includes:
  test: ./infra-test.yml  # only if docker-compose.test.yml exists

tasks:
  up:
    desc: ...
    cmds:
      - docker compose up -d

  down:
    desc: ...
    cmds:
      - docker compose down

  clean-db:
    desc: ...
    cmds:
      - docker compose down -v
      - docker compose up -d
```

### taskfiles/infra-test.yml (only if docker-compose.test.yml exists)

```yaml
version: "3"

tasks:
  up:
    desc: ...
    cmds:
      - docker compose -f docker-compose.test.yml up -d

  down:
    desc: ...
    cmds:
      - docker compose -f docker-compose.test.yml down

  clean-db:
    desc: ...
    cmds:
      - docker compose -f docker-compose.test.yml down -v
      - docker compose -f docker-compose.test.yml up -d
```

### taskfiles/test.yml

```yaml
version: "3"

tasks:
  unit:
    desc: ...
    cmds:
      - ...

  all:
    desc: ...
    cmds:
      - ...
```

Every task must have a `desc:` field. Use the project's natural language (Spanish if the project uses Spanish, English otherwise).

If a test task depends on infra, use `deps:` and `finally:` so infra always goes down even on failure.

## Step 4 — Write docs/tasks.md

Create `docs/` if it doesn't exist. Document every task using the correct syntax (`task build`, `task test:unit`, `task infra:up`, `task infra:test:up`, etc.), grouped by section. Include a notes section explaining:
- Which tasks require Docker
- Which tasks auto-start infra
- That `infra:clean-db` wipes the volume (data loss warning)
- Any tools that need to be installed (e.g. `air` for Go live reload)

## Step 5 — Confirm

One-line summary of what was created or updated:
```
✅ Taskfile.yml creado con includes (app, test[, infra]). docs/tasks.md generado.
```
