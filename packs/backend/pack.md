---
id: backend
version: "0.1.0"
description: Pack de desarrollo backend. Incluye capacidades de Docker, configuración de entornos, automatización de tareas, documentación de progreso y workflow Git.

depends_on:
  - tools

stack_hints:
  - express
  - fastify
  - nest
  - nestjs
  - django
  - fastapi
  - flask
  - spring
  - springboot
  - rails
  - laravel
  - go
  - rust
  - postgres
  - mysql
  - mongodb
  - prisma
  - typeorm

produces:
  - artifact: build-progress
    path: docs/build.md
    description: Tracker de progreso de la implementación.

consumes:
  - artifact: backend-design-spec
    from: backend-design
    required: false
    description: Plano técnico del backend — API, datos, arquitectura, convenciones. La implementación lo sigue.
  - artifact: product-context
    from: context
    required: false
    description: El qué y el porqué — usado si no hay backend-design previo.

structure_extras:
  - skills
  - agents
  - workflows

skills:
  - id: build-progress
    description: Tracker de progreso del proyecto en docs/build.md

  - id: find-docs
    description: Documentación actualizada de cualquier framework o librería

  - id: branch-pr
    description: Workflow de branches y pull requests

agents:
  - id: backend-engineer
    description: Ingeniero backend senior con foco en APIs, infraestructura y buenas prácticas

workflows:
  - id: feature-development
    description: Diseño → implementación → tests → Docker → commit → PR

  - id: environment-bootstrap
    description: Setup inicial del entorno de desarrollo, staging y producción
---

# Backend Pack

Pack de capacidades para proyectos backend. Al instalarse, distribuye sus skills, agentes y workflows en las carpetas canónicas del workspace (`.my-system/skills/`, `.my-system/agents/`, `.my-system/workflows/`).

## Skills incluidas

| Skill | Descripción |
|---|---|
| `docker-expert` | Containerización, optimización de imágenes, Compose |
| `environment-setup` | Gestión de .env, separación de entornos |
| `taskfile-setup` | Automatización con Taskfile.yml |
| `build-progress` | Tracker de progreso en docs/build.md |
| `find-docs` | Documentación actualizada de librerías y frameworks |
| `git-commit` | Commits semánticos con análisis de diff |
| `branch-pr` | Creación de branches y pull requests |

## Agentes incluidos

- **backend-engineer** — perfil operativo para APIs e infraestructura

## Workflows incluidos

- **feature-development** — flujo completo desde diseño hasta PR
- **environment-bootstrap** — setup de entornos dev/staging/prod

## Stack detectado automáticamente

Si `/app-init` detecta cualquiera de los siguientes, este pack se recomienda automáticamente:
`express`, `fastify`, `nestjs`, `django`, `fastapi`, `flask`, `spring`, `rails`, `laravel`, `go`, `rust`, `prisma`, `typeorm`
