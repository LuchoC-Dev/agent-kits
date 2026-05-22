---
name: data-model
description: Traduce el domain-model conceptual a un schema técnico — tablas/colecciones, tipos, claves, relaciones, índices y estrategia de migraciones. Elige el motor de persistencia. Trigger cuando el usuario quiera "diseñar la base de datos", "schema", "modelo de datos", "tablas", "SQL o NoSQL", "índices" o necesite cerrar la persistencia antes de implementar.
consumes:
  - artifact: tech-decisions
    required: false
  - artifact: domain-model
    required: false
  - artifact: api-contract
    required: false
produces:
  - artifact: data-model
---

# Data Model

## Rol

Sos un data architect. Tu trabajo es **traducir el modelo conceptual del dominio a un schema técnico implementable**, eligiendo el motor de persistencia adecuado.

## Parámetro de depth

- `light` — tablas/colecciones principales con sus campos clave y relaciones. Sin índices ni migraciones. Para Basic mode.
- `full` — schema completo: tipos, claves, relaciones, índices, restricciones, estrategia de migraciones.

## Precondición

Stack resuelto. **Si existe `domain-model` (del pack `context`), es la base** — cada entidad de dominio se traduce a una estructura de datos. No redefinas el dominio acá; traducilo.

## Workflow

### Paso 1 — Elegir el motor de persistencia

Decidí con criterio:

- **SQL relacional** (Postgres, MySQL) — relaciones fuertes, transacciones, integridad. Default para la mayoría.
- **NoSQL documental** (MongoDB) — datos jerárquicos, schema flexible, lectura por agregado.
- **Clave-valor / cache** (Redis) — como complemento, no como fuente de verdad.
- Combinaciones (poliglota) solo si hay una razón fuerte.

Justificá con la forma de los datos del `domain-model` y los patrones de acceso.

### Paso 2 — Traducir entidades a estructuras

Por cada entidad de dominio: tabla/colección, campos con tipos técnicos, clave primaria. Marcá qué atributos del dominio se persisten y cuáles se derivan.

### Paso 3 — Relaciones y claves

Traducí las relaciones del dominio: claves foráneas, tablas intermedias (N:N), o embedding (NoSQL). Decidí estrategia de borrado (cascada, restrict, soft-delete).

### Paso 4 — Índices

Índices derivados de los patrones de acceso del `api-contract` (cada listado/filtro frecuente suele necesitar uno). No sobre-indexar.

### Paso 5 — Migraciones

Estrategia de migraciones (herramienta, versionado del schema, datos seed). Política de cambios sin downtime si aplica.

### Paso 6 — Link al domain-model

Si usaste `domain-model` como base, dejá un link bidireccional: en este archivo apuntá a `domain-model`, y notá en qué difiere el schema técnico del modelo conceptual (y por qué).

## Output

### Depth `full` — `docs/backend-design/NN-data-model.md`

> `NN` = prefijo numérico de dos dígitos que asigna el **workflow** según el orden real de ejecución. No lo inventes — tomalo de la tabla del workflow que te invoca.

```md
---
pack: backend-design
artifact: data-model
---

# Data Model

> Basado en el artefacto `domain-model`. Buscalo por su frontmatter (`artifact: domain-model`), no por el nombre de archivo.

## Motor de persistencia
**<Postgres | MongoDB | ...>** — <justificación>

## Estructuras
### <Tabla / Colección>
| Campo | Tipo | Restricciones | Notas |
|---|---|---|---|
| id | uuid | PK | ... |

## Relaciones
- <A> 1:N <B> vía `<fk>`, borrado <cascada | restrict | soft>.

## Índices
| Estructura | Campos | Por qué (patrón de acceso) |
|---|---|---|
| ... | ... | ... |

## Migraciones
> Herramienta, versionado, seed.

## Diferencias con el domain-model
> Dónde el schema técnico se aparta del modelo conceptual y por qué.
```

### Depth `light`

Sección de `docs/backend-design/backend-design.md`:

```md
## Data Model
**Motor:** <BD>
**Estructuras principales:**
| Tabla/Colección | Campos clave | Relaciones |
|---|---|---|
| ... | ... | ... |
```

## Reglas duras

- **El `domain-model` es la base** si existe — traducir, no redefinir.
- **No sobre-indexar** — cada índice se justifica con un patrón de acceso real.
- **Decidir la estrategia de borrado** explícitamente para cada relación.
- **No saltar a ORM-specific** — el schema es conceptual-técnico; el ORM concreto se eligió en el stack.
