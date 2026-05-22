---
name: assumptions-risks
description: Identifica las apuestas (assumptions/hypotheses) y los riesgos (unknowns/threats) del producto. Clasifica por criticidad y propone forma de validación temprana. Trigger cuando el usuario quiera "listar supuestos", "riesgos del proyecto", "qué estamos asumiendo", "hipótesis", "unknowns" o necesite mapear bets y amenazas antes de implementar.
consumes:
  - artifact: problem-framing
    required: true
  - artifact: stakeholders
    required: true
  - artifact: domain-model
    required: true
  - artifact: job-stories
    required: true
  - artifact: vision
    required: true
produces:
  - artifact: assumptions-risks
---

# Assumptions & Risks

## Rol

Sos un product strategist haciendo risk mapping. Tu trabajo es **explicitar lo implícito**: qué estamos apostando que es verdad, qué nos podría derrumbar el plan, y cómo validarlo barato antes de invertir.

## Parámetro de depth

El workflow ejecuta esta skill con un `depth`:

- **`quick`** — derivá del contexto el top 3 de leaps of faith y el top 3 de riesgos a mitigar, cada uno con su validación/mitigación. Sin matrices completas ni listados largos. Sin pregunta — es síntesis pura.
- **`full`** — comportamiento estándar: assumptions y riesgos completos, matrices de clasificación, top 3 de cada uno.
- **`deep`** — assumptions y riesgos exhaustivos por cada tipo; matrices completas; plan de mitigación **y** de contingencia para cada riesgo de alto impacto.

## Precondición

Los artefactos `problem-framing`, `stakeholders`, `domain-model`, `job-stories` y `vision` aprobados. Las assumptions y riesgos se identifican mejor cuando ya tenés visión y jobs claros — sabés contra qué se evalúan.

## Workflow

### Paso 1 — Listar assumptions (apuestas)

Una assumption es algo que **damos por cierto** sin haberlo verificado. Suelen estar implícitas en el problem-framing y la visión.

Tipos:

- **De deseabilidad** — "los usuarios quieren X"
- **De viabilidad** — "podemos construir esto en N meses con M personas"
- **De factibilidad** — "técnicamente es posible"
- **De negocio** — "esto genera retorno"

Por cada assumption:

1. Enunciado en frase declarativa
2. Por qué la creemos (evidencia actual)
3. Criticidad: si es **falsa**, ¿qué pasa? (alto / medio / bajo)
4. Confianza actual: alta / media / baja
5. Forma de validación más barata

### Paso 2 — Listar riesgos

Un riesgo es un evento posible que **dañaría el plan** si ocurriera.

Tipos:

- **Técnicos** — dependencias, performance, escalabilidad
- **De producto** — usuarios no adoptan, no hay product-market fit
- **De equipo** — capacidad, conocimiento, rotación
- **De negocio** — competencia, regulación, presupuesto
- **Externos** — proveedores, terceros, mercado

Por cada riesgo:

1. Descripción
2. Probabilidad: alta / media / baja
3. Impacto: alto / medio / bajo
4. Plan de mitigación
5. Plan de contingencia (si ocurre)

### Paso 3 — Matriz Criticidad × Confianza (para assumptions)

|  | Alta confianza | Baja confianza |
|---|---|---|
| **Crítica** | OK (monitorear) | **Validar YA** |
| **Poco crítica** | Ignorar | Monitorear |

Las assumptions del cuadrante "Validar YA" son **leap of faith** — se validan primero, sin ellas el resto del plan es construcción sobre arena.

### Paso 4 — Matriz Probabilidad × Impacto (para riesgos)

|  | Alto impacto | Bajo impacto |
|---|---|---|
| **Alta probabilidad** | **Mitigar YA** | Plan de contingencia |
| **Baja probabilidad** | Plan de contingencia | Aceptar |

## Output

`docs/context/NN-assumptions-risks.md` (`NN` lo asigna el workflow):

```md
---
pack: context
artifact: assumptions-risks
---

# Assumptions & Risks

## Assumptions

### Críticas — validar primero (leap of faith)

| Assumption | Por qué la creemos | Confianza | Cómo validar |
|---|---|---|---|
| Usuarios pagarán $X/mes | Encuesta n=20 | Media | Pre-venta 30 días |
| Stack soporta 10k req/s | Benchmarks de docs | Alta | Stress test propio |

### Menos críticas — monitorear

| Assumption | Confianza |
|---|---|
| ... | ... |

## Riesgos

### Cuadrante "Mitigar YA" (alta prob + alto impacto)

| Riesgo | Probabilidad | Impacto | Mitigación | Contingencia |
|---|---|---|---|---|
| Equipo de 2 personas no termina en 3 meses | Alta | Alto | Reducir scope a MVP | Re-priorizar y comunicar |

### Plan de contingencia (alto impacto, baja prob)

| Riesgo | Impacto | Si ocurre... |
|---|---|---|
| Regulador rechaza el modelo | Crítico | Pivot a B2B-only |

### Aceptados (baja prob + bajo impacto)

- ...

## Top 3 leap of faith

> Las 3 assumptions que, si son falsas, derrumban todo. Validar antes de invertir.

1. ...
2. ...
3. ...

## Top 3 riesgos a mitigar

1. ...
2. ...
3. ...
```

## Reglas duras

- **No listar assumptions con "alta confianza + poco crítica"** — son ruido.
- **Cada leap of faith debe tener un plan de validación concreto y barato.**
- **No aceptar "lo asumimos pero confiamos"** — si no hay forma de validar, marcarlo explícitamente como bet irreductible.
