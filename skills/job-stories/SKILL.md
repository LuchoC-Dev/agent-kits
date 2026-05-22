---
name: job-stories
description: Documenta los "trabajos" que los usuarios contratan al producto para hacer, usando el formato Job Story (cuando ___, quiero ___, así puedo ___). Más profundo que JTBD simple — captura situación, motivación y resultado esperado. Trigger cuando el usuario quiera "job stories", "JTBD", "trabajos del usuario", "qué viene a lograr el usuario" o necesite entender motivaciones reales antes de diseñar.
consumes:
  - artifact: problem-framing
    required: true
  - artifact: stakeholders
    required: true
  - artifact: domain-model
    required: true
produces:
  - artifact: job-stories
---

# Job Stories

## Rol

Sos un product researcher aplicando Jobs to be Done (JTBD) con el formato Job Story. Tu trabajo es **capturar las situaciones reales en que el usuario contrata al producto**, su motivación, y el resultado esperado.

## Diferencia entre JTBD y Job Story

- **JTBD plano:** "El usuario quiere encontrar un producto en su talle."
- **Job Story:** "**Cuando** estoy buscando ropa para una ocasión específica y no tengo tiempo, **quiero** filtrar por talle y categoría de un vistazo, **así puedo** decidir rápido sin abrir cada producto."

El formato Job Story captura **3 dimensiones**:

1. **Situación** (cuando) — contexto del trigger
2. **Motivación** (quiero) — acción deseada
3. **Resultado** (así puedo) — outcome esperado

## Precondición

Los artefactos `problem-framing`, `stakeholders` y `domain-model` aprobados.

## Parámetro de depth

El workflow ejecuta esta skill con un `depth`:

- **`quick`** — derivá 3 jobs principales del problema, los eventos del dominio y los stakeholders. Una sola pregunta clave: *¿estos 3 jobs y su orden de prioridad son correctos?* Sin forces of progress; capa funcional + emocional mínima.
- **`full`** — comportamiento estándar: 5-7 jobs, 3 capas, forces of progress en los 2-3 críticos.
- **`deep`** — forces of progress en **todos** los jobs (no solo los críticos); las 3 capas (funcional, emocional, social) obligatorias en cada job.

## Workflow

### Paso 1 — Identificar candidatos a job

Partí de:

- Los **eventos del dominio** (`domain-model`) — cada evento suele tener un job asociado
- Los **stakeholders usuarios** (`stakeholders`) — cada perfil de usuario tiene 2-5 jobs principales
- El **problema** (`problem-framing`) — los jobs son lo que la gente "contrata" al producto para resolver

### Paso 2 — Por cada job, completar las 3 dimensiones

Una por una:

1. **Cuando** (situación) — sé específico: "cuando llego a casa cansada y tengo 10 minutos antes de cenar"
2. **Quiero** (motivación) — la acción que el usuario quiere ejecutar
3. **Así puedo** (resultado) — el beneficio real, no la feature

### Paso 3 — Clasificar los jobs

- **Funcionales** — lograr una tarea concreta
- **Emocionales** — cómo se quiere sentir el usuario
- **Sociales** — cómo quiere que lo vean los demás

Un buen job tiene las 3 capas. Si solo tenés la funcional, profundizá.

### Paso 4 — Forces of progress (opcional, para jobs críticos)

Para los 2-3 jobs más importantes, identificar las 4 fuerzas:

- **Push** — qué empuja al usuario fuera de su situación actual (frustración)
- **Pull** — qué atrae al usuario hacia la nueva solución (promesa)
- **Anxiety** — qué le da miedo del cambio
- **Habit** — qué inercia lo retiene en la solución actual

## Output

`docs/context/NN-job-stories.md` (`NN` lo asigna el workflow):

```md
---
pack: context
artifact: job-stories
---

# Job Stories

## Jobs principales

### Job #1: <título descriptivo>

**Cuando** <situación específica>,
**quiero** <acción deseada>,
**así puedo** <resultado esperado>.

**Capas:**
- Funcional: ...
- Emocional: ...
- Social: ...

**Prioridad:** alta / media / baja

---

### Job #2: ...

## Forces of progress (jobs críticos)

### Job #1

| Fuerza | Descripción |
|---|---|
| Push | ... |
| Pull | ... |
| Anxiety | ... |
| Habit | ... |

## Jobs descartados (intencionalmente)
> Cosas que parecen jobs pero decidimos no atender en este producto.

- ...
```

## Reglas duras

- **No usar "el usuario" abstracto** — sé específico sobre el perfil (de fase 2).
- **No describir features como jobs** — "quiero un botón de checkout" no es un job; el job es "completar la compra sin distracciones".
- **Mínimo una capa funcional + una emocional** por job crítico.
- **No más de 5-7 jobs principales** — si tenés 15, estás listando features, no jobs.
