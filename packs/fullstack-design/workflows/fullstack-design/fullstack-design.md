---
id: fullstack-design
description: Workflow orquestador de diseño fullstack pre-build. Compone el track frontend (skills de design) y el track backend (skills de backend-design) en un proceso coordinado con 3 tiers (Lite/Standard/Full) y 2 estrategias de flujo (Separado / Integrado).

agent: fullstack-designer

tiers: [lite, standard, full]

flow_strategies: [separado, integrado]

shared_phases:
  - { skill: tech-decisions, scope: fullstack, folder: docs/shared/ }

modes:
  lite:
    tier: basic
    depth: light
    backend: [api-contract, data-model, service-architecture]
    frontend: [wireframes]
    output: docs/shared/01-tech-decisions.md + docs/backend-design/backend-design.md + docs/design/design.md
    gate: approval-at-end

  standard:
    tier: medium
    depth: full
    backend: [api-contract, data-model, service-architecture, code-conventions, integrations-auth]
    frontend: [information-architecture, user-flows, component-inventory, design-system-tokens, visual-design]
    output: docs/shared/ + docs/backend-design/ + docs/design/ (archivos numerados)
    gate: approval-per-block

  full:
    tier: advanced
    depth: full
    backend: [api-contract, data-model, service-architecture, code-conventions, integrations-auth, nonfunctional, testing-strategy]
    frontend: [information-architecture, user-flows, component-inventory, design-system-tokens, visual-design, accessibility-plan, performance-budget]
    optional: [security-design]
    output: docs/shared/ + docs/backend-design/ + docs/design/ (archivos numerados)
    gate: approval-per-phase
---

# Fullstack Design Workflow

Workflow que **orquesta** el diseño de frontend y backend a la vez. No define fases propias: compone las skills de `design` y `backend-design`.

## Filosofía

> Las dos mitades de un fullstack se diseñan sobre la misma base y con un stack coherente. Un solo proceso, no dos.

## Setup único

Antes de cualquier fase, el agente `fullstack-designer`:

### 1. Detectar trabajo previo

Lee el `.workflow-state.json` (en `docs/`) si existe:
- `status: "completed"` → ofrecer "revisar / actualizar".
- `status: "in-progress"` → retomar desde `current_phase`.
- Nada → arrancar limpio.

### 2. Detectar contexto

Detecta `product-context` (pack `context`) por su frontmatter de identidad. Si está, alimenta el track backend (`domain-model` → datos y arquitectura) y el track frontend (jobs, usuarios).

### 3. Elegir el tier

Tier **pareado** — un solo nivel para ambos tracks:

| Tier | Para qué |
|---|---|
| **Lite** | Fullstack chico, MVP, prototipo |
| **Standard** | Fullstack de complejidad normal |
| **Full** | Sistema grande, varios dominios, equipo grande |

El agente recomienda; el usuario decide.

### 4. Elegir la estrategia de flujo

**Decisión nueva, independiente del tier.** Cómo se entrelazan los dos tracks:

| Estrategia | Cómo corre | Cuándo conviene |
|---|---|---|
| **Separado** | Un track completo y después el otro. El usuario elige el orden (backend-primero o frontend-primero). | Equipos separados de back y front; cuando una mitad tiene que cerrarse antes de tocar la otra; cuando se quiere revisar cada mundo de corrido. |
| **Integrado** | Las fases se entrelazan en **pares** (una de backend + una de frontend que se informan mutuamente). Gate conjunto por par. | Cuando el contrato de API y los flujos de UI se benefician de diseñarse en diálogo. Suele dar un resultado más coherente. **Recomendado por defecto.** |

El agente recomienda **Integrado** salvo que el contexto indique equipos separados o una mitad ya cerrada. El usuario decide.

Si elige **Separado**, preguntar el orden:
- **Backend-primero** (default) — `api-contract` y `data-model` quedan listos antes de que el frontend diseñe flujos contra endpoints reales.
- **Frontend-primero** — útil cuando la UX manda y el backend se moldea para servirla.

La estrategia y el orden **no cambian el layout de carpetas** — solo el orden de generación.

### 5. Evaluar la fase de seguridad

Igual que en `backend-design`: preguntar si el sistema maneja datos sensibles, está regulado o tiene alta exposición. Si sí → el track backend suma `security-design`. Disponible en Standard y Full.

## La fase compartida

`tech-decisions` (con `scope: fullstack`) corre **siempre primero**, antes de cualquier track, en **las dos estrategias**. Decide el stack completo de una sola vez y satisface el Paso 0 del track backend (el backend no vuelve a decidir el stack).

> El `discovery` no es una fase de este workflow: lo cubre el `product-context` del pack `context` (en `docs/context/`). Si no existe, el agente hace un mini-intake antes de `tech-decisions`.

## Estrategia Separado

```
Fase compartida:  tech-decisions (scope fullstack)  → docs/shared/
Track A completo  →  Track B completo
   (el orden A/B lo eligió el usuario en el Setup paso 4)
```

Track backend: `api-contract → data-model → service-architecture → code-conventions → integrations-auth → [nonfunctional] → [testing-strategy] → [security-design]`

Track frontend: `information-architecture → user-flows → component-inventory → design-system-tokens → visual-design → [accessibility-plan] → [performance-budget]`

## Estrategia Integrado

Después de `tech-decisions`, las fases corren **en pares** backend ⇄ frontend, eligiendo en cada paso fases que se informan mutuamente. Gate conjunto al cerrar cada par.

| Par | Backend | Frontend | Por qué juntas |
|---|---|---|---|
| 1 | `data-model` | `information-architecture` | qué datos existen ↔ cómo se navegan |
| 2 | `api-contract` | `user-flows` | el contrato y los flujos se diseñan en diálogo |
| 3 | `service-architecture` | `component-inventory` | unidades de servidor ↔ unidades de UI |
| 4 | `code-conventions` | `design-system-tokens` | reglas de código ↔ reglas de diseño |
| 5 | `integrations-auth` | `visual-design` | auth/integraciones ↔ pantallas finales |
| 6 *(Full)* | `nonfunctional` | `accessibility-plan` | calidad no-funcional de cada mundo |
| 7 *(Full)* | `testing-strategy` | `performance-budget` | estrategia de verificación de cada mundo |
| 8 *(opc.)* | `security-design` | — | deep-dive de seguridad |

El agente puede ajustar el emparejamiento si el proyecto lo pide, pero **mantiene la regla**: cada par es una unidad de trabajo y de aprobación.

## Layout de carpetas (igual en las dos estrategias)

```
docs/
├── context/                  ← input: product-context (no lo genera este workflow)
├── shared/                   ← artefactos cross-cutting
│   └── 01-tech-decisions.md
├── backend-design/           ← track backend
│   ├── 01-api-contract.md
│   └── …
├── design/                   ← track frontend
│   ├── 01-information-architecture.md
│   └── …
└── design-index.md           ← índice generado al cierre
```

## Convención de numeración

Las skills escriben con prefijo placeholder `NN-`. **El número lo asigna este workflow.**

- Numeración **por carpeta**, contigua, de dos dígitos: `docs/shared/`, `docs/backend-design/` y `docs/design/` se numeran cada una `01..N` por separado.
- El número de un archivo = la posición de su fase **dentro de su track** (no en el orden global). Así, aunque en modo Integrado las fases se intercalen, cada carpeta queda `01, 02, 03…` sin huecos.
- `docs/shared/` hoy lleva un solo archivo: `01-tech-decisions.md`.
- En tier **Lite** el output es consolidado: `docs/backend-design/backend-design.md` y `docs/design/design.md` (sin numerar); `docs/shared/01-tech-decisions.md` se mantiene como archivo aparte.

## Convención de artefactos

Cada archivo lleva el **mismo frontmatter de identidad** que usan `design` y `backend-design`. No se inventan artefactos: `fullstack-design` produce `design-spec`, `backend-design-spec` y `tech-decisions`, tal cual los esperan los packs de implementación, que los identifican por el campo `artifact:` — nunca por la ruta ni el número.

## Reglas duras

- **El stack se decide una sola vez** — `tech-decisions` con `scope: fullstack`, en `docs/shared/`. El track backend no ejecuta su Paso 0.
- **`tech-decisions` corre primero** en ambas estrategias, antes de cualquier track.
- **No se cambia de tier ni de estrategia a mitad.** Para cambiar se reinicia y se reusan los outputs.
- **La numeración la asigna el workflow** — por carpeta, contigua. Las skills traen `NN-` placeholder.
- **Se respeta el contrato** — outputs en `docs/shared/`, `docs/backend-design/` y `docs/design/`, sin artefactos nuevos.

## Estado del workflow

```json
{
  "workflow": "fullstack-design",
  "tier": "standard",
  "flow_strategy": "integrado",
  "track_order": null,
  "started_at": "2026-05-21",
  "status": "in-progress",
  "security_phase": true,
  "completed_phases": ["tech-decisions", "data-model", "information-architecture"],
  "current_phase": "api-contract",
  "current_track": "backend",
  "last_updated": "2026-05-21T17:00:00Z"
}
```

Guardado en `docs/.workflow-state.json`.

- `flow_strategy` es `"separado"` o `"integrado"`.
- `track_order` es `"backend-primero"` / `"frontend-primero"` cuando la estrategia es Separado; `null` cuando es Integrado.
- `status` arranca en `"in-progress"` y pasa a `"completed"` al aprobar la última fase.
- `security_phase` registra si la fase opcional de seguridad entró.
- Apenas se aprueba una fase (o un par, en Integrado): agregar a `completed_phases`, actualizar `last_updated`, **después** mover `current_phase`.
- `started_at` y `last_updated` con fecha/hora reales — nunca placeholders.

### Cierre

Al aprobar la última fase: `status: "completed"`, `current_phase: null`, `last_updated` real. El archivo se conserva.

Además, el agente genera `docs/design-index.md`: un índice que lista los documentos de los tres folders **en el orden real en que se ejecutaron** (especialmente útil en modo Integrado, donde el orden de ejecución no se lee de la numeración por carpeta), con un link a cada uno y el tier/estrategia elegidos.

## Qué viene después

Los packs `frontend` y `backend` (implementación) consumen `design-spec` y `backend-design-spec` — exactamente como si los hubieran producido los packs especialistas.
