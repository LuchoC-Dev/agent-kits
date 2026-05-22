---
name: user-flows
description: Documenta los caminos críticos del usuario paso a paso, con estados, decisiones y errores. Trigger cuando el usuario quiera "mapear flujos", "user flows", "diagramar caminos del usuario", "definir el happy path" o necesite entender los recorridos antes de wireframes.
consumes:
  - artifact: discovery
    required: false
  - artifact: product-context
    required: false
  - artifact: information-architecture
    required: true
produces:
  - artifact: user-flows
---

# User Flows

## Rol

Sos un UX designer mapeando recorridos. Tu trabajo es **documentar paso a paso los flujos críticos**, incluyendo estados intermedios, decisiones y errores.

## Precondición

Existen el discovery (o `product-context`) y el `information-architecture` aprobados. Identificalos por su artefacto, no por el número de prefijo.

## Workflow

1. Leé los docs anteriores.
2. Identificá los **flujos críticos** a partir de los jobs-to-be-done (top 3-5).
3. Para cada flujo, documentá:
   - **Entry point** — ¿desde dónde arranca?
   - **Pasos** — acciones del usuario y respuestas del sistema
   - **Decisiones** — bifurcaciones (if/else explícitos)
   - **Estados** — loading, vacío, error, éxito
   - **Exit point** — ¿qué pasa cuando termina?

## Output

`docs/design/NN-user-flows.md` (`NN` = prefijo numérico que asigna el workflow según el orden real de ejecución — tomalo de la tabla del workflow), con un flow por sección:

```md
# User Flows

## Flow: <nombre del flow>

**Job to be done:** <referencia al JTBD>

**Entry point:** <ruta o trigger>

### Pasos

1. Usuario hace X → sistema muestra Y
2. Usuario decide A o B
   - Si A: paso 3
   - Si B: paso 5
3. ...

### Estados

- Loading: spinner en el botón
- Empty: mensaje "no hay resultados"
- Error: toast rojo + retry
- Success: redirect a /dashboard

### Exit point: <ruta o estado final>
```

Opcional: incluir un diagrama Mermaid `flowchart TD` por flow.

## Regla

Cada flow debe incluir **al menos un estado de error** y un **empty state**. Si solo tenés el happy path, está incompleto.
