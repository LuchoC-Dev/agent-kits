# Contexto del Proyecto — Sistema AI-First de Workspaces

> **Propósito de este archivo:** dar a cualquier sesión (humana o agente) el contexto
> completo del proyecto, sus decisiones de diseño y el camino recorrido, sin necesidad
> de leer todo el código. Si entrás nuevo: leé esto primero.
>
> Última actualización del documento: 2026-05-22.

---

## 1. Qué es este proyecto

Es un **sistema AI-first de workspaces** basado en **skills** y **packs** escritos en
Markdown. Sirve para que, al entrar a un proyecto de software, un agente pueda
**bootstrapear un espacio de trabajo estructurado** (`.agents/`) con memoria,
contexto, agentes, skills y workflows, y después **ejecutar fases de diseño e
implementación** de manera guiada y reproducible.

El caso de uso central: arrancar un proyecto nuevo (o sumar estructura a uno existente)
y llevarlo de forma ordenada desde el **descubrimiento del producto** hasta el
**diseño técnico** listo para implementar.

### Filosofía: AI-first, filesystem-first

- **El agente ES el runtime.** No hay un motor desplegado, ni servidor, ni proceso. Todo
  es Markdown que un agente lee e interpreta.
- **Filesystem-first.** El estado vive en archivos y carpetas. No hay base de datos.
- **Runtime mínimo.** Las "ejecuciones" son el agente siguiendo instrucciones escritas
  en los `SKILL.md`, `pack.md` y workflows.
- **Composición declarativa.** Los packs declaran qué componen; las skills declaran qué
  consumen y producen. La compatibilidad se valida leyendo frontmatter, no con un motor.

---

## 2. Ubicación y estructura

Directorio raíz del sistema:

```
.claude/skills/app-init/
```

> Nota: durante el testing local todo se movió bajo `.claude/skills/app-init/` para que
> Claude Code lo auto-descubra. La variable de entorno `MY_SYSTEM_HOME` (en
> `.claude/settings.local.json`) apunta acá. Claude Code auto-descubre `SKILL.md` solo
> un nivel de profundidad.

### Árbol actual

```
app-init/
├── SKILL.md                  ← thin launcher: trigger /app-init, lanza el agente
├── agent.md                  ← el agente app-init (todo el flujo de inicialización)
├── PROJECT-CONTEXT.md        ← este archivo
├── skills/                   ← POOL GLOBAL de skills (35 skills, plano)
│   ├── <skill-id>/SKILL.md
│   └── ...
├── agents/                   ← POOL GLOBAL de agentes Clase 2 (agnósticos al flujo)
│   ├── artifact-validator.md
│   ├── design-critic.md
│   ├── research-scout.md
│   ├── wireframe-renderer.md
│   └── ...
├── meta/                     ← agentes META de autoría del sistema (no se distribuyen)
│   └── catalog-author.md
├── packs/                    ← los packs (composiciones de skills)
│   ├── context/
│   ├── design/
│   ├── backend-design/
│   ├── fullstack-design/
│   ├── frontend/
│   ├── backend/
│   └── tools/
│       └── cada pack: pack.md + agents/ + workflows/
└── tests/                    ← sandboxes de prueba (sandbox … sandbox6)
```

---

## 3. Los tres conceptos centrales

### Skill

Una unidad de capacidad. Vive en `skills/<id>/SKILL.md`. Tiene un rol, un workflow
interno, reglas duras y un formato de output. **Una skill no sabe nada del flujo**: no
sabe en qué fase corre, ni quién la invoca, ni qué número de archivo le toca. Solo
declara, en su frontmatter, **qué artefactos consume y qué artefacto produce**.

### Pack

Una **composición** de skills + agentes + workflows para un dominio concreto (ej.
"diseño de backend"). Vive en `packs/<id>/`. Contiene únicamente:

```
packs/<id>/
├── pack.md          ← manifest: qué skills/agentes/workflows compone, qué consume/produce
├── agents/          ← los agentes que orquestan los workflows del pack
└── workflows/       ← los workflows (la secuencia de fases)
```

El pack **NO contiene skills**. Solo las **referencia por `id`**. Las skills viven en el
pool global `skills/`.

### Workflow

La secuencia de fases: qué skill corre, en qué orden, con qué modo. El workflow **SÍ
conoce el flujo** — es su responsabilidad. Asigna a cada fase su número de archivo (`NN`)
y decide qué skills entran.

### Agente — dos clases

Un agente es un **actor con identidad** que corre, típicamente, **en su propio contexto
aislado** e **invoca** una o más skills para ejecutar. No es lo mismo que una skill:

- **Skill** = el *conocimiento* de cómo hacer algo (instrucciones, reglas, output).
- **Agente** = el *ejecutor* que corre ese conocimiento en una sesión enfocada.

Skill y agente **conviven, no compiten**: el agente usa la skill.

Hay **dos clases de agente**, y la clase determina dónde vive:

| Clase | Qué es | Conoce el flujo | Dónde vive |
|---|---|---|---|
| **1 — Orquestador de workflow** | Conduce un workflow completo de un pack (ej. `backend-designer`, `fullstack-designer`, `context-builder`). Su identidad *es* el flujo. | **Sí** | En el **pack** (`packs/<id>/agents/`) |
| **2 — Agente de tarea específica** | Especialista en *una* tarea acotada (ej. un agente de wireframe, un validador de reglas del proyecto). No orquesta un workflow. | **No** — agnóstico | **Pool global** `agents/` si es general o reusado; en el pack si es exclusivo de él |

Además, existe una clase aparte de agente que **no opera sobre proyectos de usuario sino
sobre el catálogo del sistema app-init en sí mismo**:

| Clase | Qué es | Dónde vive |
|---|---|---|
| **Meta — agente de autoría del sistema** | Autora o mantiene el catálogo (skills, packs, agentes, workflows). Ej. `catalog-author`. **No se distribuye a workspaces de usuario.** | Carpeta `meta/` (hermana de `skills/` y `agents/`) |

**Regla unificada de ubicación:** *agnóstico al flujo → pool global; conoce el flujo →
pack.* Es la misma regla que separa skills (pool) de orquestadores (pack). El criterio
para mover un agente de Clase 2 al pool es la **reutilización**, no la simetría: un
validador de reglas lo quiere cada pack → pool; un agente de tarea exclusivo de un pack
puede quedarse en él.

### Contrato de invocación de agentes Clase 2

Un orquestador (Clase 1) declara los agentes Clase 2 que necesita en su frontmatter:

```yaml
uses_agents:
  - artifact-validator
  - design-critic
```

**Resolución de id:** igual que las skills. Al instalar un pack, `app-init` resuelve
cada `id` contra `<global>/agents/<id>.md` y lo copia a `.agents/agents/`. Los
agentes `meta/` **nunca se distribuyen**.

**Patrón de delegación:** el orquestador invoca el agente Clase 2 como sub-agente
aislado para no inflar su propio contexto. El agente Clase 2 recibe inputs concretos
(rutas, preguntas, directivas), corre en contexto propio, devuelve un output compacto
(reporte, comparación, archivo) y termina. El orquestador decide qué hacer con ese
output — aprobar, reintentar, escalar al usuario.

---

## 4. El camino recorrido (decisiones clave)

Este proyecto evolucionó a través de varias decisiones de diseño importantes. Entender
**por qué** se tomaron es clave para no revertirlas sin querer.

### 4.1 — Contrato de integración entre packs (declarativo, no motor)

Se decidió un contrato **declarativo liviano**, NO un motor de compatibilidad en runtime
(eso sería overengineering). Dos piezas, ambas Markdown puro:

1. **`produces` / `consumes` en el frontmatter de cada `pack.md`.** Ej: `backend-design`
   consume `product-context` (de `context`) y `tech-decisions` (de `design`).
2. **Frontmatter de identidad en cada artefacto generado.** Cada archivo de output lleva
   `pack: <pack>` y `artifact: <nombre>` para poder ser encontrado por su identidad y no
   por su nombre de archivo.

### 4.2 — Skills independientes del flujo y de su pack

**Problema detectado:** las skills "sabían" demasiado. Sus títulos decían
`# User Flows — Fase 3`, sus descripciones decían "Fase N del diseño de backend", y su
prosa referenciaba números de fase y nombres de archivo concretos. Eso acopla la skill
al workflow: la misma skill no se puede reutilizar en otro orden o en otro pack.

**Decisión:** una skill debe ser **invocable** y agnóstica. El **flujo** (dónde y cuándo
corre) lo decide el workflow. La skill solo declara su **contrato de input/output**.

Distinción importante que se mantuvo:

| Conocimiento de flujo (PROHIBIDO en skill) | Contrato de input (LEGÍTIMO en skill) |
|---|---|
| "Fase 3 del Diseño de Backend" | "Consumo el artefacto `api-contract`" |
| "el paso 5 del workflow" | `consumes: [{artifact: design-tokens}]` |
| nombre de archivo `03-domain-model.md` | "buscalo por su frontmatter `artifact: domain-model`" |

### 4.3 — Pool global de skills (sacarlas de los packs)

**Antes:** las skills vivían dentro de cada pack: `packs/<pack>/skills/<id>/SKILL.md`.
**Problema:** `find-docs` estaba **duplicada idéntica** en `packs/backend/skills/` y
`packs/frontend/skills/`. Eso prueba que la ubicación per-pack estaba mal.

**Ahora:** todas las skills viven en un único **pool global plano**:
`app-init/skills/<id>/SKILL.md` (35 skills únicas, `find-docs` deduplicada).

Cada `pack.md` referencia las skills **solo por `id`**. Se eliminaron los mecanismos
`source: ./skills/<id>/SKILL.md` y `from: <pack>` de las entradas de skills. Al instalar
un pack, cada `id` se resuelve contra `<global>/skills/<id>/`.

> Cuidado: el `from:` que aparece dentro de los bloques `consumes:` de un `pack.md`
> **sí es legítimo** — declara de qué pack viene un artefacto consumido. Es distinto del
> `from:` que se eliminó de las entradas de skills (que indicaba ubicación).

### 4.4 — Convención `NN-` para numeración de outputs

Las skills **no inventan** el número de archivo. En su sección de Output usan el
placeholder `NN-`:

```
docs/backend-design/NN-data-model.md
```

> `NN` = prefijo numérico de dos dígitos que asigna el **workflow** según el orden real
> de ejecución. La skill no lo inventa — lo toma de la tabla del workflow que la invoca.

Esto permite que la misma skill aparezca en distinto orden según el modo del workflow,
sin tener un número hardcoded.

### 4.5 — `fullstack-design`: tracks paralelos + carpeta `shared/`

El pack `fullstack-design` **no redefine fases** — compone las skills de `design`
(frontend) y `backend-design` en dos tracks paralelos. Algunas skills se corren **una
sola vez** para los dos tracks (ej. `discovery`, `tech-decisions` con `scope: fullstack`).
Los artefactos transversales van a una carpeta `docs/shared/`.

### 4.6 — Dos clases de agente (decisión 2026-05-21)

Al evaluar si convenía aplicar a los agentes el mismo rediseño que a las skills, se
concluyó que **la analogía es parcial**: un agente orquestador *debe* conocer el flujo
(es su razón de existir), así que no se le puede quitar como a una skill. Pero sí surgió
una **categoría nueva** de agente que todavía no existe en el proyecto.

**Decisión:** distinguir formalmente **dos clases de agente** (ver §3):

1. **Orquestadores de workflow** — los que ya existen. Viven en el pack. No se mueven.
2. **Agentes de tarea específica** — agnósticos al flujo, especialistas en una tarea
   (wireframe, validación de reglas, etc.). Cuando se creen, los generales/reusados van
   a un **pool global `agents/`** (hermano de `skills/`), referenciados por `id`; los
   exclusivos de un pack se quedan en él.

Esto eleva el principio "agnóstico → pool / conoce el flujo → pack" a **regla vigente
del proyecto** para todo lo nuevo, no solo para skills.

### 4.7 — Skills de disciplina de desarrollo (decisión 2026-05-22)

Los packs de implementación (`backend`, `frontend`) tenían un único workflow genérico
(`feature-development`) sin soporte para metodologías de trabajo. Se decidió incorporar
**disciplinas de desarrollo** — TDD, BDD, contract-first, trunk-based — **como skills**,
no como workflows ni como pack nuevo. SDD se dejó afuera (ya existe maduro en el entorno).

Una **skill de disciplina** es distinta de una skill de diseño: no produce un
documento-artefacto en `docs/`, sino que rige *cómo se ejecuta* el código (tests,
contratos, ramas). Lleva `discipline: true` y `combinable: true` en el frontmatter.

Propiedades acordadas: las disciplinas son **autónomas** (cada una funciona sola, no
depende de otra) y **combinables** (cualquier subconjunto activable a la vez, ninguna es
mutuamente excluyente).

**Modelo de activación — híbrido:**

1. **`workspace.json` registra QUÉ disciplinas están activas** — campo `disciplines:
   ["tdd", "bdd", ...]`. Es una decisión de proyecto, no de feature individual.
2. **El workflow `feature-development` sabe DÓNDE se engancha cada una** — lee
   `workspace.json` e invoca condicionalmente solo las activas. Puntos de enganche:
   `contract-first`→paso design; `bdd`→paso behavior-spec antes de implementation;
   `tdd`→envuelve implementation; `trunk-based`→pasos commit y pr.

La skill queda **agnóstica al flujo** — describe la técnica, igual que `git-commit`
describe cómo commitear sin saber que es el paso 8. El workflow es el único que conoce
dónde se enchufa cada disciplina.

### 4.8 — `app-init` rediseñado de skill a agente (decisión 2026-05-22)

`app-init` era una **skill orquestadora**: el `SKILL.md` contenía todo el procedimiento
de bootstrap. Se decidió convertirlo en un **agente** — un actor con identidad que corre
en contexto propio y conduce al usuario, pregunta por pregunta, en armar su workspace.

Decisiones de diseño tomadas:

- **Taxonomía: caso único raíz.** `app-init` no es Clase 1, ni Clase 2, ni Meta — es el
  **punto de entrada del sistema**. No se abrió una categoría nueva: es un agente especial
  que vive en la raíz `app-init/` como `agent.md`.
- **Invocación: thin launcher.** El `SKILL.md` se reescribió como un **lanzador delgado**:
  conserva el trigger `/app-init` y "cuándo actuar", y su única acción es lanzar el agente
  `app-init` (`agent.md`) en contexto propio. Todo el flujo real vive en el agente.
- **Flujo enriquecido + disciplinas.** El agente agrega la **Fase 4bis — Disciplinas de
  desarrollo**: detecta señales del stack (`.feature`→bdd, `openapi.*`→contract-first,
  config de tests→tdd) y **propone**; el usuario confirma o ajusta con `AskUserQuestion`.
- **`workspace.json` schema v2.** Nuevo campo `disciplines: [...]` — fuente de verdad de
  las disciplinas activas, leído por los workflows `feature-development`. Un workspace v1
  sin el campo se trata como `disciplines: []`; la Fase 6 (Upgrade) lo agrega.

Trabajo pendiente derivado: cómo se distribuyen/empaquetan las skills de disciplina
(antes en §10) queda absorbido por este agente — la Fase 4bis las instala y registra.

---

## 5. Los packs actuales

| Pack | Propósito | Workflow |
|---|---|---|
| **context** | Descubrimiento de producto: problema, stakeholders, dominio, job-stories, visión, riesgos | `context-building` |
| **design** | Diseño pre-build del frontend: IA, flujos, componentes, tokens, wireframes, a11y, performance, stack | `pre-build-design` |
| **backend-design** | Diseño técnico de backend pre-build: API, datos, arquitectura, convenciones, integraciones, no-funcionales, testing, seguridad | `backend-design` |
| **fullstack-design** | Orquesta `design` + `backend-design` en tracks paralelos | `fullstack-design` |
| **frontend** | Implementación de frontend | `feature-development` |
| **backend** | Implementación de backend | `feature-development`, `environment-bootstrap` |
| **tools** | Herramientas transversales: git, gh, docker, env, taskfile. Sin agentes ni workflows. | — |

### Flujo recomendado entre packs

```
context  →  design          ┐
         →  backend-design   ┘ → frontend / backend (implementación)
```

`context` alimenta tanto a `design` como a `backend-design`, que corren en paralelo.
`fullstack-design` es la versión orquestada de esos dos juntos.

---

## 6. Las 39 skills del pool global

**Context (6):** `problem-framing`, `stakeholders-map`, `domain-modeling`,
`job-stories`, `vision`, `assumptions-risks`.

**Design / frontend pre-build (11):** `discovery`, `information-architecture`,
`user-flows`, `component-inventory`, `design-system-tokens`, `wireframes`,
`accessibility-plan`, `performance-budget`, `tech-decisions`, `html-mockup`,
`progressive-design`.

**Backend-design (8):** `api-contract`, `data-model`, `service-architecture`,
`code-conventions`, `integrations-auth`, `nonfunctional`, `testing-strategy`,
`security-design`.

**Tools (6):** `git-commit`, `branch-pr`, `gh-cli`, `environment-setup`,
`docker-expert`, `taskfile-setup`.

**Transversales (4):** `find-docs`, `build-progress`, `brainstorming`, `ui-ux-pro-max`.

**Disciplinas de desarrollo (4):** `tdd`, `bdd`, `contract-first`, `trunk-based`.
Skills de disciplina (`discipline: true`) — rigen *cómo* se ejecuta el desarrollo, no
producen documento-artefacto. Ver §4.7.

---

## 7. Anatomía de un `SKILL.md`

```markdown
---
name: data-model
description: <qué hace + triggers de invocación — NO números de fase>
consumes:
  - artifact: domain-model
    required: false
  - artifact: tech-decisions
    required: false
produces:
  - artifact: data-model
---

# Data Model            ← título limpio, sin "— Fase N"

## Rol                  ← quién es el agente para esta skill
## Parámetro de depth   ← light / full (adapta la profundidad)
## Precondición         ← qué artefactos necesita (por nombre de artifact)
## Workflow             ← pasos internos de la skill
## Output               ← formato del archivo, con path `NN-<nombre>.md`
## Reglas duras         ← invariantes no negociables
```

Reglas que sigue **toda** skill ya limpiada:

- Título sin número de fase.
- Description sin "Fase N de …".
- `consumes:` / `produces:` declarados en frontmatter.
- Output con placeholder `NN-`, nunca un número hardcoded.
- Referencias a otros artefactos por su `artifact:` del frontmatter, no por nombre de
  archivo.

---

## 8. El agente `app-init` (bootstrap)

`app-init` es el **agente de inicialización** del sistema — el punto de entrada. Se
compone de dos archivos en la raíz `app-init/`:

- **`SKILL.md`** — *thin launcher*. Conserva el trigger `/app-init` y "cuándo actuar". Su
  única acción es **lanzar el agente** en contexto propio. No contiene procedimiento.
- **`agent.md`** — el **agente** propiamente dicho: identidad, principios y todo el flujo
  de inicialización (6 fases + Fase 4bis de disciplinas).

Su responsabilidad es **inicializar `.agents/`** en el directorio actual de forma
conversacional y guiada: detecta el stack, descubre packs y skills del sistema global,
pregunta por la composición y por las disciplinas de desarrollo, y registra todas las
decisiones en `workspace.json` (schema v2, con campo `disciplines`).

No escribe contenido de skills ni agentes — solo arma la estructura y distribuye lo que
el catálogo global ya tiene. Al instalar un pack, resuelve cada `id` de skill contra el
pool global `<global>/skills/<id>/`.

Ver §4.8 para las decisiones de diseño del rediseño skill → agente.

---

## 9. Estado actual

✅ **Completado:**

- 35 skills en el pool global plano `app-init/skills/`.
- Packs reducidos a `pack.md + agents/ + workflows/`; referencian skills solo por `id`.
- Eliminados `source:` y `from:` de las entradas de skills en los `pack.md`.
- 25 skills limpiadas de conocimiento de flujo (títulos, descripciones, prosa).
- Frontmatter `consumes:` / `produces:` formalizado en esas 25 skills.
- Convención `NN-` aplicada en todas las skills, incluidas las de `context`.
- `app-init/SKILL.md` actualizado al modelo de pool global y al pool `agents/` (Clase 2 se resuelve por `id` contra `<global>/agents/`; `meta/` no se distribuye nunca).
- Verificación: cero referencias "Fase N" y cero paths `01-06` hardcoded en las skills.
- Pool global `agents/` creado (hermano de `skills/`). Agentes de Clase 2 en pool global: `artifact-validator`, `design-critic`, `research-scout`, `wireframe-renderer`. Agente de Clase 2 exclusivo de pack: `cross-track-auditor` (en `packs/fullstack-design/agents/`).
- Carpeta `meta/` creada para agentes de autoría del sistema. Primer agente meta: `catalog-author` — crea skills/agentes/workflows/packs desde un brief detallado y los engancha en todas sus referencias.
- **Contrato de invocación de agentes Clase 2** formalizado: frontmatter `uses_agents: [id, ...]` en los 4 orquestadores; patrón de delegación y resolución documentado en §3.
- **Agentes Clase 2 integrados** en todos los orquestadores: secciones `## Sub-agentes` en `designer`, `backend-designer`, `fullstack-designer`, `context-builder` con tabla de momentos de invocación. `wireframe-renderer` documentado en `pre-build-design` para fase 6 HTML/Progressive.
- **Numeración `NN` unificada** en los 4 workflows: frontmatter usa `NN-*.md`; tablas de modos son la única fuente de verdad de los números; columnas `NN` explícitas en `backend-design` y `context-building`.
- **Skills de disciplina de desarrollo** creadas en el pool global: `tdd`, `bdd`, `contract-first`, `trunk-based` (`discipline: true`, `combinable: true`). Ver §4.7.
- **Workflows `feature-development`** de `backend` y `frontend` modificados: paso 0 `disciplines-check` que lee `workspace.json`; tabla de puntos de enganche; invocación condicional de cada disciplina activa; paso `behavior-spec` condicional a `bdd`.
- **`app-init` rediseñado de skill a agente** (ver §4.8): `SKILL.md` reescrito como thin launcher; `agent.md` nuevo con identidad, principios y el flujo completo; Fase 4bis de disciplinas (detección + confirmación); `workspace.json` schema v2 con campo `disciplines`.

## 10. Trabajo pendiente

- [ ] **Tools skills** (`git-commit`, `branch-pr`, `gh-cli`, `environment-setup`,
  `docker-expert`, `taskfile-setup`, y transversales): podrían recibir `consumes:` /
  `produces:`. Opcional — no tenían referencias de fase, así que quedaron sin tocar.
- [ ] **Agentes de Clase 2:** todos los planificados están implementados e integrados. Ver §3, §4.6 y el contrato de invocación (§3 "Contrato de invocación") si se agregan nuevos.

---

## 11. Glosario rápido

| Término | Significado |
|---|---|
| **Skill** | Unidad de capacidad agnóstica al flujo. `skills/<id>/SKILL.md`. |
| **Pack** | Composición de skills/agentes/workflows para un dominio. |
| **Workflow** | Secuencia de fases; conoce el flujo y asigna los números `NN`. |
| **Agente** | Actor que corre en contexto propio e invoca skills. Clase 1 (orquestador, en el pack) o Clase 2 (tarea específica, pool global si es general). |
| **Artefacto** | Archivo de output, identificado por frontmatter `artifact:`. |
| **`consumes` / `produces`** | Contrato declarativo de input/output de skills y packs. |
| **`NN-`** | Placeholder de prefijo numérico; lo resuelve el workflow. |
| **Pool global** | `app-init/skills/` — todas las skills, plano, compartido. |
| **`.agents/`** | Workspace que `app-init` genera en cada proyecto. |
| **`app-init`** | Agente de inicialización — punto de entrada del sistema. `SKILL.md` (thin launcher) + `agent.md` (el agente). Ver §8. |
| **Skill de disciplina** | Skill con `discipline: true` que rige *cómo* se desarrolla (tdd, bdd, contract-first, trunk-based); no produce documento-artefacto. |
| **`disciplines`** | Campo de `workspace.json` (schema v2); ids de las disciplinas activas del proyecto. |
| **AI-first** | El agente es el runtime; todo es Markdown; no hay motor. |
