---
id: environment-bootstrap
description: Setup inicial del entorno de desarrollo, staging y producción para un proyecto backend nuevo.

steps:
  - id: env-files
    skill: environment-setup
    description: Crear .env, .env.example, .env.test con variables base

  - id: docker-setup
    skill: docker-expert
    description: Crear Dockerfile multi-stage + docker-compose.yml para dev y prod

  - id: taskfile
    skill: taskfile-setup
    description: Crear Taskfile.yml con comandos estándar (dev, build, test, docker:up, docker:down)

  - id: docs
    skill: build-progress
    description: Crear docs/build.md con el plan de construcción del proyecto
---

# Environment Bootstrap

Setup estándar para arrancar un proyecto backend desde cero.

## Pasos

1. **Env files** (`environment-setup`) — `.env`, `.env.example`, `.env.test`
2. **Docker** (`docker-expert`) — Dockerfile multi-stage + Compose para dev/prod
3. **Taskfile** (`taskfile-setup`) — comandos estándar del proyecto
4. **Progreso** (`build-progress`) — tracker inicial del build

## Resultado esperado

```
project/
├── .env
├── .env.example
├── .env.test
├── Dockerfile
├── docker-compose.yml
├── docker-compose.prod.yml
├── Taskfile.yml
└── docs/
    └── build.md
```
