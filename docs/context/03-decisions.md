# Registro de decisiones — Agent Kits Next

**Estado:** Active
**Fecha inicial:** 2026-07-29

Cada decisión tiene un identificador estable. Las decisiones reemplazadas se conservan
marcadas como `Superseded`.

## D-001 — Evolucionar mediante un fork

**Estado:** Accepted

La nueva versión se implementará en un fork independiente. El repositorio original se
mantiene como referencia y upstream.

**Razón:** permitir cambios profundos, colaboración y experimentación sin alterar la
aplicación existente.

## D-002 — El repositorio es la fuente de verdad

**Estado:** Accepted

Skills, agents, workflows y kits viven en repositorios Git versionados. Los runtimes son
destinos de instalación, no el almacenamiento canónico.

## D-003 — Separar creación y obtención

**Estado:** Accepted

Agent Kits solo obtiene e instala. La creación y validación autoral pertenecen a otra
superficie.

**Consecuencia:** Agent Kits no incluirá `create`, `import` o `publish` como operaciones
de escritura sobre el catálogo.

## D-004 — Prohibir publicación desde Agent Kits

**Estado:** Accepted

La CLI no hace commits, push, ramas ni PR en las sources. La incorporación al catálogo
usa procesos Git separados y controlados.

## D-005 — Separar físicamente lo público y lo privado

**Estado:** Accepted

Los recursos públicos y privados viven en repositorios con permisos distintos.

**Razón:** un campo `visibility` dentro de un repositorio público no proporciona
privacidad.

## D-006 — IDs globalmente únicos

**Estado:** Accepted

Un ID solo puede corresponder a una capacidad activa entre todas las fuentes
administradas.

**Consecuencia:** el origen no forma parte del ID. Los duplicados son errores de
integridad, nunca candidatos a precedencia.

## D-007 — Una visibilidad activa por recurso

**Estado:** Accepted

Un recurso privado que pasa a público se traslada conservando su identidad. No se crean
variantes simultáneas con el mismo ID.

## D-008 — Convención semántica de nombres

**Estado:** Accepted

- skill → disciplina u objeto (`frontend-design`);
- agent → persona o rol (`frontend-designer`);
- workflow → proceso (`design-frontend-interface`);
- kit → colección (`frontend-design-kit`);
- tool → instrumento (`figma-inspector`);
- template → artefacto (`design-brief-template`).

La convención se documenta y puede generar advertencias, pero no es una restricción
estructural rígida.

## D-009 — La CLI está diseñada para agentes

**Estado:** Accepted

Las operaciones deben ser no interactivas cuando se proporcionan flags suficientes,
idempotentes y capaces de devolver JSON estructurado.

## D-010 — Planificar antes de escribir

**Estado:** Accepted

Toda instalación o actualización debe ofrecer un plan o dry-run. Los cambios
significativos requieren confirmación explícita o `--yes`.

## D-011 — No sobrescribir silenciosamente

**Estado:** Accepted

Si un archivo administrado cambió localmente, Agent Kits no lo reemplaza por
precedencia. La política de resolución final sigue abierta.

## D-012 — Orquestación liviana como kit

**Estado:** Accepted

Un sistema compuesto por un agente orquestador, agentes delegados, memoria y guías es un
kit instalable. El agente anfitrión es el runtime.

**No confundir con:** un servicio persistente de orquestación como Event Agent Manager.

## D-013 — Mantener la autoría fuera del MVP

**Estado:** Accepted

La herramienta de autoría puede diseñarse posteriormente. Su nombre provisional no
forma parte de la API ni de la marca de Agent Kits.

## D-014 — Mantener upstream sin push

**Estado:** Accepted

El remoto `upstream` del fork local tiene `pushurl=DISABLED`. Un `origin` colaborativo se
añadirá solo después de acordar propietario y nombre.

## D-015 — Especificar antes de implementar

**Estado:** Accepted

La nueva versión no comenzará con código. Primero se audita el legado, se revisa la
especificación y se aprueban contratos y límites del MVP.

## D-016 — Stack: Go sin dependencias externas

**Estado:** Accepted
**Fecha:** 2026-07-29

La CLI se implementa en Go, usando exclusivamente la librería estándar.

**Razón:** `go build` produce binarios estáticos para Windows, macOS y Linux sin runtime
instalado; la stdlib cubre JSON, SHA-256, filesystem y ejecución de procesos; `go test`
no requiere configuración. Rust era la alternativa considerada, pero habría exigido
crates externos (`serde`, `clap`, `anyhow`) para llegar al mismo punto.

**Consecuencia:** `go.mod` no declara `require`. Añadir la primera dependencia externa
requiere una decisión nueva.

## D-017 — Manifests en JSON

**Estado:** Accepted

Cada recurso canónico se describe con un `agent-kit.json` que incluye `schema_version`.

**Razón:** JSON se parsea con la stdlib, es inequívoco y es el formato que un agente
consume sin ambigüedad. No se admite YAML en los contratos nuevos.

## D-018 — Lockfile en JSON dentro del workspace

**Estado:** Accepted

El lockfile es `.agents/agent-kits.lock.json`, con `schema_version`, y registra por
recurso: `id`, `type`, `source`, `version`, `commit`, `checksum`, `requested` y la lista
de archivos instalados con su checksum individual.

**Razón:** vive junto a lo que describe, se versiona con el proyecto y es legible por el
mismo parser que los manifests.

## D-019 — Gramática de IDs canónicos con propiedad de kit

**Estado:** Accepted

Un ID canónico tiene dos formas:

- `<name>` para recursos del pool global;
- `<kit>/<name>` para recursos cuya identidad pertenece a un kit.

La unicidad se exige sobre el ID canónico completo. Una referencia corta que coincida con
más de un ID canónico devuelve `ambiguous_id` y detiene la operación.

**Razón:** el catálogo heredado contiene `feature-development` con contenido distinto en
los packs `backend` y `frontend`, y workflows que colisionan con nombres de pack
(`backend-design`, `fullstack-design`). La propiedad de kit resuelve esas colisiones sin
renombrar contenido heredado ni elegir un ganador por precedencia.

**No viola D-006:** el prefijo es el kit propietario, no la source. Mover un kit entre
una source privada y una pública conserva los IDs de sus componentes.

## D-020 — El MVP soporta los cuatro tipos desde el arranque

**Estado:** Accepted

`skill`, `agent`, `workflow` y `kit` son ciudadanos de primera clase en resolución, plan,
instalación, lockfile y doctor. `tool` y `template` siguen fuera del alcance.

## D-021 — Adaptadores iniciales

**Estado:** Accepted

Se implementan tres adaptadores: `agents` (genérico), `claude-code` y `opencode`, con
detección automática por variables de entorno. Comparten el layout de destino `.agents/`
y difieren en el runtime declarado en `workspace.json`.

**Razón:** son los runtimes que el sistema heredado ya soporta. Codex se añade cuando su
layout esté verificado.

## D-022 — Compatibilidad con `kits-init` es obligatoria

**Estado:** Accepted
**Reemplaza la incógnita de:** `04-specification.md §12.9`

La CLI lee y escribe `workspace.json` v2 preservando los campos que no administra, y
ofrece `agent-kits import` para adoptar un workspace creado por `kits-init` generando su
lockfile a partir de los archivos presentes.

**Consecuencia:** un workspace puede alternar entre el flujo conversacional heredado y la
CLI sin corromperse. El catálogo heredado no se migra ni se mueve.

## D-023 — Conflictos: fail closed con anulación explícita

**Estado:** Accepted
**Refina:** D-011

Se comparan tres checksums: el registrado en el lockfile, el del archivo en disco y el
del contenido nuevo.

| Lockfile | Disco | Nuevo | Resultado |
|---|---|---|---|
| = | = | = | `unchanged` |
| = disco | ≠ nuevo | — | `update` |
| ≠ disco | — | — | `divergent` → bloquea |
| sin registro | existe | — | `adopt` requiere confirmación |

Un `divergent` devuelve `local_divergence` y no escribe nada. `--force` es la única forma
de sobrescribir y nunca es implícita.

## D-024 — Semver con constraints acotados

**Estado:** Accepted

Las versiones son SemVer 2.0.0 sin metadatos de build. Los constraints admitidos son:
exacto (`1.2.0`), caret (`^1.2.0`), tilde (`~1.2.0`) y cualquiera (`*` o ausente).

**Razón:** cubre los casos reales del catálogo sin implementar un solver completo. Rangos
compuestos requieren una decisión nueva.

## D-025 — Confianza por source y prohibición de ejecución

**Estado:** Accepted

Cada source declara `trust`: `trusted` o `review`. La CLI **nunca** ejecuta contenido del
catálogo, en ninguna circunstancia. Antes de escribir valida rutas, rechaza symlinks,
aplica un límite de tamaño por archivo y detecta secretos.

**Razón:** cierra la pregunta abierta sobre el modelo de confianza sin construir firmas
criptográficas en el MVP. Un repositorio público no implica confianza automática.

## D-026 — El catálogo heredado se consume mediante adaptador

**Estado:** Accepted

El loader de catálogos detecta el layout heredado (`catalog-index.md` + `skills/` +
`packs/`) y sintetiza manifests canónicos en memoria. No se reescribe ningún archivo del
catálogo actual.

**Razón:** permite que las 50 skills, 7 packs y 4 agentes existentes sean instalables por
la CLI nueva sin una migración destructiva, y mantiene el legado como fuente autoritativa
de su propio contenido.

## D-027 — Las referencias mutuas se informan, no se rechazan

**Estado:** Accepted
**Refina:** `RF-05` de `04-specification.md`

Un ciclo de referencias produce un diagnóstico `dependency_cycle` en el plan, no un error.
La resolución sigue detectando ciclos y los expone; lo que no hace es abortar.

**Razón:** la resolución calcula un *conjunto* de recursos y la instalación escribe
archivos independientes, así que no existe requisito de orden que un ciclo pueda violar.
Además el catálogo heredado contiene referencias mutuas legítimas en cuatro packs: un
workflow nombra al agente que lo orquesta y ese agente nombra el workflow que ejecuta.
Tratarlas como error fatal dejaba `context`, `design`, `backend-design` y
`fullstack-design` sin poder instalarse.

## D-028 — Dos recursos no pueden escribir el mismo archivo

**Estado:** Accepted

Si dos recursos distintos resuelven al mismo archivo de destino, el plan se bloquea con
`destination_conflict` y no escribe nada.

**Razón:** el flujo heredado resolvía esta situación con la regla "si ya fue instalado por
otro pack, no lo copies de nuevo", que conserva silenciosamente el primero. Los packs
`backend` y `frontend` traen ambos un `feature-development.md` con **contenido distinto**,
de modo que instalar los dos perdía una de las dos versiones sin avisar. El conflicto es
un defecto del catálogo que hay que corregir en el catálogo, no una ambigüedad que la
herramienta deba resolver por su cuenta.

## Decisiones todavía necesarias

- ubicación definitiva del registro global de reserva de IDs para sources remotas
  (el MVP valida unicidad sobre la vista agregada, que alcanza para sources conocidas);
- rangos de versión compuestos;
- firma criptográfica de sources;
- adaptador de Codex;
- topología del repositorio colaborativo (`origin`).
