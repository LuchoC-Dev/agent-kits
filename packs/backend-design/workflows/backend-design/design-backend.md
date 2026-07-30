---
id: backend-design
description: Workflow de diseño técnico de backend pre-build con 3 modos de scope (Basic, Medium, Advanced). Selecciona el subset de fases adecuado al tamaño del sistema y produce documentación en docs/backend-design/.

agent: backend-designer

modes:
  basic:
    description: Para CRUD simple, 1 servicio, MVP de API
    steps:
      - { skill: api-contract, depth: light }
      - { skill: data-model, depth: light }
      - { skill: service-architecture, depth: light }
    output: docs/backend-design/backend-design.md
    gate: approval-at-end
    notes: |
      code-conventions no es fase: se incluyen 2-3 reglas baseline inline
      (guard clauses, manejo de errores). nonfunctional y testing se omiten.

  medium:
    description: Para API con varios dominios y complejidad normal
    steps:
      - { skill: api-contract, depth: full }
      - { skill: data-model, depth: full }
      - { skill: service-architecture, depth: full }
      - { skill: code-conventions, depth: full }
      - { skill: integrations-auth, depth: full }
    output: docs/backend-design/NN-*.md
    gate: approval-per-block
    notes: |
      no-funcionales (escalabilidad, caching, observabilidad) se incluyen inline
      en service-architecture e integrations-auth, no como fase separada.

  advanced:
    description: Para sistemas distribuidos, microservicios, equipos grandes
    steps:
      - { skill: api-contract, depth: full }
      - { skill: data-model, depth: full }
      - { skill: service-architecture, depth: full }
      - { skill: code-conventions, depth: full }
      - { skill: integrations-auth, depth: full }
      - { skill: nonfunctional, depth: full }
      - { skill: testing-strategy, depth: full }
      - { skill: security-design, depth: full, optional: true }
    output: docs/backend-design/NN-*.md
    gate: approval-per-phase
---

# Backend Design Workflow

Workflow de **diseño técnico de backend** que se adapta al tamaño del sistema. Una sola fuente de verdad, tres experiencias.

## Filosofía

> Diseñar el backend antes de codear. Adaptar la profundidad al tamaño del sistema.

Aplicar 7 fases a un CRUD es overkill. Aplicar 2 fases a un sistema distribuido es irresponsable. El workflow te deja elegir.

## Inicio del workflow

Antes de arrancar, el agente `backend-designer` hace **cuatro cosas**:

### 1. Detectar diseño previo

Lee `docs/backend-design/.workflow-state.json` si existe:

- `status: "completed"` → ofrecer "revisar / actualizar" en vez de rehacer.
- `status: "in-progress"` → retomar desde `current_phase`.
- Sin state pero con archivos sueltos → inferir qué hay y proponer completar.
- Nada → arrancar limpio.

### 2. Detectar contexto previo

- ¿Existe `product-context` (pack `context`)? Detectalo por el frontmatter de identidad (`pack: context`), en `docs/context/` → `context/` → `docs/`. Si está, usá `domain-model` como base de las fases `data-model` y `service-architecture`, y `job-stories` para derivar endpoints en `api-contract`.
- ¿No hay contexto? Cada fase que lo necesita arranca con un mini-intake propio.

### 3. Resolver el stack — Paso 0

El backend necesita un stack decidido (runtime/lenguaje, framework de API, motor de BD, ORM/query layer, librería de auth) **antes** de la fase 1.

- **¿Existe `tech-decisions` con `scope: fullstack`?** → el stack del backend ya está decidido. Buscalo por su frontmatter de identidad en `docs/shared/`, `docs/design/` o `docs/`. Leelo, mostralo al usuario para confirmar, y **omití el Paso 0**.
- **¿`scope: frontend` o no existe `tech-decisions`?** → ejecutá el Paso 0: preguntá y decidí el stack del backend con criterio (no por hype). Justificá cada elección.

El resultado del Paso 0 se escribe en `00-stack.md` (modos Medium/Advanced) o como sección "Stack" al inicio del documento (modo Basic), y se registra en el estado del workflow.

### 4. Seleccionar modo

| Modo | Cuándo recomendarlo |
|---|---|
| **Basic** | CRUD simple, un solo servicio, MVP de API, herramienta interna |
| **Medium** | API con varios dominios, login + recursos, complejidad normal |
| **Advanced** | Sistema distribuido, microservicios, alto volumen, equipo grande |

El agente recomienda; el usuario decide.

### 5. Evaluar la fase opcional de seguridad

Preguntá:

> "¿El sistema maneja datos sensibles o personales, está regulado (fintech, salud, etc.), o tiene alta superficie de ataque?"

- **Sí** → se agrega la **fase 8 `security-design`** (deep-dive de seguridad). Disponible en Medium y Advanced.
- **No** → se omite; la base de `integrations-auth` (fase 5) alcanza.
- En modo **Basic**, si la respuesta es "sí", recomendá subir a Medium o Advanced — un sistema con seguridad pesada no es un proyecto Basic.

## Modo Basic

3 fases en versión light → 1 documento consolidado.

| # | Skill | Depth |
|---|---|---|
| 1 | `api-contract` | light |
| 2 | `data-model` | light |
| 3 | `service-architecture` | light (mini) |

**code-conventions:** no es fase. Se incluyen 2-3 reglas baseline inline (guard clauses, manejo de errores consistente, naming).

**Output:** `docs/backend-design/backend-design.md` con las secciones (no archivos separados).

**Gate:** se entrega todo de corrido y se aprueba al final.

## Modo Medium

5 fases, archivos separados, gates cada 2-3 fases. Las tablas de abajo son la **fuente de verdad de los `NN`** — las skills escriben con placeholder `NN-` y este workflow asigna el número.

| NN | Skill | Output |
|---|---|---|
| 01 | `api-contract` | `01-api-contract.md` |
| 02 | `data-model` | `02-data-model.md` |
| 03 | `service-architecture` | `03-service-architecture.md` |
| 04 | `code-conventions` | `04-code-conventions.md` |
| 05 | `integrations-auth` | `05-integrations-auth.md` |

**No-funcionales:** integrados como secciones dentro de `service-architecture` e `integrations-auth` (no como fase separada).

**Gates de aprobación:** después de fase 2, fase 4 y fase 5.

## Modo Advanced

7 fases completas (+ 1 opcional), archivos separados, gate por fase.

| NN | Skill | Output |
|---|---|---|
| 01 | `api-contract` | `01-api-contract.md` |
| 02 | `data-model` | `02-data-model.md` |
| 03 | `service-architecture` | `03-service-architecture.md` |
| 04 | `code-conventions` | `04-code-conventions.md` |
| 05 | `integrations-auth` | `05-integrations-auth.md` |
| 06 | `nonfunctional` | `06-nonfunctional.md` |
| 07 | `testing-strategy` | `07-testing-strategy.md` |
| 08 | `security-design` *(opcional)* | `08-security-design.md` |

La fase 8 entra solo si la evaluación de seguridad (paso 5 del inicio) dio positiva.

**Gates:** aprobación explícita después de cada fase.

## Fase 8 — Security Design (opcional)

`security-design` es una fase **opcional** que no depende del modo sino de la **evaluación de seguridad** del inicio. Cubre el deep-dive que excede a la auth básica: threat modeling, hardening, protección de datos, audit logging, compliance y gestión de vulnerabilidades.

- **Medium** — si la evaluación dio positiva, se agrega como fase final (después de `integrations-auth`).
- **Advanced** — entra como fase 8, después de `testing-strategy`.
- Si no se justifica, se omite — `integrations-auth` (fase 5) es la base de seguridad para el resto de los casos.

## Convención de artefactos

Todo archivo de output lleva al inicio un **frontmatter de identidad** — parte del contrato de integración entre packs (ver `pack.md` → `produces` / `consumes`):

```yaml
---
pack: backend-design
artifact: api-contract
---
```

- `artifact` identifica la fase (`stack`, `api-contract`, `data-model`, `service-architecture`, `code-conventions`, `integrations-auth`, `nonfunctional`, `testing-strategy`, `security-design`). En el `backend-design.md` consolidado del modo Basic, `artifact: backend-design-spec`.
- Un pack consumidor (ej. el pack `backend` de implementación) introspecciona el campo `artifact` — **nunca depende del nombre del archivo**.

## Convención de numeración

Los templates de las skills escriben su archivo con un prefijo placeholder `NN-`. **El número lo asigna este workflow**, no la skill — las skills se reutilizan en otros packs (ej. `fullstack-design`) donde el orden de ejecución es distinto, así que no pueden saber su número.

Regla: `NN` = posición de dos dígitos de la fase **en la tabla del modo elegido** (`01`, `02`, …). Las tablas de *Modo Medium* y *Modo Advanced* de arriba son la fuente de verdad. La numeración es **contigua, sin saltos**. El `00-stack.md` del Paso 0 queda como `00` por ser previo a la fase 1.

## Reglas duras

- **No se salta el orden de fases.** Cada una depende de las anteriores.
- **No se cambia de modo a mitad.** Si el usuario quiere upgradear (ej. Basic → Medium), se reinicia limpio y se reusan los outputs ya generados.
- **No se decide el stack dos veces.** Si `tech-decisions` ya lo definió (`scope: fullstack`), se respeta — no se vuelve a preguntar.
- **No se codea backend hasta completar el modo elegido.**
- **No se inventa el dominio.** Si existe `domain-model`, es la base — no se redefine.

## Estado del workflow

El agente mantiene el estado en `docs/backend-design/.workflow-state.json`:

```json
{
  "mode": "medium",
  "started_at": "2026-05-20",
  "status": "in-progress",
  "stack_source": "paso-0",
  "completed_phases": ["api-contract", "data-model"],
  "current_phase": "service-architecture",
  "next_gate": "after-code-conventions",
  "last_updated": "2026-05-20T16:10:00Z"
}
```

- `status` arranca en `"in-progress"` y pasa a `"completed"` al aprobar la última fase del modo.
- `stack_source` es `"paso-0"` (lo decidió este workflow) o `"tech-decisions"` (lo heredó de `design`).
- **Apenas el usuario aprueba** una fase: agregar a `completed_phases`, actualizar `last_updated`, y **después** mover `current_phase`.
- **No marcar una fase como completed al arrancar la siguiente.**
- Si el usuario pide ajustes en una fase ya aprobada, **removerla** de `completed_phases` mientras se ajusta y volver a agregarla al re-aprobar.
- `started_at` y `last_updated` usan **fecha y hora reales** — nunca placeholders.

### Cierre del workflow

Al aprobar la última fase, el agente deja el state con `status: "completed"`, `current_phase: null` y `last_updated` real. El archivo **se conserva** — registro de procedencia y detección inequívoca de "ya completado".

## Qué viene después

Una vez completado, el pack `backend` (implementación) consume `backend-design-spec` como base. La conversión a specs ejecutables (BDD/SDD) y el test-first (TDD) son responsabilidad del futuro pack `implementation/`.
