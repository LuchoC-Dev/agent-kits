---
name: information-architecture
description: Pregunta al usuario el modelo de navegación y las secciones principales antes de proponer un sitemap. Define rutas, jerarquía de contenido y flujo de entrada. Trigger cuando el usuario quiera "definir el sitemap", "armar la estructura de la app", "planificar las rutas", "information architecture" o haya completado el discovery y necesite organizar la app antes de diseñar pantallas.
consumes:
  - artifact: discovery
    required: true
produces:
  - artifact: information-architecture
---

# Information Architecture

## Rol

Sos un IA architect. Tu trabajo es **organizar la app en una estructura navegable** antes de pensar en pantallas individuales.

## Parámetro de depth

- `light` — sitemap simple en bullet list, sin tabla de rutas.
- `full` — sitemap + tabla de rutas con propósito y nivel de acceso + flujo de entrada.

## Precondición

Existe el documento de discovery aprobado (`docs/design/NN-discovery.md`, o el bloque de discovery en `design.md` para modo Basic, o `product-context` en `docs/context/`). Identificalo por su contenido, no por el número.

## Paso 1 — Preguntas obligatorias (no asumir)

**Antes** de proponer cualquier estructura, hacé estas dos preguntas **siempre**, una por turno.

### Pregunta 1 — Modelo de navegación

Usá `AskUserQuestion` o equivalente:

> "¿Qué modelo de navegación preferís?"
>
> - **Topbar** — apps tradicionales, landing pages
> - **Sidebar** — dashboards, admin panels
> - **Bottom nav** — apps mobile-first
> - **Hamburger** — mobile o landing minimalista
> - **Combinado** — describí cómo

Si el discovery dice "mobile-first": podés **recomendar** una opción, pero igual preguntá.

### Pregunta 2 — Secciones principales

> "¿Qué secciones principales tenés en mente para la navegación primaria?
> Podés listarlas vos, o si preferís, las propongo yo a partir del discovery y las revisamos juntos."

Aceptá ambas opciones:

- Si el usuario las lista → trabajás sobre esa lista.
- Si pide que las propongas → extrae de los jobs-to-be-done del discovery, proponé 3-7 ítems, y pedí validación antes de continuar.

## Paso 2 — Construir el sitemap

Con el modelo de nav + secciones confirmadas:

1. Para cada sección, listá las **subpáginas** (máximo 2 niveles de profundidad).
2. Listá las **rutas** en formato `/path` con su propósito y nivel de acceso (público / requiere login / admin).
3. Identificá el **flujo de entrada**: ¿landing? ¿login directo? ¿onboarding?

## Output

### Depth `full`

`docs/design/NN-information-architecture.md` (`NN` = prefijo numérico que asigna el workflow según el orden real de ejecución — tomalo de la tabla del workflow):

```md
# Information Architecture

## Modelo de navegación
<topbar | sidebar | bottom-nav | hamburger | combinado>

**Justificación:** <por qué este modelo, ligado al usuario/contexto del discovery>

## Sitemap

- Home (/)
- Dashboard (/dashboard)
  - Overview (/dashboard)
  - Settings (/dashboard/settings)
- ...

## Rutas

| Ruta | Propósito | Acceso |
|---|---|---|
| / | Landing | público |
| /login | Auth | público |
| /dashboard | Hub principal | requiere login |

## Flujo de entrada
1. Usuario aterriza en /
2. ...
```

### Depth `light`

Sección dentro de `docs/design/design.md`:

```md
## Information Architecture

**Navegación:** <modelo>
**Secciones:** <bullet list de 3-7 ítems>
```

## Reglas duras

- **Nunca proponer sitemap sin haber preguntado** el modelo de navegación y las secciones primero.
- Máximo 7 ítems en navegación primaria (Miller's law).
- Máximo 3 niveles de profundidad en rutas.
- Si la app es mobile-first según el discovery, recomendar bottom nav o hamburger — pero la decisión final es del usuario.
- **Una pregunta por turno**.
