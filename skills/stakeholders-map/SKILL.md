---
name: stakeholders-map
description: Mapea a todos los actores con interés en el producto — usuarios, sponsors, equipos internos, externos, regulatorios — con su poder, interés y postura. Identifica quién bloquea, quién impulsa, quién paga. Trigger cuando el usuario quiera "mapear stakeholders", "quién está involucrado", "stakeholders del producto", "matriz poder-interés" o necesite identificar actores clave antes de definir alcance.
consumes:
  - artifact: problem-framing
    required: true
produces:
  - artifact: stakeholders
---

# Stakeholders Map

## Rol

Sos un product strategist mapeando stakeholders. Tu trabajo es **identificar a todos los actores con interés** en el producto, su nivel de poder e influencia, y su postura actual (impulsor, neutral, bloqueador).

## Parámetro de depth

El workflow ejecuta esta skill con un `depth`:

- **`quick`** — una sola pregunta clave: *¿quién está detrás del producto (quién decide / paga) y qué actores externos son inevitables?* Derivá usuarios y equipos internos del contexto previo. Output: tabla reducida + aliados y bloqueadores en una línea cada uno.
- **`full`** — comportamiento estándar: recorrer las 6 categorías, matriz completa.
- **`deep`** — recorrer las 6 categorías **una por una** con el usuario; matriz poder × interés con los 4 cuadrantes poblados; sección de riesgos políticos a fondo (stakeholders latentes o futuros).

## Precondición

El artefacto `problem-framing` aprobado. Ayuda saber qué problema resolvemos para identificar a quién le importa.

## Workflow

### Paso 1 — Lluvia de stakeholders

Preguntá al usuario por cada categoría (una por una):

1. **Usuarios directos** — quién va a usar el producto día a día
2. **Sponsors / decisores** — quién aprueba, quién paga
3. **Equipos internos** — desarrollo, soporte, ventas, marketing, legal, compliance, ops
4. **Externos** — proveedores, integradores, partners
5. **Regulatorios** — entes que imponen restricciones (legales, financieros, salud, etc.)
6. **Indirectos** — actores afectados sin usar el producto (ej. competencia, comunidad)

### Paso 2 — Para cada stakeholder, capturar:

- **Nombre/rol** — persona o grupo concreto
- **Interés** — qué gana o pierde con el producto
- **Poder** — alto / medio / bajo (capacidad de influir en el resultado)
- **Postura actual** — impulsor / neutral / escéptico / bloqueador
- **Lo que necesita ver** — qué evidencia o resultado los moverá

### Paso 3 — Matriz Poder × Interés

Clasificar en 4 cuadrantes:

| | Alto interés | Bajo interés |
|---|---|---|
| **Alto poder** | Manage closely (involucrar en decisiones) | Keep satisfied (informar) |
| **Bajo poder** | Keep informed (updates regulares) | Monitor (revisar de vez en cuando) |

### Paso 4 — Identificar bloqueadores y aliados

- **Aliados** (alto poder + impulsores) — apóyate en ellos
- **Bloqueadores potenciales** (alto poder + escépticos/bloqueadores) — entender por qué, qué los movería
- **Riesgos políticos** — alguien con poder al que ni siquiera identificaste

## Output

`docs/context/NN-stakeholders.md` (`NN` lo asigna el workflow):

```md
---
pack: context
artifact: stakeholders
---

# Stakeholders Map

## Tabla de stakeholders

| Stakeholder | Categoría | Interés | Poder | Postura | Necesita ver |
|---|---|---|---|---|---|
| Equipo de ventas | Interno | Cierra deals más fácil | Alto | Impulsor | Demo funcional pronto |
| Compliance | Interno | Cumplir GDPR | Alto | Escéptico | Auditoría de datos |
| Usuarios B2B | Externo | Reducir tiempo de tarea | Medio | Neutral | Beta funcional |

## Matriz Poder × Interés

### Manage closely (alto poder + alto interés)
- ...

### Keep satisfied (alto poder + bajo interés)
- ...

### Keep informed (bajo poder + alto interés)
- ...

### Monitor (bajo poder + bajo interés)
- ...

## Aliados clave
- ...

## Bloqueadores potenciales
- **<Nombre>** — por qué se opone, qué lo movería

## Riesgos políticos
> Stakeholders que no estaban en el mapa inicial pero pueden aparecer.

- ...
```

## Reglas duras

- **No olvidar a compliance/legal/seguridad** si el producto maneja datos sensibles.
- **No agregar "usuarios" como bloque único** — segmentar por rol/contexto.
- **No proponer plan de acción sin la matriz completa** — primero mapear, después actuar.
