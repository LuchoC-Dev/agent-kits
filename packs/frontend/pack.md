---
id: frontend
version: "0.3.0"
description: Pack de desarrollo frontend moderno. Capacidades de UI/UX, ideación y documentación. El diseño pre-build vive en el pack `design` (dependencia).

depends_on:
  - tools
  - design

stack_hints:
  - react
  - vue
  - next
  - nuxt
  - vite
  - typescript
  - tailwind
  - svelte

produces: []

consumes:
  - artifact: design-spec
    from: design
    required: false
    description: Base de la implementación de UI — el código sigue lo definido acá.
  - artifact: product-context
    from: context
    required: false
    description: El qué y el porqué del producto, si discovery no lo absorbió ya.

structure_extras:
  - skills
  - agents
  - workflows

skills:
  - id: ui-ux-pro-max
    description: Diseño UI/UX — estilos, paletas, componentes, guidelines

  - id: brainstorming
    description: Ideación y diseño antes de implementar

  - id: find-docs
    description: Documentación actualizada de React, Vite, Tailwind, etc.

agents:
  - id: frontend-architect
    description: Arquitecto frontend con foco en React/TS, composabilidad y accesibilidad

workflows:
  - id: feature-development
    description: Ideación → diseño → implementación → commit → PR
---

# Frontend Pack

Pack de capacidades **de implementación** para proyectos frontend modernos. El diseño pre-build (discovery, IA, flows, tokens, wireframes, etc.) vive en el pack `design`, que se instala automáticamente como dependencia.

## Skills incluidas

| Skill | Descripción |
|---|---|
| `ui-ux-pro-max` | Guía completa de diseño UI/UX |
| `brainstorming` | Exploración de ideas y requerimientos |
| `find-docs` | Documentación actualizada de librerías |

## Agentes incluidos

- **frontend-architect** — arquitectura de implementación

## Workflows incluidos

- **feature-development** — ideación → código → PR

## Dependencias

- `tools` — git, branches, docker, env, taskfile
- `design` — workflow pre-build-design (Basic/Medium/Advanced) y agente `designer`

## Stack detectado automáticamente

Si `/kits-init` detecta en `package.json`: `react`, `vue`, `next`, `nuxt`, `vite`, `typescript`, `tailwind`, `svelte`.
