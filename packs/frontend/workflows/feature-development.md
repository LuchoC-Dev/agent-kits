---
id: feature-development
description: Workflow completo para desarrollar una feature frontend desde la idea hasta el PR. Aplica las disciplinas de desarrollo activas del proyecto.

steps:
  - id: disciplines-check
    description: Leer workspace.json para saber qué disciplinas de desarrollo están activas

  - id: ideation
    skill: brainstorming
    description: Explorar requerimientos y definir diseño con el usuario

  - id: design
    skill: ui-ux-pro-max
    description: Definir estilos, componentes y estructura visual
    applies_disciplines:
      - contract-first

  - id: behavior-spec
    skill: bdd
    description: Escribir los escenarios de comportamiento de la feature
    condition: bdd activa en disciplines

  - id: docs-lookup
    skill: find-docs
    description: Verificar APIs y documentación relevante antes de implementar

  - id: implementation
    description: Implementar componentes siguiendo el diseño aprobado
    applies_disciplines:
      - tdd

  - id: commit
    skill: git-commit
    description: Commit con mensaje convencional
    applies_disciplines:
      - trunk-based

  - id: pr
    skill: branch-pr
    description: Abrir PR con descripción y checklist
    applies_disciplines:
      - trunk-based
---

# Feature Development Workflow — Frontend

Flujo estándar para nuevas features en proyectos frontend.

## Disciplinas de desarrollo

Este workflow respeta las **disciplinas de desarrollo** activas del proyecto. El conjunto
activo se registra en `workspace.json` bajo el campo `disciplines`, por ejemplo:

```json
"disciplines": ["tdd", "bdd"]
```

Las disciplinas son **autónomas y combinables**: cualquier subconjunto es válido y ninguna
es excluyente con otra. Cada una se engancha en un punto natural del flujo; el workflow
la invoca **solo si está activa**.

| Disciplina | Skill | Punto de enganche |
|---|---|---|
| `contract-first` | `contract-first` | Paso 2 (design) — congela el contrato de la interfaz consumida (API, props de componentes públicos) |
| `bdd` | `bdd` | Paso 3 (behavior-spec) — escenarios antes de implementar |
| `tdd` | `tdd` | Paso 5 (implementation) — envuelve la implementación |
| `trunk-based` | `trunk-based` | Pasos 6–7 (commit, pr) — gobierna la cadencia de integración |

> **Si `workspace.json` no tiene el campo `disciplines`** (proyecto inicializado con una
> versión previa de `kits-init`): preguntá al usuario qué disciplinas quiere aplicar y, si
> lo confirma, persistí su elección en `workspace.json`.

## Pasos

0. **Disciplinas** — leer `workspace.json` y resolver el conjunto activo de disciplinas.
1. **Ideación** (`brainstorming`) — definir qué se va a construir y por qué.
2. **Diseño** (`ui-ux-pro-max`) — decidir estilos, layout y componentes.
   - Si `contract-first` está activa → aplicar la skill `contract-first` para congelar el
     contrato de las interfaces que el componente consume o expone (API, tipos de props).
3. **Especificación de comportamiento** *(solo si `bdd` activa)* (`bdd`) — escribir los
   escenarios Given/When/Then de la feature y acordarlos con el usuario.
4. **Docs** (`find-docs`) — verificar APIs actualizadas de las librerías a usar.
5. **Implementación** — código de los componentes.
   - Si `tdd` está activa → la implementación se conduce con la skill `tdd`: ciclo
     red → green → refactor. Cada escenario de BDD (si está activa) se satisface con
     varios ciclos.
6. **Commit** (`git-commit`) — si `trunk-based` está activa: commits chicos, `trunk`
   siempre verde.
7. **PR** (`branch-pr`) — si `trunk-based` está activa: ramas de vida corta, PR chico,
   integración frecuente.

## Reglas

- No saltar al paso de implementación sin aprobar el diseño del paso 1.
- Las disciplinas activas no son opcionales una vez resueltas: si `tdd` está activa, no se
  escribe código sin test rojo previo; si `bdd` está activa, no se implementa sin
  escenarios acordados.
