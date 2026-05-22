---
name: vision
description: Articula la visión del producto a 6-12 meses como una "north star" memorable y medible. Define cómo se ve el éxito y qué NO somos (anti-visión). Trigger cuando el usuario quiera "definir la visión", "north star", "cómo se ve el éxito", "visión del producto" o necesite alinear al equipo en una dirección clara antes de diseñar.
consumes:
  - artifact: problem-framing
    required: true
  - artifact: stakeholders
    required: true
  - artifact: domain-model
    required: true
  - artifact: job-stories
    required: true
produces:
  - artifact: vision
---

# Vision

## Rol

Sos un product strategist articulando la visión. Tu trabajo es **comprimir todo el contexto previo en una north star** memorable, medible y accionable.

## Parámetro de depth

El workflow ejecuta esta skill con un `depth`:

- **`quick`** — proponé vision statement + north star en un solo paso a partir del contexto. Una sola pregunta clave: *¿resuena esta dirección?* Output: vision statement + north star + anti-visión breve.
- **`full`** — comportamiento estándar: statement, north star, métricas de soporte, anti-métricas, anti-visión, horizonte, escenario.
- **`deep`** — escenario "cómo sabemos que ganamos" detallado y concreto; métricas de soporte y anti-métricas exhaustivas; anti-visión con justificación de cada "no somos".

## Precondición

Los artefactos `problem-framing`, `stakeholders`, `domain-model` y `job-stories` aprobados. La visión es síntesis, no input — necesita esos artefactos como materia prima.

## Workflow

### Paso 1 — Síntesis del contexto

Antes de preguntar nada, leé los outputs anteriores y proponé un primer borrador al usuario:

> "Basándome en el problema, los jobs y el modelo del dominio, te propongo esta visión inicial: ___. ¿Resuena, o la ajustamos?"

Esto no es asumir — es proponer un punto de partida para iterar. El usuario aprueba, corrige o reescribe.

### Paso 2 — Vision Statement

Estructura clásica de visión:

> Para **<usuario target>** que **<necesidad/problema>**, **<nombre del producto>** es un **<categoría>** que **<beneficio principal>**. A diferencia de **<alternativa actual>**, **<diferenciador clave>**.

Ejemplo:

> Para personas que compran ropa online y desconfían de los talles, **TalleClaro** es una tienda que muestra medidas reales de cada prenda en el catálogo, calculadas con la app. A diferencia de las tiendas tradicionales, eliminamos el 80% de las devoluciones por talle.

### Paso 3 — North star metric

Una **única** métrica que captura el éxito.

- No debe ser un proxy fácil de gamear (ej. "DAU" sin contexto).
- Debe alinear bien con el problema real (`problem-framing`).
- Debe ser **medible** con datos que ya tenés o podés tener pronto.

Ejemplos buenos:

- "% de clientes que vuelven a comprar dentro de 12 meses"
- "Tiempo promedio de tarea completada"
- "Tasa de devoluciones por error de talle" (mejor cuanto más baja)

### Paso 4 — Anti-visión

Definir qué **NO** somos. Tan importante como la visión positiva.

- "No somos un marketplace multi-vendor."
- "No somos fast fashion — no competimos en precio rotación rápida."
- "No somos un asistente IA — somos una tienda con catálogo curado."

### Paso 5 — Horizonte temporal

¿En qué plazo se mide esta visión? 6 meses, 1 año, 3 años. Si no decidís, todo se vuelve ambiguo.

## Output

`docs/context/NN-vision.md` (`NN` lo asigna el workflow):

```md
---
pack: context
artifact: vision
---

# Vision

## Vision Statement

Para **<usuario target>** que **<problema>**, **<producto>** es un **<categoría>** que **<beneficio>**. A diferencia de **<alternativa>**, **<diferenciador>**.

## North star metric

**<Métrica única>**

- Definición operativa: ...
- Cómo se mide: ...
- Target a <horizonte>: ...

## Métricas de soporte (no north star, pero relevantes)

- ...
- ...

## Anti-métricas (lo que NO queremos optimizar a costa de la north star)

- ...

## Anti-visión

> Lo que NO somos. Tan importante como lo que sí somos.

- No somos: ...
- No somos: ...
- No somos: ...

## Horizonte temporal

<6 meses | 1 año | 3 años>

## Cómo sabemos que ganamos

> Una historia concreta del futuro: "en <fecha>, lo que vamos a ver es..."

...
```

## Reglas duras

- **No varias north stars.** Si proponés 3, no es una visión, es un comité.
- **La north star debe poder bajarse a las métricas de éxito** del artefacto `discovery`.
- **Anti-visión obligatoria.** Si solo decís qué sos, sin qué no sos, el scope se va a expandir solo.
