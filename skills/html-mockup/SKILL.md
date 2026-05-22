---
name: html-mockup
description: Genera mockups HTML+Tailwind por pantalla con fidelidad ajustable (mid-fi-plain, mid-fi-tokens, hi-fi). Alternativa visual a wireframes ASCII cuando el usuario necesita "sentir" el producto antes de implementar. Trigger cuando el usuario pida "mockup HTML", "prototipo visual", "wireframe en HTML" o "ver cómo se siente".
consumes:
  - artifact: component-inventory
    required: true
  - artifact: design-tokens
    required: false
  - artifact: accessibility-plan
    required: false
  - artifact: wireframes
    required: false
produces:
  - artifact: html-mockups
---

# HTML Mockup

## Rol

Sos un visual designer haciendo mockups HTML. Tu trabajo es **traducir la estructura definida en fases anteriores a HTML+Tailwind navegable** para que el usuario sienta el producto antes de implementarlo de verdad.

No sos un developer todavía. El HTML que producís es **descartable** — sirve para validar, no para shippear.

## Cuándo usar esta skill en vez de `wireframes`

| Usá `wireframes` (ASCII) cuando... | Usá `html-mockup` cuando... |
|---|---|
| Querés validar estructura/jerarquía rápido | Querés validar **feel** y fidelidad visual |
| El proyecto es chico (modo Basic) | El proyecto es mediano/grande (Medium/Advanced) |
| Tokens son escasos | Podés gastar tokens en algunas pantallas críticas |
| Vas a iterar mucho antes de mockear | Las estructuras ya están definidas |

## Parámetro de depth global

Esta skill, además del depth de la skill (`light` / `full`), tiene un parámetro **`fidelity` por pantalla**:

| Fidelity | Tokens aprox/pantalla | Uso |
|---|---|---|
| `mid-fi-plain` | 2.000-4.000 | Tailwind con grises/blancos. Valida layout y spacing. |
| `mid-fi-tokens` | 2.500-5.000 | Usa design tokens reales de fase 5. **Default recomendado.** |
| `hi-fi` | 6.000-15.000+ | Estados (`hover:`, `focus:`), ARIA, responsive completo, interactividad básica. **Reservar para pantallas críticas (2-3 máximo).** |

## Precondición

- El `component-inventory` aprobado.
- Para `mid-fi-tokens` y `hi-fi`: los `design-tokens` aprobados (los tokens existen).
- Para `hi-fi`: ideal tener también el `accessibility-plan` para incluir ARIA correctamente.

> Identificá esos documentos en `docs/design/` por su artefacto (frontmatter `artifact:`), no por el prefijo numérico — el número depende del workflow.

## Paso 1 — Selección de fidelidad por pantalla

**Antes de generar nada**, mostrale al usuario la lista de pantallas (del `information-architecture`) y preguntá qué fidelidad usar en cada una:

> Te muestro las pantallas. Por cada una, elegí fidelidad:
>
> | Pantalla | Recomendación | Mid-fi-plain | Mid-fi-tokens | Hi-fi |
> |---|---|---|---|---|
> | Home (/) | mid-fi-tokens | | ✓ | |
> | PDP (/p/[slug]) | hi-fi | | | ✓ |
> | Checkout | hi-fi | | | ✓ |
> | Carrito | mid-fi-tokens | | ✓ | |
>
> **Estimado total:** ~XX.000 tokens. ¿Aprobás?

Recomendá `hi-fi` solo para pantallas críticas del producto (definidas por los JTBD del discovery). Todo el resto: `mid-fi-tokens`.

Si el usuario aprueba el plan, continuá. Si no, ajustá.

## Paso 2 — Generar mockups

Para cada pantalla:

1. Tomá el wireframe ASCII (documento `wireframes`) si existe, o el component inventory.
2. Generá un archivo HTML standalone en `docs/design/mockups/<screen-id>.html`.
3. Usá Tailwind via CDN (`<script src="https://cdn.tailwindcss.com"></script>`) para que funcione sin build step.
4. Cargá los tokens del `design-tokens` como CSS variables al inicio del archivo (si fidelity ≥ `mid-fi-tokens`).
5. Marcá zonas con comentarios HTML: `<!-- Hero section -->`, `<!-- Product grid -->`.

### Estructura de cada mockup

```html
<!DOCTYPE html>
<html lang="es">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Mockup — <pantalla></title>
  <script src="https://cdn.tailwindcss.com"></script>
  <style>
    /* Solo si fidelity ≥ mid-fi-tokens — tokens del documento design-tokens */
    :root {
      --color-primary-500: #...;
      --color-foreground: #...;
      /* ... */
    }
  </style>
</head>
<body>
  <!-- Mockup content -->
</body>
</html>
```

### Reglas por fidelity

**`mid-fi-plain`:**
- Solo Tailwind clases utilitarias con grises (`bg-gray-100`, `text-gray-900`, `border-gray-200`)
- Sin hover/focus states
- Sin imágenes reales — usar placeholders (`bg-gray-300` con altura fija)
- Sin íconos de librería — usar emojis o cajas grises

**`mid-fi-tokens`:**
- Reemplazar grises por tokens reales (`bg-primary-500`, `text-foreground`)
- Tipografía con tokens (`font-sans`, `text-base`, `font-semibold`)
- Spacing del sistema de `design-tokens`
- Aún sin hover/focus states ni interactividad

**`hi-fi`:**
- Todos los estados: `hover:`, `focus:`, `active:`, `disabled:`
- Atributos ARIA correctos según el `accessibility-plan`
- Responsive completo (`sm:`, `md:`, `lg:`)
- Imágenes con `alt` real (placeholder describiendo qué iría ahí)
- Iconos (Lucide vía CDN o Heroicons inline SVG)
- Links cruzados entre pantallas (`href="./other-screen.html"`)
- Forms con validación HTML básica

## Output

```
docs/design/
├── NN-wireframes.md            (si también se hizo wireframes ASCII)
└── mockups/
    ├── index.html              (página listing con links a cada mockup)
    ├── home.html
    ├── pdp.html
    ├── checkout.html
    └── ...
```

El `index.html` lista todos los mockups con su fidelity y un link para abrirlos.

## Regla de cierre

Al terminar, mostrale al usuario:

- La lista de archivos generados
- Cómo abrirlos (`open docs/design/mockups/index.html`)
- Pedile feedback **por pantalla** antes de aprobar la fase completa
