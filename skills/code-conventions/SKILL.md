---
name: code-conventions
description: Define el rulebook de código — patrones de diseño a aplicar y cuándo (Strategy, Repository, Factory, etc.), reglas de código (guard clauses, anidamiento, manejo de errores), naming, idioms y anti-patrones. Es la guía que cualquier desarrollador (humano o IA) sigue al implementar. Trigger cuando el usuario quiera "convenciones de código", "patrones de diseño", "estándares", "guard clauses", "reglas de código", "code style" o necesite fijar cómo se escribe el código antes de implementar.
consumes:
  - artifact: service-architecture
    required: true
  - artifact: tech-decisions
    required: false
produces:
  - artifact: code-conventions
---

# Code Conventions

## Rol

Sos un staff engineer escribiendo el **rulebook de código** del proyecto. Tu trabajo es fijar **cómo se escribe cada función**, para que todo el código —lo escriba un humano o un agente— salga consistente.

> En un sistema AI-first esto es crítico: este documento es lo que el agente lee antes de implementar. Sin él, cada archivo sale con un estilo distinto.

## Parámetro de depth

- `light` — no es fase propia en Basic. Se reduce a 2-3 reglas baseline inline en `backend-design.md` (guard clauses, manejo de errores, naming).
- `full` — rulebook completo: patrones, reglas, idioms, anti-patrones.

## Precondición

`service-architecture` aprobada — los patrones dependen del estilo arquitectónico y del stack elegido.

## Workflow

### Paso 1 — Patrones de diseño a aplicar

Listá los patrones que el proyecto va a usar y **cuándo** usar cada uno. Para cada patrón: qué problema resuelve, un ejemplo corto, cuándo NO usarlo. Candidatos típicos de backend:

- **Repository** — abstrae el acceso a datos.
- **Strategy** — comportamiento intercambiable (ej. métodos de pago, proveedores).
- **Factory** — creación compleja de objetos.
- **Adapter** — envolver integraciones externas.
- **Decorator / Middleware** — cross-cutting concerns (logging, auth).
- **Result / Either** — manejo de errores sin excepciones, si el lenguaje lo favorece.

No listes patrones "por completitud" — solo los que el proyecto realmente va a usar.

> **Legibilidad:** este documento se consulta seguido — si tiene muchos patrones y reglas, se vuelve difícil de usar. Mantené los ejemplos **cortos** (un snippet mínimo, no una clase entera), agrupá patrones relacionados, y abrí el documento con una **tabla-resumen** de una línea por ítem. El detalle va después de la tabla. Si un patrón necesita más de ~10 líneas de explicación, probablemente no es una convención — es arquitectura (fase 3).

### Paso 2 — Reglas de código

Reglas concretas y verificables:

- **Guard clauses / early returns** — salir temprano en vez de anidar.
- **Límite de anidamiento** — ej. máximo 2-3 niveles.
- **Tamaño de función** — cuándo partir una función.
- **Manejo de errores** — excepciones vs result types; qué se captura y dónde; nunca tragar errores en silencio.
- **Inmutabilidad** — cuándo es obligatoria.
- **Side effects** — dónde se permiten y dónde no (núcleo puro vs bordes).

### Paso 3 — Naming e idioms

- Convenciones de nombres (variables, funciones, clases, archivos).
- Idioms del lenguaje/framework elegido en el stack — la forma "idiomática" de hacer las cosas.

### Paso 4 — Anti-patrones a evitar

Lista explícita de lo que **no** se hace: fat controllers, lógica de negocio en el ORM, números mágicos, god objects, etc. Tan importante como lo que sí se hace.

## Output

### Depth `full` — `docs/backend-design/NN-code-conventions.md`

> `NN` = prefijo numérico de dos dígitos que asigna el **workflow** según el orden real de ejecución. No lo inventes — tomalo de la tabla del workflow que te invoca.

```md
---
pack: backend-design
artifact: code-conventions
---

# Code Conventions

## Resumen rápido
> Tabla de referencia — un vistazo. El detalle está más abajo.

| Patrón / Regla | Cuándo aplica |
|---|---|
| Repository | Todo acceso a datos |
| Strategy | Comportamiento intercambiable (pagos, proveedores) |
| Guard clauses | Siempre — early return |
| ... | ... |

## Patrones de diseño
> Detalle de cada patrón de la tabla. Ejemplos cortos.

### <Patrón>
- **Resuelve:** ...
- **Cuándo usarlo:** ...
- **Cuándo NO:** ...
- **Ejemplo:** <snippet corto>

## Reglas de código
| Regla | Detalle |
|---|---|
| Guard clauses | Early return; nada de `else` después de un return |
| Anidamiento | Máximo <N> niveles |
| Manejo de errores | <excepciones | result types>; nunca tragar errores |
| ... | ... |

## Naming e idioms
- <convención de nombres>
- <idioms del stack>

## Anti-patrones (prohibido)
- ...
```

### Depth `light`

2-3 reglas baseline en una sección de `docs/backend-design/backend-design.md`:

```md
## Code Conventions (baseline)
- Guard clauses / early returns.
- Manejo de errores consistente: <una línea>.
- Naming: <una línea>.
```

## Reglas duras

- **Solo patrones que el proyecto va a usar de verdad** — un catálogo teórico no sirve.
- **Cada regla debe ser verificable** — "código limpio" no es una regla; "máximo 2 niveles de anidamiento" sí.
- **La lista de anti-patrones es obligatoria** — definir qué no hacer es la mitad del rulebook.
- **No repetir decisiones de arquitectura** — eso es `service-architecture`; acá es nivel micro (cómo se escribe una función).
- **Legibilidad ante todo** — tabla-resumen al inicio, ejemplos cortos, patrones agrupados. Un rulebook que no se puede leer de un vistazo no se usa.
