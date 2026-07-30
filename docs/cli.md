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
| `<proyecto>/.agents/agent-kits.lock.json` | **único** estado del proyecto (lockfile v2) |
| `<proyecto>/.agents/workspace.json.migrated.bak` | copia intacta que deja `migrate`, propiedad del usuario |

`AGENT_KITS_HOME` reubica el directorio base completo. Es lo que usan los tests y lo que
permite tener perfiles aislados.

## Modelo

Un **recurso** es una `skill`, un `agent`, un `workflow` o un `kit` (D-020).

Su **identidad** es un UUID asignado una vez y para siempre (D-035). Su **nombre** es cómo
lo pedís y dónde se instala, y puede cambiar (D-036). Pertenecer a un kit es una relación,
no parte de la identidad: mover un recurso de un kit a otro no cambia quién es.

| | Qué es | ¿Cambia? |
|---|---|---|
| `id` | la identidad | nunca |
| `name` | el nombre de instalación | sí, es un rename explícito |
| kit | una relación de composición | libremente |

Por eso un rename no rompe nada: el lockfile registra la identidad, así que sigue
apuntando al mismo recurso aunque su nombre cambie.

### Cómo se referencia un recurso

```text
frontend-design          el nombre, cuando una sola source lo ofrece
acme:frontend-design     calificado por source, cuando varias lo ofrecen
9f2c1b7a-…               el UUID, que siempre funciona
```

El nombre es único **dentro de una source**. Entre sources puede repetirse: dos
organizaciones pueden publicar cada una su `frontend-design` y son recursos distintos. Si
una referencia corta coincide con más de uno, el comando falla con `ambiguous_id` y lista
los candidatos calificados. Nunca elige por precedencia.

Dos recursos con el mismo **nombre de instalación** no pueden coexistir en un proyecto:
dos archivos no ocupan la misma ruta. Eso es `destination_conflict` (D-028).

## Layout de una source

Cada recurso vive en su propio directorio y se describe con un `agent-kit.json` (D-017).
Es el **único** layout reconocido: el adaptador que sintetizaba manifests desde el
Markdown heredado se retiró cuando todo el catálogo pasó a ser nativo (D-034).

```text
skills/<id>/agent-kit.json          + SKILL.md y sus archivos
agents/<id>/agent-kit.json          + <archivo>.md
packs/<kit>/agent-kit.json          + pack.md
packs/<kit>/agents/<id>/…           agentes que el kit posee
packs/<kit>/workflows/<id>/…        workflows que el kit posee
```

```json
{
  "schema_version": 1,
  "id": "7b6b5f3c-1e2d-4a90-8c31-5f0be2a7d914",
  "name": "backend-feature-development",
  "type": "workflow",
  "version": "1.0.0",
  "description": "…",
  "dependencies": [
    { "id": "5c954f33-73e3-4c05-b1b0-9d72baff0182", "name": "tdd" }
  ],
  "files": ["backend-feature-development.md"]
}
```

`files` es relativo al directorio del manifest y decide **la ruta instalada**, no la
ubicación en la source: mover un recurso de directorio no cambia dónde aterriza. Si se
omite, se descubren todos los archivos del directorio.

Una dependencia apunta a la **identidad**, no al nombre, así que sobrevive a un rename del
recurso del que depende. El `name` que la acompaña es informativo —una lista de UUIDs es
ilegible— y se verifica contra el recurso resuelto: si no coincide, se informa que el
manifest quedó desactualizado. También se admite la forma corta `"<uuid>"`.

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

### Sources emparentadas por publicación

`--publishes <source>` declara que una source es **el espejo publicado** de otra: el
público de Agent Kits es el subconjunto publicado del privado (D-038).

```powershell
agent-kits source add private https://github.com/LuchoC-Dev/repository-private.git --access private
agent-kits source add public  https://github.com/LuchoC-Dev/repository.git --publishes private
```

Un recurso publicado existe en las dos y comparte identidad. Entre sources emparentadas eso
es lo esperado, no un duplicado: gana el **origen privado**, que por construcción está igual
o más adelantado, y el recurso aparece una sola vez al buscar o resolver. Entre sources **no**
emparentadas, una identidad repetida sigue siendo `registry_integrity_error`.

La precedencia no se infiere nunca: existe solo porque alguien la declaró.

### Proyecto

```text
agent-kits plan <id>...   --project <path> [--runtime <name>] [--json]
agent-kits install <id>... --project <path> [--runtime <name>] [--yes] [--force] [--json]
agent-kits update [<id>...] --project <path> [--yes] [--force] [--json]
agent-kits remove <id>...  --project <path> [--yes] [--json]
agent-kits list            --project <path> [--json]
agent-kits doctor          --project <path> [--json]
agent-kits migrate         --project <path> [--yes] [--json]
agent-kits version [--json]
```

`agent-kits import` sigue existiendo como **alias deprecado** de `migrate`: comparte su
implementación, avisa por `stderr` y añade un campo `deprecated` al JSON. Se elimina en un
cambio posterior aprobado (D-031).

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

## Estado del proyecto

El lockfile v2 concentra **todo** el estado que Agent Kits administra (D-030): identidad
del proyecto, stack, disciplinas, recursos instalados con sus archivos y checksums, y el
registro de la migración si la hubo.

```text
.agents/
├── agent-kits.lock.json     ← único estado
├── skills/<skill>/…
├── agents/<agent>.md
├── workflows/<workflow>.md
└── packs/<kit>/pack.md
```

`project.id` y `project.created_at` se asignan en la primera escritura y **no cambian
nunca** después: instalar, actualizar, eliminar o migrar los preservan.

Un lockfile v1 sigue siendo legible: se actualiza en memoria al leerlo, así que toda
escritura produce v2. Un `schema_version` desconocido falla con `workspace_invalid`.

## Migración desde `kits-init`

`workspace.json` ya no se escribe. Un proyecto creado por el flujo conversacional heredado
se adopta una sola vez:

```powershell
agent-kits migrate --project . --json   # calcula el plan, no escribe
agent-kits migrate --project . --yes    # lo aplica
```

La migración:

- genera y valida el lockfile v2 **antes** de retirar `workspace.json`;
- conserva identidad, stack, disciplinas y fechas de instalación heredadas;
- guarda en `migration` los datos históricos (`pack`, `flags`, `structure`,
  `system_version`) y **todo** campo desconocido, tal cual estaba;
- escribe `.agents/workspace.json.migrated.bak` con los bytes originales, no con una
  reserialización;
- aplica lockfile, backup y retirada como una sola operación journalizada: si algo falla,
  el proyecto queda exactamente como estaba.

Es idempotente: repetirla no cambia ningún archivo. Si se interrumpió después del backup,
la siguiente ejecución solo completa la retirada.

La adopción de recursos es conservadora y **fail-closed**:

| Situación | Resultado |
|---|---|
| todos los archivos coinciden con el catálogo | se adopta |
| un archivo administrado difiere | **bloquea** con `local_divergence` |
| un recurso declarado en `workspace.json` no se identifica | **bloquea** |
| un archivo suelto en `.agents/` no se identifica | se informa y se deja como está |
| ya existe un backup con contenido distinto | **bloquea** con `integrity_mismatch` |

`migrate` no acepta `--force`: una migración de estado nunca descarta datos para continuar.

Mientras exista un `workspace.json` sin migrar, `install`, `update` y `remove` se detienen
con `workspace_invalid` y no escriben nada — un proyecto nunca se opera con dos fuentes de
verdad. `plan`, `list`, `info` y `doctor` siguen funcionando; `doctor` informa que la
migración es necesaria.

El backup es propiedad del usuario: Agent Kits no lo borra nunca.

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
├── source/            sources y cache
├── git/               lista blanca de subcomandos
├── catalog/           carga de manifests nativos y unicidad
├── resolve/           resolución de dependencias
├── plan/              planificación determinística
├── install/           aplicación, lockfile y doctor
├── migrate/           transición a lockfile v2 (temporal)
├── journal/           backup y rollback de una operación
├── adapter/           destinos por runtime
├── workspace/         lockfile (y lector heredado en legacy.go)
├── security/          rutas, límites y secretos
├── fsutil/            checksums y escritura atómica
└── internaltest/      fixtures de test
```

Pruebas: `go test ./...`.
