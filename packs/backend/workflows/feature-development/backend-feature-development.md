---
id: feature-development
description: Workflow para desarrollar una feature backend desde el diseño hasta el PR. Aplica las disciplinas de desarrollo activas del proyecto.

steps:
  - id: disciplines-check
    description: Leer workspace.json para saber qué disciplinas de desarrollo están activas

  - id: design
    description: Definir contrato de la API (endpoints, request/response, errores)
    applies_disciplines:
      - contract-first

  - id: behavior-spec
    skill: bdd
    description: Escribir los escenarios de comportamiento de la feature
    condition: bdd activa en disciplines

  - id: docs-lookup
    skill: find-docs
    description: Verificar APIs y documentación del framework antes de implementar

  - id: implementation
    description: Implementar lógica de negocio, repositorios, controladores
    applies_disciplines:
      - tdd

  - id: environment
    skill: environment-setup
    description: Agregar variables de entorno necesarias a .env y .env.example

  - id: docker
    skill: docker-expert
    description: Actualizar Dockerfile o Compose si la feature lo requiere

  - id: tasks
    skill: taskfile-setup
    description: Agregar comandos al Taskfile si hay nuevos scripts necesarios

  - id: progress
    skill: build-progress
    description: Marcar la feature como completada en docs/build.md

  - id: commit
    skill: git-commit
    description: Commit semántico
    applies_disciplines:
      - trunk-based

  - id: pr
    skill: branch-pr
    description: Pull request con descripción y checklist
    applies_disciplines:
      - trunk-based
---

# Feature Development — Backend

## Disciplinas de desarrollo

Este workflow respeta las **disciplinas de desarrollo** activas del proyecto. El conjunto
activo se registra en `workspace.json` bajo el campo `disciplines`, por ejemplo:

```json
"disciplines": ["tdd", "bdd", "contract-first"]
```

Las disciplinas son **autónomas y combinables**: cualquier subconjunto es válido y ninguna
es excluyente con otra. Cada una se engancha en un punto natural del flujo; el workflow
la invoca **solo si está activa**.

| Disciplina | Skill | Punto de enganche |
|---|---|---|
| `contract-first` | `contract-first` | Paso 1 (design) — congela el contrato antes de implementar |
| `bdd` | `bdd` | Paso 2 (behavior-spec) — escenarios antes de implementar |
| `tdd` | `tdd` | Paso 4 (implementation) — envuelve la implementación |
| `trunk-based` | `trunk-based` | Pasos 8–9 (commit, pr) — gobierna la cadencia de integración |

> **Si `workspace.json` no tiene el campo `disciplines`** (proyecto inicializado con una
> versión previa de `kits-init`): preguntá al usuario qué disciplinas quiere aplicar a esta
> feature y, si lo confirma, persistí su elección en `workspace.json`.

## Pasos

0. **Disciplinas** — leer `workspace.json` y resolver el conjunto activo de disciplinas.
1. **Diseño** — contrato de API antes de escribir código.
   - Si `contract-first` está activa → aplicar la skill `contract-first`: el contrato se
     materializa como artefacto formal (OpenAPI/schema) y se congela.
2. **Especificación de comportamiento** *(solo si `bdd` activa)* (`bdd`) — escribir los
   escenarios Given/When/Then de la feature y acordarlos con el usuario.
3. **Docs** (`find-docs`) — verificar APIs actualizadas.
4. **Implementación** — lógica, repositorios, controladores.
   - Si `tdd` está activa → la implementación se conduce con la skill `tdd`: ciclo
     red → green → refactor por cada unidad de comportamiento. Cada escenario de BDD
     (si está activa) se satisface con varios ciclos.
5. **Entorno** (`environment-setup`) — variables nuevas en .env.
6. **Docker** (`docker-expert`) — actualizar contenedores si aplica.
7. **Tasks** (`taskfile-setup`) — nuevos comandos al Taskfile.
8. **Progreso** (`build-progress`) — actualizar tracker.
9. **Commit** (`git-commit`) — si `trunk-based` está activa, respetar su cadencia:
   commits chicos, `trunk` siempre verde.
10. **PR** (`branch-pr`) — si `trunk-based` está activa: ramas de vida corta, PR chico,
    integración frecuente.

## Reglas

- No saltar al paso de implementación sin el diseño aprobado.
- Las disciplinas activas no son opcionales una vez resueltas: si `tdd` está activa, no se
  escribe código sin test rojo previo; si `bdd` está activa, no se implementa sin
  escenarios acordados.
