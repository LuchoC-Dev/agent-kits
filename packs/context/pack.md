---
id: context
version: "0.3.0"
description: Pack de construcción de contexto pre-diseño. 6 áreas (problem framing, stakeholders, domain model, job stories, vision, assumptions & risks) con profundidad y cadencia de gate configurables. Genera docs/context/ que alimenta al pack `design`. Las metodologías de implementación (BDD/SDD/TDD) viven en otro pack — este es puro contexto.

stack_hints: []

produces:
  - artifact: product-context
    path: docs/context/
    description: Contexto del producto — 6 áreas, en 6 archivos canónicos o un context.md consolidado.

consumes: []

structure_extras:
  - skills
  - agents
  - workflows

skills:
  - id: problem-framing
    description: Fase 1 — definir el problema real (5 whys, problema vs solución)

  - id: stakeholders-map
    description: Fase 2 — actores con interés, matriz poder × interés

  - id: domain-modeling
    description: Fase 3 — entidades, relaciones, eventos, ubiquitous language (DDD ligero)

  - id: job-stories
    description: Fase 4 — jobs en formato "cuando ___, quiero ___, así puedo ___"

  - id: vision
    description: Fase 5 — vision statement, north star metric, anti-visión

  - id: assumptions-risks
    description: Fase 6 — apuestas, leaps of faith, riesgos a mitigar

agents:
  - id: context-builder
    description: Orquesta el workflow context-building (6 áreas, profundidad y gate configurables)

workflows:
  - id: context-building
    description: Workflow de 6 áreas con dos ejes — profundidad (rápido/completo/profundo) y gate (batch/por-fase)
---

# Context Pack

Pack de **construcción de contexto** previo al diseño y la implementación. Aplica el principio: *lo que no se articula, se asume; lo que se asume, se construye mal.*

## Para qué sirve

- Producto nuevo desde una idea poco articulada
- Refundación de producto cuando el problema original se perdió
- Onboarding de equipos nuevos
- Pre-discovery para alimentar el pack `design` con material rico

## Áreas (workflow `context-building`)

| # | Skill | Qué define |
|---|---|---|
| 1 | `problem-framing` | El problema real (no la solución imaginada) |
| 2 | `stakeholders-map` | Quién tiene interés, poder y postura |
| 3 | `domain-modeling` | Entidades, eventos, ubiquitous language |
| 4 | `job-stories` | Trabajos que el usuario contrata al producto |
| 5 | `vision` | North star, vision statement, anti-visión |
| 6 | `assumptions-risks` | Apuestas y amenazas explícitas |

## Modos (dos ejes configurables)

El workflow se configura con dos ejes independientes, preguntados al inicio:

- **Profundidad** — `rápido` (3 fases fusionadas, 1 pregunta clave c/u) · `completo` (6 fases, default) · `profundo` (6 fases + sub-análisis y repreguntas).
- **Gate** — `batch` (genera todo y pide una sola verificación al final) · `por-fase` (gate de aprobación tras cada fase).

Default sugerido: `completo` + `batch`. Cualquier combinación es válida.

El **formato de archivos es adaptativo**: en `batch`, el agente evalúa la riqueza del contexto extraído y decide entre un `context.md` consolidado (poco contexto) o documentos separados (mucho contexto).

## Qué NO está en este pack

Las metodologías de **implementación** (BDD, SDD, TDD) viven en un pack separado (futuro `implementation/`). Este pack es **solo contexto** — el qué y el por qué. Cómo construir, formalizar specs ejecutables o aplicar test-first es responsabilidad de otro pack.

## Integración con `design`

Si este pack se completa antes de `design`, el `discovery` (fase 1 de design) detecta `docs/context/` y lo usa como insumo del Paso 0 (Intake). El usuario no tiene que re-articular nada.

## Output

El formato depende de la riqueza del contexto extraído:

```
# Mucho contexto → documentos separados
docs/context/
├── 01-problem-framing.md
├── 02-stakeholders.md
├── 03-domain-model.md
├── 04-job-stories.md
├── 05-vision.md
└── 06-assumptions-risks.md

# Poco contexto → un solo archivo consolidado
docs/context/
└── context.md
```

## Contrato de integración

Este pack participa del contrato declarativo entre packs (ver `produces` / `consumes` en el frontmatter):

- **Produce** `product-context` en `docs/context/`. Cada archivo de output lleva un frontmatter de identidad (`pack: context`, `artifact: <área>`) para que cualquier consumidor lo introspeccione sin depender del nombre del archivo.
- **No consume** nada — es el primer eslabón de la cadena.

El pack `design` declara `product-context` como input opcional: su `discovery` lo detecta y lo usa automáticamente.

## Dependencias

Ninguna. Pack standalone.

## Recomendación de uso

```
context  →  design  →  frontend/backend
```

El orden importa: contexto antes que diseño, diseño antes que implementación.
