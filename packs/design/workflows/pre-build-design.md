---
id: pre-build-design
description: Workflow de diseño pre-build con 3 modos de scope (Basic, Medium, Advanced). Selecciona el subset de fases adecuado al tamaño del problema y produce documentación en docs/design/.

agent: designer

modes:
  basic:
    description: Para landings, MVPs, prototipos y apps pequeñas
    steps:
      - { skill: discovery, depth: light }
      - { skill: wireframes, depth: light }
      - { skill: tech-decisions, depth: light }
    output: docs/design/design.md
    gate: approval-at-end

  medium:
    description: Para apps de tamaño medio y complejidad normal
    steps:
      - { skill: discovery, depth: full }
      - { skill: information-architecture, depth: full }
      - { skill: user-flows, depth: full }
      - { skill: component-inventory, depth: full }
      - { skill: design-system-tokens, depth: full }
      - { phase: "visual-design", selector: true }
      - { skill: tech-decisions, depth: full }
    output: docs/design/NN-*.md
    gate: approval-per-block
    notes: |
      Accessibility y performance se incluyen inline en wireframes y tech-decisions,
      no como fases separadas.

  advanced:
    description: Para apps grandes, sistemas críticos, equipos grandes
    steps:
      - { skill: discovery, depth: full }
      - { skill: information-architecture, depth: full }
      - { skill: user-flows, depth: full }
      - { skill: component-inventory, depth: full }
      - { skill: design-system-tokens, depth: full }
      - { phase: "visual-design", selector: true }
      - { skill: accessibility-plan, depth: full }
      - { skill: performance-budget, depth: full }
      - { skill: tech-decisions, depth: full }
    output: docs/design/NN-*.md
    gate: approval-per-phase

visual_design_phase:
  description: Fase 6 — selector entre 3 opciones de fidelidad visual
  options:
    - id: ascii
      skill: wireframes
      depth: full
      cost: low
      description: Wireframes ASCII low-fi. Valida estructura y jerarquía rápido.
    - id: html
      skill: html-mockup
      cost: medium
      description: Mockups HTML+Tailwind con fidelity por pantalla (mid-fi-plain | mid-fi-tokens | hi-fi).
    - id: progressive
      skill: progressive-design
      cost: variable
      description: Empieza con ASCII, aprueba estructura, eleva a HTML mid-fi, opcional hi-fi en pantallas críticas. Recomendado para Medium y Advanced.
---

# Pre-Build Design Workflow

Workflow de **diseño pre-build** que se adapta al tamaño del problema. Una sola fuente de verdad, tres experiencias.

## Filosofía

> Diseñar antes de codear. Adaptar la profundidad al tamaño del problema.

Aplicar 9 fases a una landing es overkill. Aplicar 3 fases a un sistema grande es irresponsable. El workflow te deja elegir.

## Inicio del workflow

Antes de arrancar, el agente `designer` hace **dos cosas**:

### 1. Detectar contexto previo

- ¿Existe `docs/design/` con archivos previos? → modo "continuar"
- ¿Existe un pack de context-building con outputs previos (BDD/SDD)? → consumirlo como insumo del intake
- ¿Hay conversación reciente con descripción del producto? → usarla como base

### 2. Seleccionar modo

Preguntar al usuario qué modo usar, recomendando uno según el contexto detectado:

| Modo | Cuándo recomendarlo |
|---|---|
| **Basic** | Landing, MVP, prototipo, herramienta interna chica, "una pantalla y listo" |
| **Medium** | App con varias pantallas, login + dashboard, complejidad normal |
| **Advanced** | E-commerce completo, SaaS multi-tenant, sistema crítico, equipo grande |

El agente recomienda; el usuario decide.

## Modo Basic

3 fases en versión light → 1 documento consolidado.

| # | Skill | Depth |
|---|---|---|
| 1 | `discovery` | light |
| 2 | `wireframes` | light |
| 3 | `tech-decisions` | light |

**Output:** `docs/design/design.md` con 3 secciones (no archivos separados).

**Gate:** se entrega todo de corrido y se aprueba al final.

## Modo Medium

7 fases, archivos separados, gates cada 2-3 fases.

| # | Skill | Output (`NN` = posición en esta tabla) |
|---|---|---|
| 1 | `discovery` | `01-discovery.md` |
| 2 | `information-architecture` | `02-information-architecture.md` |
| 3 | `user-flows` | `03-user-flows.md` |
| 4 | `component-inventory` | `04-component-inventory.md` |
| 5 | `design-system-tokens` | `05-design-tokens.md` |
| 6 | **visual-design** (selector) | `06-<slug-de-la-opcion-elegida>.md` |
| 7 | `tech-decisions` | `07-tech-decisions.md` |

**A11y y performance:** integrados como secciones dentro de `wireframes` y `tech-decisions` (no como fases separadas).

**Gates de aprobación:**

- Después de fase 1 (discovery)
- Después de fase 3 (flows)
- Después de fase 5 (tokens)
- Después de fase 7 (tech-decisions)

## Modo Advanced

9 fases completas, archivos separados, gate por fase.

| # | Skill | Output (`NN` = posición en esta tabla) |
|---|---|---|
| 1 | `discovery` | `01-discovery.md` |
| 2 | `information-architecture` | `02-information-architecture.md` |
| 3 | `user-flows` | `03-user-flows.md` |
| 4 | `component-inventory` | `04-component-inventory.md` |
| 5 | `design-system-tokens` | `05-design-tokens.md` |
| 6 | **visual-design** (selector) | `06-<slug-de-la-opcion-elegida>.md` |
| 7 | `accessibility-plan` | `07-accessibility-plan.md` |
| 8 | `performance-budget` | `08-performance-budget.md` |
| 9 | `tech-decisions` | `09-tech-decisions.md` |

**Gates:** aprobación explícita después de cada fase.

## Fase 6 — Selector de Visual Design

En modos **Medium** y **Advanced**, la fase 6 no es una skill fija. Es un selector entre 3 opciones:

| Opción | Skill | Agente Clase 2 | Costo | Cuándo elegirla |
|---|---|---|---|---|
| **ASCII** | `wireframes` | — | Bajo | Apps chicas, validación rápida, presupuesto de tokens ajustado |
| **HTML** | `html-mockup` | `wireframe-renderer` | Medio-alto | Estructura ya clara, querés sentir el producto, fidelity por pantalla |
| **Progressive** | `progressive-design` | `wireframe-renderer` (etapa mid-fi) | Variable | Querés iterar — ASCII → mid-fi HTML → hi-fi selectivo. **Recomendado** |

En las opciones **HTML** y **Progressive** (etapa mid-fi), el agente `designer` invoca `wireframe-renderer` como sub-agente en vez de generar el HTML inline. El renderer escribe los archivos directamente al disco; el agente recibe solo el manifiesto y lo muestra al usuario.

El agente `designer` debe preguntar al usuario qué opción usar al llegar a fase 6:

> "Llegamos a la fase visual. ¿Cómo querés trabajarla?
>
> 1. **ASCII** — wireframes Markdown low-fi. Más rápido, valida estructura.
> 2. **HTML** — mockups HTML+Tailwind. Sentís el producto. Podés elegir fidelity por pantalla.
> 3. **Progressive** — arrancás ASCII, aprobás estructura, elevás a HTML mid-fi, opcional hi-fi en pantallas críticas. **Recomendado para Medium/Advanced.**"

Una vez elegida la opción, el agente invoca la skill correspondiente y la fase 6 queda resuelta por esa skill.

## Reglas duras

- **No se salta el orden de fases.** Cada una depende de las anteriores.
- **No se cambia de modo a mitad.** Si el usuario quiere upgradear (ej. Basic → Medium), se reinicia limpio y se reusan los outputs ya generados.
- **No se codea UI hasta completar el modo elegido.**

## Convención de artefactos

Todo archivo de output lleva al inicio un **frontmatter de identidad** — parte del contrato de integración entre packs (ver `pack.md` → `produces` / `consumes`):

```yaml
---
pack: design
artifact: tech-decisions
scope: frontend
---
```

- `artifact` identifica el área (`discovery`, `information-architecture`, `user-flows`, `component-inventory`, `design-system-tokens`, `wireframes`, `accessibility-plan`, `performance-budget`, `tech-decisions`). En el `design.md` consolidado del modo Basic, `artifact: design-spec`.
- El artefacto `tech-decisions` agrega `scope: frontend | fullstack`.
- Un pack consumidor introspecciona estos campos — **nunca depende del nombre del archivo**.

## Convención de numeración

Los templates de las skills escriben su archivo con un prefijo placeholder `NN-`. **El número lo asigna este workflow**, no la skill — las skills se reutilizan en otros packs (ej. `fullstack-design`) donde el orden de ejecución es distinto, así que no pueden saber su número.

Regla: `NN` = posición de dos dígitos de la fase **en la tabla del modo elegido** (`01`, `02`, …). Las tablas de *Modo Medium* y *Modo Advanced* de arriba son la fuente de verdad. La numeración es **contigua, sin saltos**: si una fase no corre, las siguientes se corren para no dejar huecos.

## Estado del workflow

El agente mantiene el estado en `docs/design/.workflow-state.json`:

```json
{
  "mode": "medium",
  "started_at": "2026-05-19",
  "status": "in-progress",
  "completed_phases": ["discovery", "information-architecture"],
  "current_phase": "user-flows",
  "next_gate": "after-user-flows",
  "last_updated": "2026-05-19T14:32:00Z"
}
```

Si se reinicia la sesión, el agente lee este archivo para retomar:

- `status: "completed"` → ofrecer "revisar / actualizar" en vez de rehacer.
- `status: "in-progress"` → retomar desde `current_phase`.

### Reglas de actualización del estado

- `status` arranca en `"in-progress"` y pasa a `"completed"` cuando se aprueba la última fase del modo elegido.
- **Apenas el usuario aprueba** una fase (gate cumplido), el agente debe **inmediatamente**:
  1. Agregar la fase a `completed_phases`.
  2. Actualizar `last_updated`.
  3. **Después** mover `current_phase` a la siguiente fase.
- **No marcar una fase como completed al arrancar la siguiente.** El estado debe reflejar el momento exacto de la aprobación, no diferirse.
- Si el usuario pide ajustes en una fase ya aprobada, **removerla** de `completed_phases` mientras se ajusta y volver a agregarla al re-aprobar.

### Cierre del workflow

Al aprobar la última fase, el agente deja el state con `status: "completed"`, `current_phase: null` y `last_updated` real. El archivo **se conserva** — registro de procedencia y detección inequívoca de "ya completado" en futuras ejecuciones.

## Excepciones

Si el usuario pide explícitamente saltarse el workflow (ej. prototipo de 30 minutos), permitirlo solo si:

1. El pedido es explícito.
2. Se documenta en `docs/design/00-fast-track.md` qué se saltó y por qué.
