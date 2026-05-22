---
id: context-building
description: Workflow de construcción de contexto pre-diseño. Dos ejes configurables — profundidad (rápido/completo/profundo) y cadencia de gate (batch/por fase). Resultado, docs/context/ con material que alimenta el discovery del pack design.

agent: context-builder

depth_levels: [rapido, completo, profundo]
gate_modes: [batch, por-fase]

phases:
  - { skill: problem-framing, nn: "01", output: docs/context/NN-problem-framing.md, depends_on: [] }
  - { skill: stakeholders-map, nn: "02", output: docs/context/NN-stakeholders.md, depends_on: [problem-framing] }
  - { skill: domain-modeling, nn: "03", output: docs/context/NN-domain-model.md, depends_on: [stakeholders-map] }
  - { skill: job-stories, nn: "04", output: docs/context/NN-job-stories.md, depends_on: [problem-framing, stakeholders-map, domain-modeling] }
  - { skill: vision, nn: "05", output: docs/context/NN-vision.md, depends_on: [job-stories] }
  - { skill: assumptions-risks, nn: "06", output: docs/context/NN-assumptions-risks.md, depends_on: [vision] }

rapido_phases:
  - id: problema-actores
    skills: [problem-framing, stakeholders-map]
  - id: dominio-jobs
    skills: [domain-modeling, job-stories]
  - id: vision-riesgos
    skills: [vision, assumptions-risks]
---

# Context Building Workflow

Workflow para construir **contexto del producto antes de cualquier diseño**.

## Filosofía

> Lo que no se articula, se asume. Lo que se asume, se construye mal.

Este workflow obliga a explicitar lo implícito. Después de pasarlo, cualquier persona (humana o IA) puede arrancar a diseñar o implementar con la misma base.

## Configuración — dos ejes

Antes de arrancar, el agente `context-builder` pregunta **dos cosas** (juntas, con `AskUserQuestion` si está disponible):

### Eje 1 — Profundidad

| Nivel | Fases | Preguntas | Para |
|---|---|---|---|
| **`rápido`** | 3 fases fusionadas | hasta 1 pregunta clave por fase | Idea simple, validación rápida, prototipo |
| **`completo`** | 6 fases | preguntas normales | Producto real — el caso por defecto |
| **`profundo`** | 6 fases + sub-análisis | repreguntas y validación | Producto complejo, alto riesgo, equipo grande |

En `rápido` las 6 áreas se cubren **igual** — la fusión agrupa la **interacción** (3 rondas de preguntas en vez de 6), no la estructura de salida:

| Fase rápida | Skills (depth `quick`) | Cubre áreas |
|---|---|---|
| Problema + Actores | `problem-framing` + `stakeholders-map` | 1-2 |
| Dominio + Jobs | `domain-modeling` + `job-stories` | 3-4 |
| Visión + Riesgos | `vision` + `assumptions-risks` | 5-6 |

> **Importante:** fusionar es solo de la conversación. Cada skill, aunque corra dentro de una fase fusionada, **produce su propia área**. La estructura de archivos de salida es siempre la canónica (ver "Formato de archivos") — nunca hay "archivos fusionados".

El nivel se traduce al parámetro `depth` de cada skill:

| Profundidad | `depth` de la skill |
|---|---|
| `rápido` | `quick` |
| `completo` | `full` |
| `profundo` | `deep` |

### Eje 2 — Cadencia de gate

| Modo | Comportamiento |
|---|---|
| **`batch`** | El agente genera **todo el contenido** sin frenar y pide **una sola verificación al final**. Ideal cuando las fases de síntesis no requieren input adicional. |
| **`por-fase`** | Gate de aprobación al final de cada fase — no avanza sin OK. Ideal para `profundo` o cuando el usuario quiere iterar sobre la marcha. |

Default sugerido: **`completo` + `batch`**. Los ejes son independientes — cualquier combinación es válida (ej. `profundo` + `por-fase`, `rápido` + `batch`).

## Formato de archivos — adaptativo

Hay **solo dos formatos posibles de salida**, sin importar la profundidad elegida:

- **Separado** — los 6 archivos canónicos por área: `01-problem-framing.md`, `02-stakeholders.md`, `03-domain-model.md`, `04-job-stories.md`, `05-vision.md`, `06-assumptions-risks.md` (los `NN` son los de la tabla de fases de este workflow).
- **Consolidado** — un único `docs/context/context.md` con las 6 secciones.

Los nombres canónicos son **siempre los mismos** — también en `rápido` (la fusión es de la interacción, no de los archivos). Esto garantiza que el pack `design` siempre sepa qué buscar.

Qué formato se usa **no lo decide el modo** — lo decide la **riqueza del contexto extraído**:

- **Gate `por-fase`** → cada área aprobada escribe su archivo canónico a medida que avanza. En `rápido`, una fase fusionada aprobada escribe los 2 archivos de sus 2 áreas. Resultado: los 6 archivos separados.
- **Gate `batch`** → el agente genera todo y, al cerrar, **evalúa cuánto contexto real se extrajo**:
  - **Poco** (respuestas breves, producto simple, sin matices) → **consolidado** (`context.md`).
  - **Mucho** (respuestas ricas, varios stakeholders/entidades/jobs, matices) → **separado** (los 6 archivos).

El agente **anuncia su decisión y la justifica**: *"el contexto extraído es acotado, lo consolido en un solo `context.md`"* / *"hay material rico, lo separo en los 6 documentos"*.

## Convención de artefactos

Todo archivo de output lleva al inicio un **frontmatter de identidad** — es parte del contrato de integración entre packs (ver `pack.md` → `produces` / `consumes`):

```yaml
---
pack: context
artifact: problem-framing
---
```

- `artifact` es el área: `problem-framing`, `stakeholders`, `domain-model`, `job-stories`, `vision`, `assumptions-risks`.
- En el `context.md` consolidado, el frontmatter usa `artifact: product-context` (el bundle completo).
- Un pack consumidor (ej. `design`) introspecciona el campo `artifact` — **nunca depende del nombre del archivo**.

## Fases (orden canónico de áreas)

| NN | Área | Skill | Gate `por-fase` |
|---|---|---|---|
| 01 | Problem Framing | `problem-framing` | aprobación |
| 02 | Stakeholders | `stakeholders-map` | aprobación |
| 03 | Domain Model | `domain-modeling` | aprobación |
| 04 | Job Stories | `job-stories` | aprobación |
| 05 | Vision | `vision` | aprobación |
| 06 | Assumptions & Risks | `assumptions-risks` | aprobación |

El **orden de las áreas es siempre el mismo**, incluso en `rápido` (donde se agrupan de a dos).

## Inicio

El agente `context-builder`:

1. Detecta si existe `docs/context/` y lee `.workflow-state.json` si está:
   - `status: "completed"` → ofrecer "revisar / actualizar" en vez de rehacer.
   - `status: "in-progress"` → retomar desde `current_phase`.
   - Sin state pero con archivos sueltos → inferir qué hay y proponer completar.
   - Nada → arrancar limpio.
2. Pide una descripción libre del producto (Paso 0 / Intake).
3. Pregunta los **dos ejes** (profundidad + gate).
4. Ejecuta las fases según la configuración elegida.

## Qué viene después

Este pack se enfoca **solo en contexto**. La conversión a formatos de implementación (BDD/Gherkin, SDD/specs formales, TDD/test-first) es responsabilidad del futuro pack `implementation/`, que puede consumir `docs/context/` como insumo.

## Reglas duras

- **No se salta el orden de las áreas** — ni en `rápido`.
- **En `rápido`, las 6 áreas se cubren igual** — fusionar fases no es omitir áreas.
- **No se inventa contenido.** Cada fase usa material de las anteriores; si en `quick` derivás algo que el usuario no dijo, marcalo **"asumido — confirmar"**.
- **En `batch` igual hay verificación final obligatoria** — `batch` no es "sin gate", es "un gate al final".
- **No se sobreescribe `docs/context/` existente** sin confirmación del usuario.

## Estado del workflow

```json
{
  "started_at": "2026-05-20",
  "status": "in-progress",
  "depth": "completo",
  "gate_mode": "batch",
  "completed_phases": ["problem-framing", "stakeholders-map"],
  "current_phase": "domain-modeling",
  "file_format": null,
  "last_updated": "2026-05-20T14:30:00Z"
}
```

Guardado en `docs/context/.workflow-state.json`.

- `status` arranca en `"in-progress"` y pasa a `"completed"` cuando se cierra el workflow (las 6 áreas aprobadas).
- `depth` y `gate_mode` se fijan al inicio y no cambian durante el workflow.
- `file_format` queda `null` hasta el cierre; ahí se setea a `"consolidado"` o `"separado"`.
- En `por-fase`, la fase se marca `completed` **apenas se aprueba** — no al arrancar la siguiente.
- En `batch`, las fases se agregan a `completed_phases` a medida que se generan; la verificación final confirma todo el conjunto.
- `started_at` y `last_updated` usan **fecha y hora reales** — nunca placeholders.

### Cierre del workflow

Al terminar (las 6 áreas en `completed_phases` y aprobadas), el agente deja el state así:

- `status`: `"completed"`
- `current_phase`: `null`
- `file_format`: `"consolidado"` o `"separado"` (la decisión final)
- `last_updated`: fecha/hora reales del cierre

El archivo **se conserva** — no se borra al terminar. Sirve de registro de procedencia y permite que una re-ejecución detecte inequívocamente que el contexto ya está completo (vía `status: "completed"`) y ofrezca "revisar/actualizar" en vez de rehacer.

## Integración con el pack design

Una vez completado este workflow, el pack `design` puede arrancar:

- El `discovery` (fase 1 de design) detecta automáticamente los archivos de `docs/context/` (sea `context.md` consolidado o los `01-...` separados) y los usa como insumo del Paso 0 (Intake).
- El usuario no tiene que re-articular el problema ni los usuarios — ya está hecho.

Si solo se hicieron algunas fases de context, discovery usa lo disponible y pide el resto.
