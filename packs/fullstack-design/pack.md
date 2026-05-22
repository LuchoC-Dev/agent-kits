---
id: fullstack-design
version: "0.1.0"
description: Pack orquestador de diseño fullstack pre-build. Compone las skills de `design` y `backend-design` en un solo workflow coordinado con 3 tiers (Lite/Standard/Full) y 2 estrategias de flujo (Separado/Integrado). No instala los packs completos — referencia solo las skills que orquesta, con un único agente y un único workflow.

stack_hints: []

produces:
  - artifact: tech-decisions
    path: docs/shared/
    description: Stack fullstack — artefacto cross-cutting que usan ambos tracks.
  - artifact: design-spec
    path: docs/design/
    description: Track de diseño frontend — mismo artefacto que produce el pack `design`.
  - artifact: backend-design-spec
    path: docs/backend-design/
    description: Track de diseño backend — mismo artefacto que produce el pack `backend-design`.

consumes:
  - artifact: product-context
    from: context
    required: false
    description: Insumo compartido por ambos tracks (discovery, dominio, jobs).

structure_extras:
  - skills
  - agents
  - workflows

skills:
  # --- Compartidas (corren una vez para los dos tracks) ---
  - id: discovery
    description: Fase compartida — producto, usuarios, JTBD, métricas
  - id: tech-decisions
    description: Stack fullstack — se decide una sola vez (scope: fullstack)

  # --- Track frontend ---
  - id: information-architecture
  - id: user-flows
  - id: component-inventory
  - id: design-system-tokens
  - id: wireframes
  - id: html-mockup
  - id: progressive-design
  - id: accessibility-plan
  - id: performance-budget

  # --- Track backend ---
  - id: api-contract
  - id: data-model
  - id: service-architecture
  - id: code-conventions
  - id: integrations-auth
  - id: nonfunctional
  - id: testing-strategy
  - id: security-design

agents:
  - id: fullstack-designer
    description: Orquesta el workflow fullstack-design componiendo ambos tracks
  - id: cross-track-auditor
    class: 2
    description: Verifica coherencia entre el track frontend y el backend antes del gate de cierre. 6 chequeos cruzados (API↔flujos, data-model↔componentes, tech-decisions↔ambos tracks, auth↔flujos, arquitectura↔IA, no-funcionales↔performance budget).

workflows:
  - id: fullstack-design
    description: Workflow de diseño fullstack pre-build con 3 modos (Lite/Standard/Full)
---

# Fullstack Design Pack

Pack **orquestador** de diseño fullstack pre-build. No define fases nuevas: **compone** las skills de `design` (frontend) y `backend-design` (backend) en un único workflow coordinado.

## El problema que resuelve

Tener `design` y `backend-design` como packs especialistas separados deja dos huecos:

1. **Usar los dos** — sin orquestación, corrés dos workflows, elegís modo dos veces, decidís el stack en dos lados.
2. **Fullstack chico** — no hay un camino right-sized; correr los dos packs completos es overkill.

`fullstack-design` los cierra: **un setup, una decisión de stack, un workflow, modos a medida.**

## Cómo se compone (no instala los packs completos)

Las skills viven en un **pool global** del catálogo (`skills/`), no dentro de ningún pack. Este pack simplemente **lista los `id`** de las skills que orquesta — las mismas que usan `design` y `backend-design`. No declara `depends_on` de los packs completos: el instalador trae solo esas skills, no los agentes ni workflows de los otros packs. Resultado: un solo agente (`fullstack-designer`) y un solo workflow (`fullstack-design`) — un único punto de entrada, sin tres workflows compitiendo.

Las skills compartidas (`discovery`, `tech-decisions`, y a futuro `code-conventions`) son la **misma entrada del pool** — se instalan una sola vez.

## Dos ejes: tier y estrategia de flujo

El workflow se configura con **dos decisiones independientes**.

### Tier — cuánta profundidad

Tier **pareado** — un solo nivel aplica a ambos tracks:

| Tier | Equivale a | Para qué |
|---|---|---|
| **Lite** | design Basic + backend-design Basic | Fullstack chico, MVP, prototipo |
| **Standard** | design Medium + backend-design Medium | Fullstack de complejidad normal |
| **Full** | design Advanced + backend-design Advanced | Sistema grande, equipo grande |

### Estrategia de flujo — cómo se entrelazan los tracks

| Estrategia | Cómo corre | Cuándo |
|---|---|---|
| **Separado** | Un track completo y después el otro (orden elegible: back o front primero). | Equipos separados, o cuando una mitad cierra antes de tocar la otra. |
| **Integrado** | Fases en pares backend ⇄ frontend que se informan mutuamente. | Cuando contrato de API y flujos de UI se benefician de diseñarse en diálogo. **Default recomendado.** |

## Estructura del workflow

```
Setup único:
  detectar product-context  →  elegir tier  →  elegir estrategia de flujo  →  evaluar seguridad

Fase compartida:   tech-decisions (scope fullstack)            → docs/shared/
Separado:          un track completo → el otro track completo
Integrado:         pares backend ⇄ frontend, gate por par

Track backend:     api-contract → data-model → service-architecture → code-conventions
                   → integrations-auth → [nonfunctional] → [testing] → [security]
Track frontend:    information-architecture → user-flows → component-inventory
                   → design-system-tokens → visual-design → [accessibility] → [performance]
```

`tech-decisions` corre primero con `scope: fullstack` — decide el stack una sola vez y satisface el Paso 0 del track backend.

## Contrato de integración

- **Consume** `product-context` (de `context`, opcional).
- **Produce** `tech-decisions`, `design-spec` y `backend-design-spec` — los **mismos artefactos** que producen los packs especialistas. Los packs de implementación (`frontend`, `backend`) los consumen sin cambios, identificándolos por el frontmatter `artifact:`.

## Output

Layout **igual en las dos estrategias** (la estrategia solo cambia el orden de generación):

```
docs/
├── shared/             ← tech-decisions (cross-cutting)
├── backend-design/     ← track backend
├── design/             ← track frontend
└── design-index.md     ← índice generado al cierre
```

- **Lite** → `docs/shared/01-tech-decisions.md` + `docs/backend-design/backend-design.md` + `docs/design/design.md` (consolidado por lado).
- **Standard / Full** → archivos numerados en las tres carpetas. La numeración es **por carpeta**, contigua (`01..N`), asignada por el workflow.

## Cuándo usar este pack vs los especialistas

- **Solo frontend** → pack `design`.
- **Solo backend** → pack `backend-design`.
- **Fullstack** (los dos lados) → `fullstack-design`.
