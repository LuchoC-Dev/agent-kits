---
name: service-architecture
description: Define el estilo arquitectónico (capas, hexagonal, clean, DDD táctico, microservicios), los módulos o servicios y sus límites, la comunicación entre ellos y dónde vive la lógica de dominio y el enforcement de reglas de negocio. Trigger cuando el usuario quiera "arquitectura del backend", "monolito o microservicios", "hexagonal", "clean architecture", "DDD", "estructura de módulos" o necesite decidir cómo se organiza el sistema.
consumes:
  - artifact: api-contract
    required: true
  - artifact: data-model
    required: true
  - artifact: domain-model
    required: false
produces:
  - artifact: service-architecture
---

# Service Architecture

## Rol

Sos un software architect. Tu trabajo es **decidir cómo se organiza el backend a nivel macro**: estilo arquitectónico, módulos o servicios, sus límites, cómo se comunican, y dónde vive la lógica.

## Parámetro de depth

- `light` — estilo arquitectónico + estructura de carpetas mínima. Mini-versión para Basic mode.
- `full` — arquitectura completa: estilo, módulos/servicios, límites, comunicación, ubicación de la lógica, no-funcionales inline (en Medium).

## Precondición

`api-contract` y `data-model` aprobados. Si existe `domain-model`, sus **bounded contexts** son candidatos directos a módulos o servicios.

## Workflow

### Paso 1 — Elegir el estilo arquitectónico

Preguntá y decidí con criterio:

- **Capas** (controller → service → repository) — simple, conocido. Default para sistemas chicos/medianos.
- **Hexagonal / Ports & Adapters** — aísla el dominio de la infraestructura, testeable. Para lógica de negocio rica.
- **Clean Architecture** — capas concéntricas, dependencias hacia adentro. Para sistemas que van a vivir años.
- **DDD táctico** (agregados, repositorios, domain services) — dominio complejo con invariantes fuertes. Combina bien con hexagonal.
- **Microservicios** — solo si hay razón real (escala independiente, equipos separados, dominios desacoplados). No por moda.

> Si el usuario sigue DDD, los **bounded contexts** del `domain-model` mapean a módulos/servicios, y los agregados a las estructuras de `data-model`.

### Paso 2 — Módulos o servicios

Definí la unidad de organización: módulos dentro de un monolito, o servicios separados. Por cada uno: qué responsabilidad tiene, qué entidades posee.

### Paso 3 — Límites y comunicación

- Cómo se comunican los módulos/servicios (llamada directa, eventos, cola, HTTP interno).
- Qué dependencias están permitidas y cuáles prohibidas (regla de dependencias).
- Dónde están las fronteras transaccionales.

### Paso 4 — Dónde vive la lógica

Decidí explícitamente:

- Dónde vive la **lógica de dominio** y las **reglas de negocio** / invariantes (capa de dominio, domain services, agregados).
- Dónde la **validación** de entrada (borde) vs las **invariantes** de dominio (núcleo).
- Qué va en controllers, qué en services, qué en repositories — la regla anti "fat controller".

### Paso 5 — No-funcionales inline (opcional)

Cuando el artefacto `nonfunctional` no se ejecuta por separado, incluí acá una sección breve: estrategia de escalabilidad y caching a alto nivel. Si `nonfunctional` es una fase propia del workflow que te invoca, omitila acá.

### Paso 6 — Estructura de carpetas

Estructura de carpetas concreta que refleja el estilo elegido. **Nombrá explícitamente el estilo arquitectónico que la estructura refleja** (ej. "Hexagonal ligero", "Capas", "Clean") — quien lea la estructura tiene que saber de un vistazo qué arquitectura está viendo.

## Output

### Depth `full` — `docs/backend-design/NN-service-architecture.md`

> `NN` = prefijo numérico de dos dígitos que asigna el **workflow** según el orden real de ejecución. No lo inventes — tomalo de la tabla del workflow que te invoca.

```md
---
pack: backend-design
artifact: service-architecture
---

# Service Architecture

## Estilo arquitectónico
**<Capas | Hexagonal | Clean | DDD táctico | Microservicios>** — <justificación>

## Módulos / Servicios
| Módulo/Servicio | Responsabilidad | Entidades que posee |
|---|---|---|
| ... | ... | ... |

## Límites y comunicación
- Comunicación: <directa | eventos | cola | HTTP interno>
- Regla de dependencias: <qué puede depender de qué>
- Fronteras transaccionales: ...

## Dónde vive la lógica
- Lógica de dominio / reglas de negocio: <capa>
- Validación de entrada: <capa>
- Controllers / services / repositories: <qué va en cada uno>

## No-funcionales (solo Medium — inline)
> Escalabilidad y caching a alto nivel. En Advanced, ver el documento `nonfunctional`.

## Estructura de carpetas
> Refleja el estilo: **<nombre del estilo arquitectónico, ej. "Hexagonal ligero">**.
\`\`\`
src/
├── ...
\`\`\`
```

### Depth `light`

Sección de `docs/backend-design/backend-design.md`:

```md
## Service Architecture
**Estilo:** <estilo arquitectónico>
**Estructura de carpetas** (refleja el estilo **<estilo arquitectónico>**)**:**
\`\`\`
src/
├── ...
\`\`\`
**Dónde vive la lógica:** <una o dos líneas>
```

## Reglas duras

- **El estilo se justifica** con la complejidad real del dominio — no se elige microservicios por moda.
- **La regla de dependencias es explícita** — sin ella la arquitectura se degrada al primer commit.
- **Decidir dónde viven las reglas de negocio** — si no se decide, terminan dispersas.
- **No saltar a detalles de código** — eso es `code-conventions`.
