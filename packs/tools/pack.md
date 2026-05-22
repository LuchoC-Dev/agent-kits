---
id: tools
version: "0.1.0"
description: Pack de herramientas transversales de desarrollo. Git, GitHub CLI, Docker, variables de entorno y automatización de tareas. Funciona con cualquier stack.

stack_hints: []

produces:
  - artifact: dev-tooling
    path: ["Taskfile.yml", "Dockerfile", "docker-compose.yml", ".env*"]
    description: Configuración transversal de desarrollo (no es un artefacto de documentación).

consumes: []

structure_extras:
  - skills

skills:
  - id: git-commit
    description: Commits convencionales con análisis de diff y staging inteligente

  - id: branch-pr
    description: Workflow de branches y pull requests

  - id: gh-cli
    description: GitHub CLI — repos, issues, PRs, Actions, releases desde la terminal

  - id: environment-setup
    description: Variables de entorno, .env files, configuración por entorno

  - id: docker-expert
    description: Containerización, multi-stage builds, Docker Compose, seguridad

  - id: taskfile-setup
    description: Automatización de tareas con Taskfile.yml
---

# Tools Pack

Pack de herramientas transversales. No está atado a ningún stack — se puede instalar solo o combinado con cualquier otro pack.

## Skills incluidas

| Skill | Descripción |
|---|---|
| `git-commit` | Commits semánticos con análisis de diff |
| `branch-pr` | Branches y pull requests |
| `gh-cli` | GitHub CLI completo |
| `environment-setup` | Gestión de .env por entorno |
| `docker-expert` | Containerización y Compose |
| `taskfile-setup` | Automatización con Taskfile.yml |

## Nota

Este pack no incluye agentes ni workflows — solo skills. Los packs de dominio (frontend, backend) pueden depender de él o instalarlo en conjunto.
