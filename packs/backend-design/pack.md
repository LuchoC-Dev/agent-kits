---
id: backend-design
version: "0.1.0"
description: Pack de diseño técnico de backend pre-build. 7 fases (API contract, modelo de datos, arquitectura de servicios, convenciones de código, integraciones & auth, no-funcionales, testing) con 3 modos de scope. Hermano de `design` — corre en paralelo, ambos consumen `context`.

stack_hints: []

produces:
  - artifact: backend-design-spec
    path: docs/backend-design/
    description: Diseño técnico del backend — API, datos, arquitectura, convenciones, integraciones, no-funcionales y testing. En 7 archivos canónicos o un backend-design.md consolidado.

consumes:
  - artifact: product-context
    from: context
    required: false
    description: Usa domain-model (entidades → recursos y schema), job-stories y vision como insumo.
  - artifact: tech-decisions
    from: design
    required: false
    description: Lee el campo `scope`. Si es `fullstack`, el stack del backend ya está decidido y se omite el Paso 0.

structure_extras:
  - skills
  - agents
  - workflows

skills:
  - id: api-contract
    description: Fase 1 — recursos, endpoints, estilo (REST/GraphQL/RPC), versionado, contrato de errores

  - id: data-model
    description: Fase 2 — schema técnico desde el domain-model, motor de BD, índices, migraciones

  - id: service-architecture
    description: Fase 3 — estilo arquitectónico, módulos/servicios, límites, dónde vive la lógica

  - id: code-conventions
    description: Fase 4 — patrones de diseño, reglas de código, idioms, anti-patrones (el rulebook del agente)

  - id: integrations-auth
    description: Fase 5 — terceros, mensajería, autenticación y autorización

  - id: nonfunctional
    description: Fase 6 — escalabilidad, caching, observabilidad, resiliencia

  - id: testing-strategy
    description: Fase 7 — capas de test, contract testing, test data (estrategia, no TDD)

  - id: security-design
    description: Fase 8 (opcional) — threat modeling, hardening, protección de datos, compliance

agents:
  - id: backend-designer
    description: Orquesta el workflow backend-design en cualquiera de sus 3 modos (Basic/Medium/Advanced)

workflows:
  - id: backend-design
    description: Workflow de diseño técnico de backend pre-build con 3 modos de scope
---

# Backend Design Pack

Pack de **diseño técnico de backend pre-build**. Su propósito: dejar el contrato de API, el modelo de datos, la arquitectura y las convenciones de código documentados y aprobados **antes** de escribir código de servidor.

## Filosofía

> Diseñar el backend antes de codearlo. Adaptar la profundidad al tamaño del sistema.

Es el **hermano de `design`**: `design` es el pre-build del frontend, `backend-design` es el pre-build del backend. Los dos corren en paralelo y los dos consumen `context`.

```
context  →  design          ┐
         →  backend-design   ┘ → frontend / backend (implementación)
```

## Fases (workflow `backend-design`)

| # | Skill | Qué define |
|---|---|---|
| 1 | `api-contract` | Recursos, endpoints, estilo, versionado, errores |
| 2 | `data-model` | Schema técnico, motor de BD, índices, migraciones |
| 3 | `service-architecture` | Estilo arquitectónico, módulos, límites, dónde vive la lógica |
| 4 | `code-conventions` | Patrones, reglas de código, anti-patrones — el rulebook |
| 5 | `integrations-auth` | Terceros, mensajería, autenticación, autorización |
| 6 | `nonfunctional` | Escalabilidad, caching, observabilidad, resiliencia |
| 7 | `testing-strategy` | Capas de test, contract testing, test data |
| 8 | `security-design` *(opcional)* | Threat modeling, hardening, protección de datos, compliance |

La fase 8 es **opcional** — entra solo si el sistema maneja datos sensibles, está regulado o tiene alta superficie de ataque (el workflow lo evalúa al inicio).

## Modos del workflow

| Modo | Para qué | Fases | Output |
|---|---|---|---|
| **Basic** | CRUD simple, 1 servicio, MVP de API | 1-2 + mini-arquitectura, convenciones inline | 1 archivo `docs/backend-design/backend-design.md` |
| **Medium** | API con varios dominios, complejidad normal | 1-5 (no-funcionales inline) | archivos separados |
| **Advanced** | Sistema distribuido, microservicios, equipo grande | las 7 completas | 7 archivos numerados |

El usuario elige el modo al iniciar. El agente recomienda según el contexto, pero la decisión es del usuario.

## Paso 0 — Stack del backend

Antes de la fase 1, el workflow decide el stack del backend (runtime/lenguaje, framework de API, motor de BD, ORM, librería de auth). Este paso **se omite** si el pack `design` ya corrió `tech-decisions` con `scope: fullstack` — en ese caso el stack del backend ya está decidido y `backend-design` lo lee en vez de volver a preguntar.

## Contrato de integración

- **Consume** `product-context` (de `context`, opcional) y `tech-decisions` (de `design`, opcional).
- **Produce** `backend-design-spec` en `docs/backend-design/`. Cada archivo lleva frontmatter de identidad (`pack: backend-design`, `artifact: <fase>`).
- Si seguís una arquitectura DDD, la fase `data-model` y `service-architecture` usan `domain-model` como base y dejan un link bidireccional hacia él.

## Qué NO está en este pack

Las **metodologías de implementación** (TDD/test-first, BDD, SDD) viven en el futuro pack `implementation/`. Este pack define la *estrategia* de testing (qué probar, en qué capas), no el *cómo* test-first.

## Output

```
# Medium / Advanced → archivos separados
docs/backend-design/
├── 00-stack.md
├── 01-api-contract.md
├── 02-data-model.md
├── 03-service-architecture.md
├── 04-code-conventions.md
├── 05-integrations-auth.md
├── 06-nonfunctional.md     (solo Advanced)
├── 07-testing-strategy.md  (solo Advanced)
└── 08-security-design.md   (opcional — si lo amerita la seguridad)

# Basic → un solo archivo consolidado
docs/backend-design/
└── backend-design.md
```

## Dependencias

Ninguna formal. Pack standalone — `context` y `design` son inputs opcionales declarados en `consumes`.

## Recomendación de uso

```
context  →  design + backend-design  →  frontend / backend
```
