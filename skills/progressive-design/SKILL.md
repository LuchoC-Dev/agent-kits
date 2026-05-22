---
name: progressive-design
description: Orquesta un proceso progresivo wireframes ASCII → html-mockup → ajustes iterativos. Usar cuando el usuario quiera "ir construyendo el diseño de a poco", "wireframe primero y después HTML", "design progresivo" o necesite validar estructura antes de invertir en mockups visuales.
consumes:
  - artifact: component-inventory
    required: true
  - artifact: design-tokens
    required: true
produces:
  - artifact: wireframes
  - artifact: html-mockups
  - artifact: progressive-summary
---

# Progressive Design

## Rol

Sos un design lead orquestando un proceso **progresivo**: arrancás con un wireframe ASCII barato, lo validás, lo elevás a HTML mid-fi, lo validás de nuevo, y solo si hace falta, subís pantallas críticas a hi-fi. Cada paso es una decisión consciente, no un automatismo.

## Cuándo usar esta skill

Cuando el usuario quiere **no comprometer fidelidad de entrada** y prefiere ir descubriendo qué pantallas merecen más inversión visual.

| Alternativas | Cuándo | Costo |
|---|---|---|
| `wireframes` (solo ASCII) | Apps chicas, validación rápida | Más barato |
| `html-mockup` (solo HTML) | Estructura ya 100% clara | Más caro upfront |
| `progressive-design` | Querés iterar y descubrir | Costo controlado paso a paso |

## Precondición

- El `component-inventory` aprobado.
- El `design-tokens` aprobado (lo vas a usar en el paso 2).

## Workflow interno

### Paso 1 — Wireframes ASCII (invoca `wireframes` con depth full)

1. Ejecutá la skill `wireframes` para todas las pantallas.
2. Output en `docs/design/NN-wireframes.md` (`NN` lo asigna el workflow).
3. **Gate de aprobación** — mostrale al usuario los wireframes y pedí aprobación de estructura.
4. Iterá hasta que el usuario apruebe la estructura.

> **Por qué empezar acá:** validar jerarquía y disposición es barato en ASCII. Si en este paso descubrimos que la nav está mal, no perdimos tokens HTML.

### Paso 2 — Mid-fi HTML (invoca `html-mockup` con fidelity `mid-fi-tokens`)

Una vez aprobados los wireframes:

1. Preguntale al usuario:
   > "Wireframes aprobados. ¿Querés elevar todas las pantallas a mid-fi-tokens HTML, o solo un subset?"
2. Ejecutá `html-mockup` con `fidelity: mid-fi-tokens` para las pantallas seleccionadas.
3. Output en `docs/design/mockups/<screen>.html`.
4. **Gate de aprobación** — el usuario abre los mockups, da feedback **por pantalla**.

### Paso 3 — Loop de ajustes mid-fi

Mientras el usuario tenga feedback:

1. Tomá el feedback (ej. "el header está muy cargado", "el CTA tiene que ser más visible").
2. Regenerá solo las pantallas afectadas.
3. Pedí re-aprobación.

**Salir del loop cuando:**

- El usuario aprueba explícitamente, o
- Se llega a 3 iteraciones y el usuario quiere pausar.

### Paso 4 — Selectivo hi-fi (opcional, invoca `html-mockup` con fidelity `hi-fi`)

Una vez aprobados los mid-fi:

1. Preguntale al usuario:
   > "Mid-fi aprobado. ¿Hay pantallas críticas que quieras subir a hi-fi (con estados, ARIA, responsive completo)?
   > Recomendación: las 2-3 más críticas del producto (las que aparecen en los JTBD top del discovery)."
2. Si elige al menos una: ejecutá `html-mockup` con `fidelity: hi-fi` solo para esas.
3. Si no quiere ninguna: saltar al paso 5.

### Paso 5 — Cierre

1. Generar/actualizar `docs/design/mockups/index.html` con links a todos los mockups, marcando la fidelity de cada uno.
2. Generar `docs/design/NN-progressive-summary.md` (mismo `NN` que el de la fase visual) con el resumen del proceso:
   - Pantallas en ASCII (link al wireframe)
   - Pantallas en mid-fi-tokens (link al HTML)
   - Pantallas en hi-fi (link al HTML)
   - Decisiones tomadas en cada iteración

## Output

```
docs/design/
├── NN-wireframes.md            (paso 1)
├── NN-progressive-summary.md   (paso 5 — resumen del proceso)
└── mockups/
    ├── index.html              (paso 5 — overview con fidelity por pantalla)
    ├── home.html               (mid-fi-tokens o hi-fi)
    ├── pdp.html                (hi-fi típicamente)
    └── ...
```

## Reglas duras

- **No saltar al paso 2 sin aprobar el paso 1.** El ASCII es la base.
- **No subir a hi-fi todo.** Si el usuario quiere todas las pantallas en hi-fi, recordale el costo y sugerí que se hagan post-implementación (Figma/Penpot).
- **Loop de ajustes con límite.** Si después de 3 iteraciones no hay aprobación, pausar y volver a revisar el wireframe — quizá el problema está en la estructura, no en el visual.

## Estado del proceso

Mantené el estado en `docs/design/.progressive-state.json`:

```json
{
  "current_step": "mid-fi-loop",
  "screens": {
    "home": { "fidelity": "mid-fi-tokens", "iterations": 2, "approved": true },
    "pdp": { "fidelity": "hi-fi", "iterations": 1, "approved": false }
  },
  "last_updated": "2026-05-19T15:00:00Z"
}
```
