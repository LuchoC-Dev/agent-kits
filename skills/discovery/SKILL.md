---
name: discovery
description: Define producto, usuarios, jobs-to-be-done y métricas de éxito. Arranca con un paso 0 (Intake) que recoge la descripción del producto del usuario y se adapta al contexto disponible, luego ejecuta preguntas adaptadas al dominio. Trigger cuando el usuario quiera "arrancar un diseño", "discovery", "entender qué construir", "definir el producto" o esté por iniciar cualquier proyecto sin claridad sobre qué problema resuelve.
consumes:
  - artifact: product-context
    required: false
produces:
  - artifact: discovery
---

# Discovery

## Rol

Sos un product designer haciendo discovery. Tu trabajo es **forzar claridad sobre qué se construye y por qué**, antes de cualquier decisión visual o técnica.

## Parámetro de depth

Esta skill acepta `depth`:

- `light` — un solo párrafo por sección, sin tablas, ideal para Basic mode.
- `full` — formato completo con tablas, métricas detalladas, no-goals.

## Paso 0 — Intake (adaptable al contexto)

**Antes** de hacer cualquier pregunta estructurada, ejecutá este paso.

### 0.1 Detectar contexto previo

Buscá insumos que ya existan, en este orden:

1. **Output del pack `context`.** Buscá en estos directorios candidatos, **en orden**, y quedate con el primero que matchee:

   1. `docs/context/`
   2. `context/`
   3. `docs/`

   En cada directorio, aceptá **cualquiera** de las dos formas de output del context pack:

   - **Forma separada** — los archivos numerados:
     - `01-problem-framing.md` → problema real, 5 whys, quién lo tiene
     - `02-stakeholders.md` → matriz poder × interés (los del cuadrante "alto interés" son tus usuarios target)
     - `03-domain-model.md` → entidades, eventos, **ubiquitous language** (usalo después en toda la skill)
     - `04-job-stories.md` → jobs en formato "cuando/quiero/así puedo" (mapealos directo a JTBD)
     - `05-vision.md` → north star metric (úsala como métrica norte) + anti-visión (úsala como no-goals)
     - `06-assumptions-risks.md` → "asumidos a confirmar" en tu output
   - **Forma consolidada** — un único `context.md` con todas las secciones (problema, stakeholders, dominio, jobs, visión, riesgos). El context pack lo emite cuando el contexto extraído es acotado.

   **Identificación robusta:** cada archivo del pack `context` lleva un frontmatter de identidad (`pack: context` + `artifact: <área>`). Identificá los archivos por ese frontmatter — no por el nombre. Así el handoff funciona aunque los nombres de archivo cambien. Si un archivo no tiene frontmatter, recién ahí caé al nombre.

   Si además existen `bdd/` o `spec/` dentro del directorio de contexto, usalos como contexto adicional formal.

   Si el contexto previo existe **completo** (separado o consolidado), prácticamente todo el discovery está pre-resuelto. Tu trabajo se reduce a: resumir, mostrar al usuario, pedir confirmación.

2. **`README.md`** del proyecto — descripción, propósito, stakeholders.
3. **Conversación reciente** con el usuario — si ya describió el producto en mensajes previos, no le pidas que lo repita.
4. **Argumentos pasados al workflow** — si el agente que te invoca te dio una descripción del producto, partí de ahí.

### 0.1.b Mapeo context → discovery

Si encontraste contexto previo, mapeá así. Si es la **forma separada**, el origen es el archivo indicado; si es la **forma consolidada** (`context.md`), el origen es la sección equivalente dentro de ese archivo.

| Sección de discovery | Origen en context |
|---|---|
| Producto | `01-problem-framing.md` (problema en frase canónica) + `05-vision.md` (vision statement) |
| Usuario principal | `02-stakeholders.md` (usuarios directos) + jobs de `04-job-stories.md` |
| Jobs to be done | `04-job-stories.md` (directo, top 3) |
| Pain points | `01-problem-framing.md` (5 whys, raíz) + `02-stakeholders.md` (qué pierde cada uno) |
| Métricas | `05-vision.md` (north star + soporte + anti-métricas) |
| Restricciones | `02-stakeholders.md` (compliance, legal) + `06-assumptions-risks.md` (riesgos críticos) |
| No-goals | `05-vision.md` (anti-visión) |
| Asumidos a confirmar | `06-assumptions-risks.md` (leaps of faith) |

Si hay contexto previo: **resumilo en una frase y pedí confirmación**:

> "Entiendo que querés construir <X>. ¿Es así o lo ajusto?"

### 0.2 Intake — descripción libre

Si **no** hay contexto previo suficiente, pedí al usuario:

> "Antes de las preguntas estructuradas: contame en tus palabras qué querés construir. Sin formato, lo que se te venga a la cabeza."

Esperá la descripción libre. **No avances** hasta tenerla.

### 0.3 Extraer y proponer

Una vez que tenés la descripción:

1. **Extraé** lo que el usuario ya dijo (producto, audiencia, motivación).
2. **Identificá** qué de las 7 preguntas estructuradas (más abajo) **ya está respondida**.
3. **Adaptá** las preguntas restantes al dominio. Ejemplos:
   - Si dijo "tienda de ropa": preguntá por tipo de catálogo, marcas, política de talles.
   - Si dijo "dashboard interno": preguntá por roles de usuario, fuente de datos, frecuencia de uso.
   - Si dijo "app de meditación": preguntá por sesiones, niveles, gamificación.
4. **Mostrale al usuario** qué ya entendiste y qué te falta:

   > "De lo que me contaste, ya tengo claro: <X>, <Y>. Me falta confirmar: <Z>."

## Preguntas estructuradas (adaptadas al dominio)

Estas son las **7 preguntas base**. Adaptá el lenguaje al dominio detectado en el Intake. **No las hagas todas juntas** — una por una.

1. **Producto en una frase** — confirmar / refinar la descripción del Intake.
2. **Usuario principal** — perfil, contexto, frecuencia. (En productos B2B: rol + tamaño de empresa.)
3. **Jobs to be done** — top 3 ordenados por importancia. (Adaptados al dominio.)
4. **Pain points actuales** — qué frustración resuelve, cómo lo resuelven hoy.
5. **Métricas de éxito** — qué métrica norte + 3-5 de soporte.
6. **Restricciones** — tiempo, equipo, stack obligatorio, branding existente, regulaciones.
7. **No-goals** — qué explícitamente **no** hace este producto.

## Output

> **Importante:** este documento es la **base de todas las fases siguientes**. Tiene que reflejar **todo lo que se conversó**, no solo el resumen estructurado. Si en el Intake el usuario dio detalles, contexto o matices, **incluilos** — no los descartes solo porque no caben en el template base.

### Depth `full`

`docs/design/NN-discovery.md` (`NN` = prefijo numérico que asigna el workflow según el orden real de ejecución — tomalo de la tabla del workflow):

```md
# Discovery

## Producto
<una frase + 1-2 párrafos de contexto si hace falta>

## Contexto inicial (extraído del Intake)
> Capturá acá los matices, motivaciones y detalles que el usuario compartió en la conversación libre.
> Citas textuales si son útiles. No es un resumen — es el material crudo del que parte el resto.

- <observación 1>
- <observación 2>
- ...

## Usuario principal
- **Perfil:** <demografía, rol, nivel técnico>
- **Contexto de uso:** <dónde, cuándo, con qué dispositivo, qué interrumpe>
- **Frecuencia:** <diaria, semanal, mensual, esporádica>
- **Variantes/segmentos:** <si hay más de un tipo de usuario, listarlos>

## Jobs to be done
> Top 3 ordenados por importancia. Cada uno con su contexto del por qué.

1. **<JTBD 1>** — <por qué importa>
2. **<JTBD 2>** — <por qué importa>
3. **<JTBD 3>** — <por qué importa>

## Pain points
> Frustraciones actuales y cómo las resuelven hoy.

- **<Pain 1>** — hoy lo resuelven con: <workaround>
- ...

## Métricas de éxito
**Métrica norte:** <una sola métrica que define éxito>

**Métricas de soporte:**
- <métrica 2>
- <métrica 3>
- ...

**Anti-métricas** (lo que NO queremos optimizar a costa de la métrica norte):
- ...

## Restricciones
- **Tiempo:** ...
- **Equipo:** ...
- **Stack obligatorio:** ...
- **Branding existente:** ...
- **Regulaciones:** ...
- **Otras:** ...

## No-goals
> Qué explícitamente NO hace este producto. Tan importante como los goals.

- ...

## Asumidos a confirmar
> Si extrajiste algo del Intake que el usuario no dijo explícitamente, listalo acá para confirmar en la próxima fase.

- ...
```

**Regla de riqueza:** si el documento final es más corto que la conversación que tuviste para producirlo, **está incompleto**. Volvé a leer los mensajes del Intake y completá lo que falta.

### Depth `light`

Una sección de `docs/design/design.md`:

```md
## Discovery

**Producto:** <una frase>
**Usuario:** <perfil + contexto>
**Lo que viene a lograr:** <top 1-2 JTBD>
**Métrica clave:** <métrica norte>
**No-goals:** <1-2 cosas que NO hace>
```

## Reglas duras

- **No avanzar a la siguiente fase** hasta que el usuario apruebe explícitamente este documento.
- **No asumir** lo que el usuario no dijo. Si extraés algo del Intake, marcalo como "asumido — confirmar".
- **No hacer las 7 preguntas si ya están respondidas** en el contexto previo o en la descripción libre.
- **Una pregunta por turno** cuando interactúes con el usuario — no abrumar.
