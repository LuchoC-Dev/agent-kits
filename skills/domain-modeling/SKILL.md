---
name: domain-modeling
description: Construye el modelo del dominio — entidades clave, relaciones, eventos y lenguaje ubicuo (DDD estratégico ligero). Genera un glosario que todos los stakeholders usan con el mismo significado. Trigger cuando el usuario quiera "modelar el dominio", "domain-driven design", "ubiquitous language", "event storming", "entidades del sistema" o necesite alinear vocabulario antes de definir features.
consumes:
  - artifact: problem-framing
    required: true
  - artifact: stakeholders
    required: true
produces:
  - artifact: domain-model
---

# Domain Modeling

## Rol

Sos un domain expert aplicando DDD estratégico (versión ligera). Tu trabajo es **identificar las entidades, relaciones y eventos clave del dominio**, y construir el lenguaje ubicuo (ubiquitous language) que todos van a usar para hablar del producto.

## Parámetro de depth

El workflow ejecuta esta skill con un `depth`:

- **`quick`** — una sola pregunta clave: *¿cuáles son los conceptos de negocio centrales?* Derivá relaciones, eventos y glosario de esas entidades y del contexto previo. Output: entidades core + glosario, sin bounded contexts.
- **`full`** — comportamiento estándar: entidades, relaciones, eventos, bounded contexts si aplica, glosario.
- **`deep`** — event storming completo (todos los eventos del dominio, no solo los obvios); bounded contexts **obligatorios** con sus entidades y eventos; análisis exhaustivo de términos ambiguos.

## Precondición

Los artefactos `problem-framing` y `stakeholders` aprobados. El dominio se mapea mejor cuando ya sabés qué problema resolvés y para quién.

## Workflow

### Paso 1 — Identificar entidades

Preguntá al usuario:

> "Si tuvieras que describir tu sistema en sustantivos importantes, ¿cuáles serían? No me digas tablas — decime los conceptos del negocio."

Apuntá los sustantivos. Filtrá los genéricos (Usuario, Sistema). Buscá los específicos del dominio (Pedido, Suscripción, Diagnóstico, Reserva, etc.).

### Paso 2 — Relaciones entre entidades

Para las 5-10 entidades core, definí cómo se relacionan:

- Un **<A>** tiene uno/muchos **<B>**
- Un **<A>** pertenece a un **<B>**
- Un **<A>** se convierte en **<B>** cuando...

### Paso 3 — Eventos del dominio (event storming ligero)

Pedí al usuario que liste los **eventos** que ocurren en el sistema en pasado:

- "Pedido creado"
- "Pago confirmado"
- "Envío despachado"
- "Suscripción cancelada"

Los eventos describen **qué cambia** en el sistema y son anchors muy útiles para el diseño posterior.

### Paso 4 — Bounded contexts (si aplica)

Si el producto es lo suficientemente grande, identificar contextos delimitados:

- Contexto de **Catálogo** (productos, categorías, búsqueda)
- Contexto de **Compra** (carrito, checkout, pago)
- Contexto de **Fulfillment** (envío, devoluciones)
- Contexto de **Cuenta** (perfil, direcciones, historial)

Cada contexto puede tener entidades con el mismo nombre pero significado distinto (ej. "Producto" en Catálogo vs en Fulfillment).

### Paso 5 — Glosario / Ubiquitous Language

Construir una tabla de términos del dominio con definición canónica.

> Regla: cada término aparece **una vez** con **una definición**. Si dos stakeholders usan el mismo término con significado distinto, eso es un problema del modelo, no algo a documentar.

## Output

`docs/context/NN-domain-model.md` (`NN` lo asigna el workflow):

```md
---
pack: context
artifact: domain-model
---

# Domain Model

## Entidades core

### <Entidad>
- **Qué representa:** ...
- **Atributos clave:** ...
- **Estados posibles:** <enum si aplica>

## Relaciones

- Un **Pedido** tiene muchos **Items**.
- Un **Item** pertenece a un **Producto** (snapshot del precio al momento de la compra).
- Un **Pedido** se convierte en **Envío** cuando se confirma el pago.

## Eventos del dominio

| Evento | Disparador | Consecuencias |
|---|---|---|
| Pedido creado | Usuario completa checkout | Reserva stock, dispara cobro |
| Pago confirmado | Gateway notifica éxito | Dispara fulfillment |
| Envío despachado | Operaciones marca como enviado | Notifica al usuario |

## Bounded contexts

### Catálogo
- Entidades: Producto, Categoría, Marca
- Eventos: producto creado, stock actualizado

### Compra
- Entidades: Carrito, Pedido, Item, Pago
- Eventos: pedido creado, pago confirmado

## Ubiquitous Language

| Término | Definición canónica |
|---|---|
| Pedido | Compra confirmada por el usuario, antes de envío |
| Carrito | Selección no confirmada |
| Item | Línea de pedido con producto + cantidad + precio snapshot |
| Stock | Unidades disponibles para venta inmediata |

## Términos ambiguos detectados
> Términos que distintos stakeholders usan con sentidos distintos — resolver antes de avanzar.

- ...
```

## Reglas duras

- **No usar nombres técnicos** (UserModel, OrderTable) — usar el lenguaje del negocio.
- **No documentar términos con definiciones contradictorias** — eso es un problema, no documentación.
- **No saltar a esquemas de base de datos** — esto es modelado de dominio, no diseño técnico.
