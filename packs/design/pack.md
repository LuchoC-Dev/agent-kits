---
id: design
version: "0.1.0"
description: Pack de diseño pre-build. Orquesta el proceso completo de diseño (discovery, IA, flows, componentes, tokens, wireframes, a11y, performance, tech decisions) antes de implementar. Aplica a frontend, mobile y productos digitales en general.

stack_hints: []

produces:
  - artifact: design-spec
    path: docs/design/
    description: Diseño pre-build — discovery, IA, flows, componentes, tokens, visual, tech-decisions.
  - artifact: tech-decisions
    path: docs/design/*tech-decisions*.md
    description: Stack técnico justificado. Lleva en su frontmatter `scope: frontend | fullstack`.

consumes:
  - artifact: product-context
    from: context
    required: false
    description: Si existe, discovery lo usa como insumo del Intake en vez de preguntar desde cero.

structure_extras:
  - skills
  - agents
  - workflows

skills:
  - id: discovery
    description: Fase 1 — producto, usuarios, JTBD, métricas. Arranca con un intake adaptable al contexto.

  - id: information-architecture
    description: Fase 2 — sitemap, rutas, modelo de navegación

  - id: user-flows
    description: Fase 3 — caminos del usuario paso a paso

  - id: component-inventory
    description: Fase 4 — atoms, molecules, organisms

  - id: design-system-tokens
    description: Fase 5 — colores, tipografía, spacing, breakpoints

  - id: wireframes
    description: Fase 6 (opción ASCII) — wireframes low-fi por pantalla y estado

  - id: html-mockup
    description: Fase 6 (opción HTML) — mockups HTML+Tailwind con fidelity por pantalla

  - id: progressive-design
    description: Fase 6 (opción Progressive) — orquesta wireframes → html-mockup con loops de ajuste

  - id: accessibility-plan
    description: Fase 7 — contrato WCAG AA antes de implementar

  - id: performance-budget
    description: Fase 8 — Core Web Vitals y bundle budgets

  - id: tech-decisions
    description: Fase 9 — stack técnico justificado

agents:
  - id: designer
    description: Diseñador que orquesta el workflow pre-build-design en cualquiera de sus 3 modos (Basic/Medium/Advanced)

workflows:
  - id: pre-build-design
    description: Workflow de diseño pre-build con 3 modos de scope — Basic (3 fases), Medium (7 fases), Advanced (9 fases)
---

# Design Pack

Pack de **diseño pre-build**. Su propósito: dejar toda la base de diseño documentada y aprobada **antes** de escribir código.

## Filosofía

> Diseñar antes de codear. Siempre.

El pack provee un workflow único (`pre-build-design`) con **3 modos de scope**, todos basados en las mismas 9 skills. La diferencia entre modos es **qué subset de fases se ejecuta** y **qué tan profunda es cada una**.

## Modos del workflow

| Modo | Para qué | Fases | Output |
|---|---|---|---|
| **Basic** | Landing, MVP, prototipo, app chica | discovery (light) + wireframes + tech-decisions | 1 archivo: `docs/design/design.md` |
| **Medium** | App de tamaño medio, complejidad normal | discovery + IA + flows + inventory + tokens + wireframes + tech-decisions | Archivos separados, sin a11y/perf como fases (inline) |
| **Advanced** | App grande, sistema crítico, equipo grande | las 9 fases completas | 9 archivos numerados en `docs/design/` |

El usuario elige el modo al iniciar el workflow. El agente puede recomendarlo a partir del contexto pero la decisión es del usuario.

## Skills (las 9 fases)

| Skill | Fase |
|---|---|
| `discovery` | 1 — Producto, usuarios, JTBD |
| `information-architecture` | 2 — Sitemap y navegación |
| `user-flows` | 3 — Caminos del usuario |
| `component-inventory` | 4 — Atomic design |
| `design-system-tokens` | 5 — Lenguaje visual |
| `wireframes` / `html-mockup` / `progressive-design` | 6 — Visual design (selector) |
| `accessibility-plan` | 7 — Contrato WCAG |
| `performance-budget` | 8 — CWV y bundle |
| `tech-decisions` | 9 — Stack justificado |

Cada skill acepta un parámetro `depth: light | full` para ajustar profundidad según el modo del workflow.

## Agente

- **designer** — orquesta el workflow completo, pregunta el modo, encadena las skills, aplica los gates de aprobación.

## Contrato de integración

Este pack participa del contrato declarativo entre packs (ver `produces` / `consumes` en el frontmatter):

- **Consume** `product-context` (del pack `context`, opcional). El **paso 0 (Intake)** de `discovery` lo detecta y lo usa como insumo en vez de pedir el contexto desde cero.
- **Produce** `design-spec` en `docs/design/`, y dentro de él `tech-decisions`. Cada archivo de output lleva frontmatter de identidad (`pack: design`, `artifact: <área>`).
- El artefacto `tech-decisions` lleva además `scope: frontend | fullstack`. Un pack de backend-design posterior lee ese `scope`: si es `fullstack`, salta su propia fase de stack; si es `frontend`, la ejecuta.

## Gates de aprobación

- **Basic**: aprobación al final (un solo documento).
- **Medium**: aprobación cada 2-3 fases.
- **Advanced**: aprobación por fase.
