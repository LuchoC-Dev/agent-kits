---
version: "0.1.0"
updated_at: "2026-05-22"
---

# Catálogo global — Índice

Leé este archivo en lugar de hacer `ls` + leer cada `pack.md` / `SKILL.md`.
Contiene toda la información necesaria para presentar opciones al usuario.

## Packs

| id | descripción | grupo |
|---|---|---|
| context | Construcción de contexto pre-diseño: problem framing, stakeholders, domain model, job stories, visión, assumptions. Genera `docs/context/` que alimenta a `design`. | Diseño y contexto |
| design | Diseño pre-build completo: discovery, IA, flows, componentes, tokens, wireframes, a11y, performance, tech decisions. Aplica a frontend, mobile y productos digitales. | Diseño y contexto |
| backend-design | Diseño técnico de backend pre-build: API contract, modelo de datos, arquitectura, convenciones, integraciones & auth, no-funcionales, testing. Hermano de `design`. | Diseño y contexto |
| fullstack-design | Orquestador de diseño fullstack: compone `design` + `backend-design` en un solo workflow coordinado con 3 tiers (Lite/Standard/Full). | Diseño y contexto |
| backend | Desarrollo backend: Docker, configuración de entornos, automatización de tareas, documentación de progreso y workflow Git. | Implementación |
| frontend | Desarrollo frontend moderno: UI/UX, ideación y documentación. El diseño pre-build requiere el pack `design`. | Implementación |
| tools | Herramientas transversales: Git, GitHub CLI, Docker, variables de entorno, automatización. Funciona con cualquier stack. | Implementación |

## Skills del pool global

### Contexto y descubrimiento
| id | descripción |
|---|---|
| problem-framing | Define el problema real con "5 Whys" antes de saltar a soluciones. |
| stakeholders-map | Mapea actores, poder, interés y postura. Quién bloquea, impulsa, paga. |
| domain-modeling | Modelo del dominio: entidades, relaciones, eventos y lenguaje ubicuo (DDD estratégico). |
| job-stories | Job Stories (cuando ___, quiero ___, así puedo ___) — motivaciones reales del usuario. |
| vision | Visión del producto a 6-12 meses como north star medible y anti-visión. |
| assumptions-risks | Mapea apuestas (hypotheses) y riesgos (unknowns/threats) por criticidad. |
| brainstorming | Exploración creativa de ideas, features y soluciones antes de implementar. |

### Diseño / Frontend
| id | descripción |
|---|---|
| discovery | Define producto, usuarios, JTBD y métricas de éxito. Arranca con Intake adaptado al contexto. |
| information-architecture | Sitemap, jerarquía de navegación y rutas antes de diseñar pantallas. |
| user-flows | Caminos críticos del usuario paso a paso: estados, decisiones y errores. |
| component-inventory | Inventario exhaustivo de componentes UI en Atomic Design (atoms, molecules, organisms). |
| design-system-tokens | Design tokens: colores, tipografía, spacing, radios, sombras, breakpoints, motion. |
| wireframes | Wireframes low-fi ASCII/Markdown por pantalla: estructura y jerarquía sin estilos. |
| html-mockup | Mockups HTML+Tailwind por pantalla con fidelidad ajustable (mid-fi / hi-fi). |
| progressive-design | Diseño progresivo: wireframes ASCII → html-mockup → ajustes iterativos. |
| accessibility-plan | Plan de accesibilidad WCAG AA: contraste, teclado, ARIA, focus, screen readers. |
| performance-budget | Presupuestos de performance: Core Web Vitals, bundle size, tiempos de carga. |
| ui-ux-pro-max | UI/UX design intelligence: 50+ estilos, paletas, font pairings, guías UX. |

### Diseño de backend
| id | descripción |
|---|---|
| tech-decisions | Stack técnico: framework, styling, state management, testing. Scope frontend o fullstack. |
| api-contract | Contrato de API: recursos, endpoints, REST/GraphQL, versionado, request/response, errores. |
| data-model | Schema técnico: tablas/colecciones, tipos, claves, relaciones, índices, migraciones. |
| service-architecture | Estilo arquitectónico: capas, hexagonal, clean, DDD táctico, microservicios, módulos. |
| code-conventions | Rulebook de código: patrones, guard clauses, naming, idioms, anti-patrones. |
| integrations-auth | Integraciones externas, mensajería/eventos, autenticación y autorización (JWT, OAuth, RBAC). |
| nonfunctional | NFRs y operabilidad: escalabilidad, caching, observabilidad, resiliencia. |
| testing-strategy | Estrategia de testing: capas (unit/integration/e2e), contract testing, test data, cobertura. |
| security-design | Seguridad: threat modeling, hardening, protección de datos, compliance, audit logging. |

### Tools / Transversales
| id | descripción |
|---|---|
| git-commit | Commits convencionales con análisis del diff y staging inteligente. |
| branch-pr | Gestión de ramas y pull requests con convenciones de proyecto. |
| gh-cli | GitHub CLI: repositorios, issues, PRs, Actions, projects, releases. |
| docker-expert | Docker avanzado: optimización, seguridad, multi-stage builds, orquestación. |
| environment-setup | Entornos de desarrollo/staging/prod: .env, configuración, separación de contextos. |
| taskfile-setup | Automatización de tareas repetibles con Taskfile. |
| build-progress | Documentación del progreso mientras se construye. |
| find-docs | Búsqueda de documentación técnica relevante. |

### Disciplinas (`discipline: true`, `combinable: true`)
| id | descripción |
|---|---|
| tdd | Test-Driven Development — ciclo red → green → refactor. El test falla antes del código. |
| bdd | Behavior-Driven Development — escenarios Given/When/Then antes de implementar. |
| contract-first | API-first — contrato (OpenAPI/schema/proto) congelado antes de implementar cualquier lado. |
| trunk-based | Trunk-based development — ramas cortas, commits chicos, trunk siempre verde. |
