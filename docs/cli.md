# Agent Kits CLI

Referencia de la CLI implementada en Go (D-016). Los contratos de esta página son los que
un agente puede consumir de forma estable.

## Instalación

```powershell
go build -o agent-kits.exe ./cmd/agent-kits
```

El binario es estático y no requiere runtime instalado. La única dependencia externa es el
`git` del sistema, y solo para sincronizar sources remotas.

## Configuración

| Ruta | Contenido |
|---|---|
| `~/.agent-kits/sources.json` | sources configuradas |
| `~/.agent-kits/cache/<source>/` | espejo local de las sources remotas |
| `<proyecto>/.agents/agent-kits.lock.json` | lockfile del proyecto |
| `<proyecto>/.agents/workspace.json` | descriptor compartido con `kits-init` |

`AGENT_KITS_HOME` reubica el directorio base completo. Es lo que usan los tests y lo que
permite tener perfiles aislados.

## Modelo

Un **recurso** es una `skill`, un `agent`, un `workflow` o un `kit` (D-020). Su identidad
es un ID canónico en una de dos formas (D-019):

```text
frontend-design               recurso del pool global
backend/feature-development   recurso cuya identidad pertenece a un kit
```

Una referencia corta se acepta cuando identifica un solo recurso. Si coincide con más de
uno, el comando falla con `ambiguous_id` y lista los candidatos: nunca elige por
precedencia.

## Comandos

### Catálogo

```text
agent-kits source list [--json]
agent-kits source add <name> <url> [--access public|private] [--trust trusted|review] [--ref <ref>]
agent-kits source remove <name>
agent-kits source sync [<name>...] [--json]
agent-kits search [<query>] [--type <type>] [--source <name>] [--limit <n>] [--json]
agent-kits info <id> [--json]
```

`source add` acepta una ruta local, una URL `file://` o un remoto Git. Solo los remotos
usan cache y requieren `sync`.

### Proyecto

```text
agent-kits plan <id>...   --project <path> [--runtime <name>] [--json]
agent-kits install <id>... --project <path> [--runtime <name>] [--yes] [--force] [--json]
agent-kits update [<id>...] --project <path> [--yes] [--force] [--json]
agent-kits remove <id>...  --project <path> [--yes] [--json]
agent-kits list            --project <path> [--json]
agent-kits doctor          --project <path> [--json]
agent-kits import         --project <path> [--yes] [--force] [--json]
agent-kits version [--json]
```

Los flags pueden ir antes o después de los operandos: `install frontend-design --project .`
y `install --project . frontend-design` son equivalentes.

`--runtime` acepta `auto` (default), `agents`, `claude-code` u `opencode`. En `auto` se
detecta el entorno por `CLAUDECODE` y `OPENCODE`, igual que el flujo heredado.

## Contrato para agentes

Con `--json` todo comando emite **un solo** documento con esta forma:

```json
{
  "ok": true,
  "command": "install",
  "data": { "...": "..." },
  "error": { "code": "...", "message": "...", "details": {} }
}
```

`error` solo aparece cuando `ok` es `false`. `doctor` es el único comando que emite
`ok: false` como resultado normal: un diagnóstico con problemas no es un fallo de
ejecución, y por eso el envelope llega completo aunque el exit code sea distinto de cero.

### Códigos de salida

| Código | Significado |
|---:|---|
| 0 | operación completada |
| 1 | fallo genérico (`not_found`, `runtime_unsupported`) |
| 2 | error de uso (`usage_error`) |
| 3 | integridad del registro (`ambiguous_id`, `registry_integrity_error`, `dependency_unresolved`, `version_conflict`, `visibility_violation`, `invalid_manifest`) |
| 4 | requiere decisión (`local_divergence`, `destination_conflict`, `confirmation_required`, `integrity_mismatch`, `workspace_invalid`) |
| 5 | source inaccesible (`source_unavailable`, `source_unknown`, `source_exists`) |
| 6 | seguridad (`unsafe_path`, `unsafe_content`, `untrusted_source`) |

`agent-kits version --json` lista el vocabulario completo de códigos, para que un agente
descubra qué debe saber manejar sin hardcodearlo.

### Confirmación

Todo plan que escribe requiere aprobación. En una sesión no interactiva —un agente, un
pipe o `--json`— la aprobación **solo** puede venir de `--yes`; sin él el comando devuelve
`confirmation_required` con exit 4 después de mostrar el plan. En una terminal se pregunta.

## Flujo típico

```powershell
agent-kits source add public https://github.com/example/agent-kits.git
agent-kits source sync public
agent-kits search frontend
agent-kits plan frontend-design --project .
agent-kits install frontend-design --project . --yes
agent-kits doctor --project .
```

## Política de conflictos

Antes de escribir se comparan tres checksums: el registrado en el lockfile, el del archivo
en disco y el del contenido nuevo (D-023).

| Estado | Acción |
|---|---|
| no existe en disco | `create` |
| idéntico y registrado | `unchanged` |
| idéntico y no registrado | `adopt` — solo se registra |
| registrado, intacto, hay contenido nuevo | `update` |
| registrado y modificado localmente | `divergent` — **bloquea** |
| existe sin registrar, con otro contenido | `divergent` — **bloquea** |

Un `divergent` devuelve `local_divergence` y no escribe nada. `--force` es la única forma
de sobrescribir y nunca es implícita.

Si dos recursos distintos apuntan al mismo archivo, el plan se bloquea con
`destination_conflict` (D-028).

## Idempotencia y atomicidad

Repetir una instalación sin cambios produce un plan vacío y no reescribe el lockfile.

Aplicar un plan es journalizado: cada archivo que se sobrescribe o borra se copia a un
directorio temporal fuera del proyecto, y cualquier fallo restaura el estado anterior. No
queda una instalación a medio aplicar.

## Compatibilidad con `kits-init`

El layout de destino es exactamente el que produce el flujo conversacional heredado:

```text
.agents/
├── agent-kits.lock.json
├── workspace.json
├── skills/<skill>/…
├── agents/<agent>.md
├── workflows/<workflow>.md
└── packs/<kit>/pack.md
```

`workspace.json` se lee y escribe en el esquema v2, conservando el orden documentado de
campos y **todo** campo que la CLI no administra. Un workspace v1 se actualiza a v2
ganando el campo `disciplines`.

`agent-kits import` adopta un workspace creado por `kits-init` y genera su lockfile. La
adopción es conservadora: un recurso se adopta solo si **todos** sus archivos coinciden
byte a byte con el catálogo. Los que difieren se informan y no se adoptan, porque
registrarlos haría que un `update` posterior sobrescribiera la diferencia en silencio.

## Seguridad

- La CLI **nunca** ejecuta contenido del catálogo.
- Toda escritura está confinada al proyecto destino; se rechazan rutas absolutas,
  traversal, referencias de volumen, symlinks y nombres de dispositivo reservados.
- Límite por archivo de 2 MiB y de 512 archivos por recurso.
- Se detectan credenciales antes de escribir: un match de alta confianza bloquea.
- `git` se invoca con una lista blanca de subcomandos de solo lectura. `push`, `commit`,
  `remote` y `tag` no están en ella, así que la garantía "sin escrituras remotas" es
  verificable por inspección de `internal/git/git.go`.

## Estructura del código

```text
cmd/agent-kits/        punto de entrada
internal/
├── cli/               parseo de comandos y presentación
├── model/             vocabulario canónico
├── errs/              códigos de error y exit codes
├── semver/            subconjunto de SemVer aprobado
├── frontmatter/       parser del YAML que usa el catálogo heredado
├── source/            sources y cache
├── git/               lista blanca de subcomandos
├── catalog/           carga nativa, adaptador legacy y unicidad
├── resolve/           resolución de dependencias
├── plan/              planificación determinística
├── install/           aplicación, lockfile, doctor e import
├── adapter/           destinos por runtime
├── workspace/         lockfile y workspace.json
├── security/          rutas, límites y secretos
├── fsutil/            checksums y escritura atómica
└── internaltest/      fixtures de test
```

Pruebas: `go test ./...`.
