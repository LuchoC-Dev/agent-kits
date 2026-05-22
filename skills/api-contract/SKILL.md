---
name: api-contract
description: Define el contrato de la API — recursos, endpoints, estilo (REST/GraphQL/RPC), versionado, formato de request/response y de errores, paginación y filtrado. Trigger cuando el usuario quiera "diseñar la API", "definir endpoints", "contrato de API", "REST o GraphQL", "estructura de la API" o esté por implementar un backend sin el contrato cerrado.
consumes:
  - artifact: tech-decisions
    required: false
  - artifact: domain-model
    required: false
  - artifact: job-stories
    required: false
produces:
  - artifact: api-contract
---

# API Contract

## Rol

Sos un API designer. Tu trabajo es **definir el contrato de la API antes de implementarla**: qué recursos expone, con qué endpoints, en qué estilo, y cómo se comportan request, response y errores.

## Parámetro de depth

- `light` — estilo + recursos + endpoints principales en una tabla. Sin versionado ni paginación detallada. Para Basic mode.
- `full` — contrato completo: estilo justificado, todos los endpoints, formatos, errores, versionado, paginación, filtrado.

## Precondición

Stack resuelto (Paso 0 del workflow). Si existe `product-context`, leé `domain-model` (las entidades son candidatas a recursos) y `job-stories` (cada job suele mapear a uno o más endpoints).

## Workflow

### Paso 1 — Elegir el estilo de API

Decidí con criterio, no por costumbre:

- **REST** — recursos CRUD claros, caching HTTP, ecosistema amplio. Default para la mayoría.
- **GraphQL** — clientes con necesidades de datos muy variables, evitar over/under-fetching, un solo endpoint.
- **RPC / gRPC** — comunicación servicio-a-servicio, baja latencia, contratos tipados.

Justificá la elección con un dato del proyecto (tipo de cliente, flows, performance).

### Paso 2 — Identificar recursos

Partí de las entidades del `domain-model`. No todas las entidades son recursos de API — filtrá las que el cliente realmente consume. Distinguí recursos raíz de subrecursos.

### Paso 3 — Endpoints por recurso

Por cada recurso, definí los endpoints: método, path, qué hace, quién puede llamarlo. Mapeá los `job-stories` a endpoints concretos — si un job no tiene endpoint, falta algo.

### Paso 4 — Formatos y errores

- **Request/response** — estructura del body, campos, tipos.
- **Errores** — formato único de error (código, mensaje, detalles), tabla de códigos de estado usados.
- **Paginación, filtrado, ordenamiento** — convención única para todos los listados.

### Paso 5 — Versionado

Estrategia de versionado (path `/v1`, header, content negotiation) y política de breaking changes.

## Output

### Depth `full` — `docs/backend-design/NN-api-contract.md`

> `NN` = prefijo numérico de dos dígitos que asigna el **workflow** según el orden real de ejecución. No lo inventes — tomalo de la tabla del workflow que te invoca.

```md
---
pack: backend-design
artifact: api-contract
---

# API Contract

## Estilo
**<REST | GraphQL | RPC>** — <justificación con dato del proyecto>

## Recursos
| Recurso | Entidad de dominio | Notas |
|---|---|---|
| ... | ... | ... |

## Endpoints
| Método | Path | Qué hace | Job relacionado | Auth |
|---|---|---|---|---|
| GET | /v1/orders | Lista pedidos del usuario | Job #5 | sí |

## Formato de request / response
> Estructura estándar del body, campos comunes (ids, timestamps).

## Contrato de errores
| Código | Cuándo | Body |
|---|---|---|
| 400 | Validación falla | `{ "error": "...", "details": [...] }` |
| 401 | Sin auth | ... |

## Paginación, filtrado y ordenamiento
> Convención única para todos los listados.

## Versionado
> Estrategia + política de breaking changes.
```

### Depth `light`

Una sección de `docs/backend-design/backend-design.md`:

```md
## API Contract
**Estilo:** <REST | GraphQL | RPC>
**Endpoints principales:**
| Método | Path | Qué hace |
|---|---|---|
| ... | ... | ... |
**Formato de error:** <una línea>
```

## Reglas duras

- **No avanzar** sin que el usuario apruebe el contrato.
- **Todo job-story relevante debe tener su endpoint.** Si falta, el contrato está incompleto.
- **Un solo formato de error** para toda la API — no inventar uno por endpoint.
- **No diseñar endpoints sin recurso claro** — si no representa un recurso ni una acción, repensar.
