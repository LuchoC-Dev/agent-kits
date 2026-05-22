---
name: sdd
description: Disciplina Spec-Driven Development — la spec (intent, scope, requirements, design, tasks) es la fuente de verdad y antecede a la implementación. Delegada al ecosistema externo `sdd-flow` y sus sub-skills (sdd-init, sdd-explore, sdd-propose, sdd-spec, sdd-design, sdd-tasks, sdd-apply, sdd-verify, sdd-archive). Trigger cuando el proyecto tiene `sdd` en las disciplinas activas, o el usuario pide "SDD", "spec-driven", "spec primero", "OpenSpec", "iniciar sdd", "sdd init", o quiere proponer/diseñar/implementar un cambio bajo el flujo de specs.
discipline: true
combinable: true
delegates_to: sdd-flow
---

# SDD — Spec-Driven Development

## Rol

Sos un practicante de spec-driven development. Tu trabajo es asegurar que **toda iniciativa no trivial empiece por una spec versionada** —intent, scope, requirements, design, tasks— y que esa spec sea la fuente de verdad contra la que se implementa y se verifica.

## Naturaleza de esta skill

Es una **skill de disciplina** y un **puente**: no contiene el procedimiento SDD acá. El procedimiento ya vive maduro en el ecosistema externo de skills `sdd-*` (especialmente `sdd-flow` como orquestador). Esta skill:

1. Registra SDD como una disciplina activa del workspace.
2. Le indica al workflow / agente cuándo y cómo invocar el ecosistema externo.

Es agnóstica al flujo: el workflow del pack decide en qué momento dispararla.

## Cuándo se activa

Se aplica cuando `sdd` está en el campo `disciplines` de `workspace.json`. El workflow la invoca antes de cualquier cambio no trivial: nada se implementa sin spec.

## La disciplina

### Paso 1 — Inicialización (una sola vez)

Al activar SDD en un proyecto por primera vez, invocá la skill externa `sdd-init` para bootstrappear el contexto SDD (detección de stack, convenciones, backend de persistencia de specs).

### Paso 2 — Para cada cambio, ejecutá el ciclo SDD

Delegá al orquestador externo `sdd-flow`, que conduce el ciclo completo:

| Fase | Skill externa | Output |
|---|---|---|
| Explorar | `sdd-explore` | Investigación previa, clarificación |
| Proponer | `sdd-propose` | Proposal: intent, scope, approach |
| Especificar | `sdd-spec` | Requirements + scenarios (delta specs) |
| Diseñar | `sdd-design` | Decisiones técnicas, ADRs |
| Romper en tareas | `sdd-tasks` | Checklist ejecutable |
| Implementar | `sdd-apply` | Código que cumple la spec |
| Verificar | `sdd-verify` | Validación spec ↔ código |
| Archivar | `sdd-archive` | Sync de delta specs a specs principales |

No reimplementes ninguno de esos pasos acá. Invocá la skill externa correspondiente.

### Paso 3 — Mantener la spec sincronizada

- La spec antecede al código. Si la implementación descubre algo que la invalida, **se actualiza la spec primero**.
- Las delta specs viven con el cambio y se mergean a las specs principales en el archivo (paso 8, `sdd-archive`).

## Reglas de la disciplina

- **Nada no trivial sin spec.** Bugfixes triviales y typos pueden saltarse SDD; features, refactors no triviales y cambios de comportamiento, no.
- **La spec es la fuente de verdad.** Ante discrepancia, se reconcilia la spec o el código deliberadamente — nunca silenciosamente.
- **Una spec por cambio.** No mezclar múltiples intents en una sola spec.
- **Verificación obligatoria.** `sdd-verify` antes de archivar.

## Integración con otras disciplinas

SDD es combinable con todas:

- **Con `tdd`** — los scenarios de la spec se traducen en tests rojos antes de implementar. El ciclo red→green→refactor ocurre dentro de `sdd-apply`.
- **Con `bdd`** — los Given/When/Then enriquecen los scenarios de la spec. BDD aporta la forma legible; SDD aporta la estructura versionada.
- **Con `contract-first`** — el contrato formal de la interfaz vive como parte del `sdd-design` o como artefacto adyacente referenciado por la spec.
- **Con `trunk-based`** — cada delta spec corresponde a una rama corta; al mergear se archiva la delta.

## Reglas duras

- No implementar cambios no triviales sin pasar por `sdd-propose` → `sdd-spec` → `sdd-design` → `sdd-tasks` antes de `sdd-apply`.
- No omitir `sdd-verify`: el archivado requiere verificación previa.
- No reimplementar el ciclo SDD acá — siempre delegar a las skills externas `sdd-*`.

## Nota de implementación

Las skills `sdd-init`, `sdd-flow`, `sdd-explore`, `sdd-propose`, `sdd-spec`, `sdd-design`, `sdd-tasks`, `sdd-apply`, `sdd-verify`, `sdd-archive` y `spec-driven-development` viven en el entorno externo (no en el catálogo de `app-init`). Esta disciplina solo las registra y delega; no las redefine.
