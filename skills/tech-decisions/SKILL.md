---
name: tech-decisions
description: Decide el stack técnico concreto — framework, styling, componentes, state management, forms, routing, testing. Acepta scope frontend (solo front) o fullstack (back + front). Trigger cuando el usuario quiera "decidir el stack", "elegir framework", "Tailwind o CSS modules", "shadcn o custom", "state management", "tech decisions" o necesite cerrar las decisiones técnicas antes de implementar.
consumes:
  - artifact: discovery
    required: false
  - artifact: component-inventory
    required: false
  - artifact: design-tokens
    required: false
  - artifact: accessibility-plan
    required: false
  - artifact: performance-budget
    required: false
produces:
  - artifact: tech-decisions
---

# Tech Decisions

## Rol

Sos un tech lead cerrando decisiones técnicas. Tu trabajo es **elegir el stack concreto** basado en los artefactos de diseño disponibles, no en preferencias o hype.

## Precondición

Los artefactos de diseño disponibles. Las decisiones se justifican con datos del proyecto (`performance-budget`, `component-inventory`, `design-tokens`, etc.).

## Scope — frontend o fullstack

Por defecto esta skill decide **solo el stack de frontend**. Antes de empezar, preguntá:

> "¿Cierro solo el stack de **frontend**, o también el de **backend** (runtime del server, framework de API, base de datos, ORM, auth)?"

- **Solo frontend** (default) → `scope: frontend` en el frontmatter del output. Si después corre un pack de backend-design, ese pack se encarga del stack del back.
- **Fullstack** → `scope: fullstack`. Agregá una sección de decisiones de backend (runtime, framework de API, base de datos, ORM/query layer, auth). Un pack de backend-design posterior detecta `scope: fullstack` y **salta** su propia fase de stack para no decidir dos veces.

El `scope` se registra en el frontmatter de identidad del archivo de output (ver "Convención de artefactos" del workflow).

## Decisiones a tomar

### 1. Framework / meta-framework

- **React** + Next.js / Remix / Vite + React Router
- **Vue** + Nuxt
- **Svelte** + SvelteKit
- **Solid**, **Qwik**, **Astro** (casos específicos)

**Criterio:** SSR/SSG necesario? Equipo? Bundle size? Ecosistema?

### 2. Styling

- **Tailwind CSS** (utility-first, rápido prototipar)
- **CSS Modules** (scoped, sin runtime)
- **Vanilla Extract / Panda CSS** (typed, zero-runtime)
- **CSS-in-JS** (styled-components, emotion) — solo si hay razón fuerte

**Criterio:** `design-tokens` + `performance-budget` + DX.

### 3. Component library

- **shadcn/ui** (Radix + Tailwind, copiás los componentes)
- **Radix UI** (sin estilos, vos los ponés)
- **Headless UI**, **Ark UI**, **React Aria** (alternativas headless)
- **MUI**, **Mantine**, **Chakra** (opinionated, más rápido pero pesado)
- **Custom** (solo si tenés design system propio fuerte)

**Criterio:** `component-inventory` + `accessibility-plan` + `performance-budget`.

### 4. State management

- **Local state** (useState/useReducer) — default
- **URL state** (search params, router) — para filtros, paginación
- **Server state** — TanStack Query / SWR
- **Global client state** — Zustand / Jotai (si realmente hace falta)
- **Forms** — React Hook Form + Zod (recomendado)

**Criterio:** ¿Cuánto state es server vs client? ¿Hay state global real?

### 5. Routing

- Si meta-framework: el que viene (Next, Remix, Nuxt)
- Si Vite/SPA: React Router, TanStack Router
- File-based vs code-based

### 6. Forms + Validation

- React Hook Form + Zod (esquemas compartibles con backend)
- Alternativas: Formik, TanStack Form, Conform

### 7. Testing

- **Unit:** Vitest / Jest
- **Component:** Testing Library + Vitest
- **E2E:** Playwright (recomendado) o Cypress
- **Visual regression:** Chromatic, Percy (opcional)

### 8. Tooling

- **Linter:** ESLint + plugins relevantes (react, a11y, tailwindcss)
- **Formatter:** Prettier
- **TypeScript:** estricto sí/no
- **Pre-commit:** Husky + lint-staged
- **Bundler:** Vite / Turbopack / Webpack

## Output

`NN-tech-decisions.md` — la carpeta la indica el workflow que te invoca:

- Standalone (pack `design`) → `docs/design/NN-tech-decisions.md`
- Fullstack (workflow `fullstack-design`) → `docs/shared/NN-tech-decisions.md` — es un artefacto cross-cutting que usan ambos tracks.

`NN` = prefijo numérico que asigna el workflow según el orden real de ejecución. Contenido:

```md
---
pack: design
artifact: tech-decisions
scope: frontend
---

# Tech Decisions

## Stack

| Categoría | Decisión | Por qué |
|---|---|---|
| Framework | Next.js 15 (App Router) | SSR para SEO + RSC para budget |
| Styling | Tailwind CSS | Tokens directos del paso 5 + tree-shaking |
| Components | shadcn/ui | Accesible + customizable + sin bundle extra |
| Server state | TanStack Query | Cache + revalidación + DX |
| Client state | URL + useState | No hay state global real |
| Forms | React Hook Form + Zod | Validación compartida con backend |
| Routing | Next.js App Router | Viene con el framework |
| Testing | Vitest + Testing Library + Playwright | Stack estándar |
| Lint/Format | ESLint + Prettier + TS estricto | Calidad desde el inicio |

## Versiones lockeadas

- Node: 20.x
- pnpm: 9.x
- TypeScript: 5.x

## Estructura de carpetas

\`\`\`
src/
├── app/              # rutas (Next App Router)
├── components/
│   ├── ui/           # shadcn components
│   ├── forms/
│   └── layouts/
├── lib/              # utils, server actions
├── hooks/
└── styles/
\`\`\`
```

## Regla

Cada decisión debe tener **una línea de "por qué"** justificada con un output de fases anteriores. Si no podés justificarla, no la tomes.
