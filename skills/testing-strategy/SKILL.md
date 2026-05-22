---
name: testing-strategy
description: Define la estrategia de testing — capas de test (unit, integration, e2e), contract testing entre servicios, estrategia de test data y targets de cobertura. Es estrategia, no metodología test-first (eso vive en el pack implementation). Trigger cuando el usuario quiera "estrategia de testing", "qué probar", "tests de integración", "contract testing", "test data", "cobertura" o necesite cerrar cómo se prueba el backend antes de implementar.
consumes:
  - artifact: service-architecture
    required: true
  - artifact: integrations-auth
    required: true
  - artifact: nonfunctional
    required: false
produces:
  - artifact: testing-strategy
---

# Testing Strategy

## Rol

Sos un QA architect. Tu trabajo es definir **qué se prueba, en qué capa, y cómo** — antes de implementar.

> Esto es **estrategia de testing**, no metodología. El *cómo* del test-first (TDD), los escenarios ejecutables (BDD) y las specs formales (SDD) son del futuro pack `implementation/`. Acá se decide qué probar y con qué pirámide.

## Parámetro de depth

- `full` — estrategia completa: pirámide, contract testing, test data, cobertura. Cuando no se ejecuta por separado, el testing se define inline en `service-architecture`.

## Precondición

Los artefactos `service-architecture`, `integrations-auth` y `nonfunctional` aprobados — la estrategia de testing se apoya en la arquitectura, las integraciones y los no-funcionales ya definidos.

## Workflow

### Paso 1 — Pirámide de tests

Definí las capas y qué cubre cada una:

- **Unit** — lógica de dominio, funciones puras, reglas de negocio. Rápidos, muchos.
- **Integration** — repositorios contra BD real, adapters contra servicios (o sus dobles), endpoints contra la app.
- **End-to-end** — flujos completos de usuario contra el sistema corriendo. Pocos, los críticos.

Decidí la proporción y qué tipo de lógica vive en cada capa.

### Paso 2 — Contract testing (si hay varios servicios)

Si la arquitectura es de microservicios o consume APIs externas: contract testing entre productor y consumidor, para que un cambio de contrato rompa el test y no producción.

### Paso 3 — Estrategia de test data

- Cómo se generan los datos de prueba (factories, fixtures, builders).
- Aislamiento entre tests (transacción por test, base de datos efímera, contenedor).
- Datos seed para integration/e2e.

### Paso 4 — Dobles de prueba

Cuándo usar mocks, stubs, fakes o el servicio real. Regla: la lógica de dominio se testea sin dobles; las fronteras (red, BD) se aíslan o se contienen.

### Paso 5 — Targets de cobertura

Targets realistas y por capa (no un número único global). Qué es obligatorio cubrir (lógica de dominio, reglas de negocio) y qué no aporta cubrir (getters, mapeos triviales).

## Output — `docs/backend-design/NN-testing-strategy.md`

> `NN` = prefijo numérico de dos dígitos que asigna el **workflow** según el orden real de ejecución. No lo inventes — tomalo de la tabla del workflow que te invoca.

```md
---
pack: backend-design
artifact: testing-strategy
---

# Testing Strategy

## Pirámide de tests
| Capa | Qué cubre | Proporción | Herramienta |
|---|---|---|---|
| Unit | Lógica de dominio, reglas | mucha | ... |
| Integration | Repos, adapters, endpoints | media | ... |
| E2E | Flujos críticos | poca | ... |

## Contract testing
> Entre qué servicios, con qué herramienta. (Omitir si es monolito sin APIs externas.)

## Estrategia de test data
- Generación: <factories | fixtures | builders>
- Aislamiento: <transacción por test | BD efímera>
- Seed para integration/e2e: ...

## Dobles de prueba
> Cuándo mock / stub / fake / real.

## Targets de cobertura
| Capa | Target | Obligatorio cubrir |
|---|---|---|
| Lógica de dominio | <%> | sí |
| ... | ... | ... |
```

## Reglas duras

- **Esto es estrategia, no TDD.** El test-first vive en el pack `implementation/`.
- **La lógica de dominio se testea sin dobles** — si necesitás mockear para testear una regla de negocio, la arquitectura está mal.
- **Targets por capa, no un número global** — 80% global esconde 0% en lo que importa.
- **Toda llamada de red en un test se aísla o se contiene** — los tests no dependen de internet.
