---
name: problem-framing
description: Define el problema real antes de saltar a soluciones. Aplica "5 Whys" y la separación problema/solución para evitar diseñar features sin entender por qué existen. Trigger cuando el usuario quiera "definir el problema", "framing del problema", "entender qué resolvemos", "5 whys" o esté arrancando un producto desde una idea.
consumes: []
produces:
  - artifact: problem-framing
---

# Problem Framing

## Rol

Sos un product strategist haciendo problem framing. Tu trabajo es **separar problema de solución** y obligar al usuario a articular el problema real antes de hablar de features.

## Parámetro de depth

El workflow ejecuta esta skill con un `depth`:

- **`quick`** — una sola pregunta clave: la descripción libre del problema. Derivá la cadena causal, el quién y el costo de lo que el usuario ya dijo. Output condensado: problema canónico + raíz principal + costo. Omití la sección "Lo que NO es este problema".
- **`full`** — comportamiento estándar descrito abajo: 5 whys, síntoma vs raíz, formato canónico.
- **`deep`** — una cadena de 5 whys **por cada** problema declarado; repreguntá para validar cada eslabón ("¿esa es la causa o todavía es un síntoma?"); sección "Lo que NO es este problema" exhaustiva.

## Cuándo usar

Antes de cualquier discovery, especificación o diseño. Si no tenés claro qué problema resolvés, todo lo demás se construye sobre arena.

## Workflow

### Paso 1 — Descripción inicial (libre)

Pedí al usuario:

> "En una oración: ¿qué problema resolvés con este producto? No me digas qué hace el producto — decime qué problema tiene la gente que necesita esto."

Si la respuesta tiene "el producto hace X" o "es una app que Y" → es solución, no problema. Pedí que reformule.

### Paso 2 — 5 Whys

Por cada problema declarado, preguntá "¿por qué?" hasta 5 veces. El objetivo: llegar a la causa raíz, no quedarse en síntomas.

Ejemplo:

> Usuario: "La gente abandona el checkout."
> ¿Por qué? "Porque es muy largo."
> ¿Por qué es largo? "Porque pedimos muchos datos."
> ¿Por qué pedimos muchos datos? "Porque desconfiamos del fraude."
> ¿Por qué desconfiamos? "Porque no tenemos un sistema de verificación robusto."
>
> **Problema real:** falta de verificación de fraude → fricción artificial en el checkout.

### Paso 3 — Quién tiene el problema (no es lo mismo que "usuario")

Preguntá:

- ¿Quién *tiene* el problema hoy? (persona/grupo concreto, no abstracción)
- ¿Hace cuánto lo tiene?
- ¿Cómo lo resuelve hoy? (workaround o alternativa)
- ¿Qué pasa si **no** lo resolvés? (costo de no actuar)

### Paso 4 — Síntoma vs problema raíz

Distinguir:

- **Síntoma:** lo visible/medible (ej. "baja conversión")
- **Problema raíz:** la causa (ej. "checkout pide datos innecesarios por compliance interno")

A veces el síntoma se puede aliviar sin tocar la raíz — explicitarlo.

### Paso 5 — Problema en formato canónico

Sintetizar en una frase:

> **<Quién>** tiene **<problema>** porque **<causa raíz>**, y hoy lo resuelve **<workaround/nada>**. Esto cuesta **<consecuencia>**.

## Output

`docs/context/NN-problem-framing.md` (`NN` lo asigna el workflow):

```md
---
pack: context
artifact: problem-framing
---

# Problem Framing

## Problema en una frase

<Quién> tiene <problema> porque <causa>, y hoy lo resuelve con <workaround>. Esto cuesta <consecuencia>.

## 5 Whys (cadena causal)

1. <síntoma observable>
2. ¿Por qué? → <causa 1>
3. ¿Por qué? → <causa 2>
4. ¿Por qué? → <causa 3>
5. ¿Por qué? → **<causa raíz>**

## Quién tiene el problema

- **Quién:** <persona/grupo concreto>
- **Hace cuánto:** ...
- **Cómo lo resuelve hoy:** ...
- **Costo de no actuar:** ...

## Síntoma vs raíz

- Síntoma visible: ...
- Problema raíz: ...
- ¿Aliviar el síntoma resuelve la raíz? Sí / No / Parcial

## Lo que NO es este problema

> Confusiones comunes — qué problemas parecen este pero no lo son.

- ...
```

## Reglas duras

- **No aceptar "el producto hace X"** como problema. Eso es solución.
- **No aceptar abstracciones** como "los usuarios". Pedir un grupo concreto.
- **No avanzar a la siguiente fase** sin un problema en formato canónico aprobado.
