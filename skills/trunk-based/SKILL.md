---
name: trunk-based
description: Disciplina de integración trunk-based development — ramas de vida corta, commits chicos, integración frecuente a una sola rama principal siempre desplegable. Trigger cuando el proyecto tiene `trunk-based` en las disciplinas activas, o el usuario pide "trunk-based", "ramas cortas", "integración continua", "commits chicos", o quiere una disciplina de branching.
discipline: true
combinable: true
---

# Trunk-Based Development

## Rol

Sos un practicante de trunk-based development. Tu trabajo es mantener una **única rama principal (`trunk`/`main`) siempre integrable y desplegable**, integrando cambios chicos y frecuentes en lugar de ramas largas que divergen.

## Naturaleza de esta skill

Es una **skill de disciplina**. No produce un documento-artefacto: rige **cómo se versiona e integra el trabajo** — la cadencia de ramas, commits y merges. Es agnóstica al flujo: el workflow decide en qué pasos se aplica (típicamente los de commit y PR).

## Cuándo se activa

Se aplica cuando `trunk-based` está en el campo `disciplines` de `workspace.json`. El workflow la invoca en los pasos de commit e integración. Si no está activa, el branching sigue la convención por defecto del proyecto.

## La disciplina

### Ramas de vida corta

- Una rama por incremento de trabajo, **viva horas o pocos días**, no semanas.
- Se ramifica desde `trunk` actualizado y vuelve a `trunk` rápido.
- Cuanto más vieja una rama, más cara su integración: integrá antes de que diverja.

### Commits chicos y frecuentes

- Cada commit es un incremento coherente y pequeño. Más chico = más fácil de revisar, revertir y bisectar.
- Si una rama acumula muchos commits sin integrar, el incremento era demasiado grande — partilo.

### Trunk siempre verde y desplegable

- `trunk` nunca se rompe: lo que se integra pasó tests.
- Cualquier commit de `trunk` debería poder desplegarse.
- Trabajo incompleto que llega a `trunk` va **detrás de un feature flag**, no en una rama larga.

### Integración frecuente

- Integrá a `trunk` apenas el incremento esté verde — no esperes a "terminar todo".
- Traé `trunk` a tu rama seguido para resolver conflictos chicos y temprano.
- PRs chicos: revisión rápida, merge rápido.

## Reglas de la disciplina

- **Ramas medidas en horas/días, no en semanas.**
- **Commits chicos** — un incremento coherente por commit.
- **`trunk` siempre verde y desplegable.**
- **Trabajo incompleto → feature flag**, nunca una rama de larga vida.
- **Integrar temprano y seguido** — la divergencia es el enemigo.

## Integración con otras disciplinas

Trunk-based es combinable; no es excluyente con ninguna otra:

- **Con `tdd`** — cada ciclo red-green-refactor termina en verde: punto natural de commit chico. El ritmo de TDD alimenta la cadencia de trunk-based.
- **Con `bdd`** — cada escenario satisfecho es un incremento entregable e integrable a `trunk`.
- **Con `contract-first`** — el contrato congelado + mocks dejan que cliente y servidor integren a `trunk` en paralelo sin bloquearse.

## Relación con las skills `git-commit` y `branch-pr`

Trunk-based **no reemplaza** a `git-commit` ni a `branch-pr`: las **gobierna**. Aporta la *política* (tamaño de rama, frecuencia, trunk verde); aquellas aportan la *mecánica* (escribir el commit, abrir el PR). Cuando esta disciplina está activa, `git-commit` y `branch-pr` se ejecutan respetando esta cadencia.

## Reglas duras

- No abrir ramas de larga vida; si una rama se está volviendo larga, partir el trabajo o usar feature flags.
- No integrar a `trunk` algo que no esté verde.
- No acumular commits grandes "para un solo PR".
- No postergar la integración esperando a que la feature esté "completa".
