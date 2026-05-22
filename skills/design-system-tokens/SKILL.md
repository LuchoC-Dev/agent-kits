---
name: design-system-tokens
description: Define los design tokens del proyecto — colores, tipografía, spacing, radios, sombras, breakpoints, motion. Trigger cuando el usuario quiera "definir tokens", "design system", "paleta de colores", "tipografía", "tokens de diseño" o necesite establecer las bases visuales del proyecto antes de implementar componentes.
consumes:
  - artifact: component-inventory
    required: true
produces:
  - artifact: design-tokens
---

# Design System Tokens

## Rol

Sos un design systems architect. Tu trabajo es **definir el lenguaje visual del proyecto** como un conjunto de tokens reutilizables, no como decisiones puntuales por componente.

## Precondición

El `component-inventory` aprobado. Útil tener referencias visuales del cliente (branding existente, mood board).

## Tokens a definir

### 1. Colores

- **Primary** — color de marca + escala 50-950
- **Neutral/Gray** — escala completa para textos, bordes, backgrounds
- **Semantic** — success, warning, error, info
- **Surface** — background, card, overlay
- **Foreground** — text-primary, text-secondary, text-muted
- **Dark mode** — variantes de cada uno (si aplica)

### 2. Tipografía

- **Font families** — sans, serif, mono (con fallbacks)
- **Type scale** — xs, sm, base, lg, xl, 2xl, 3xl, 4xl (línea + tamaño)
- **Font weights** — normal, medium, semibold, bold
- **Letter spacing** — para títulos grandes

### 3. Spacing

- Escala basada en 4px o 8px (0, 1, 2, 3, 4, 6, 8, 12, 16, 24, 32...)
- Mismo sistema para padding, margin, gap

### 4. Border radius

- none, sm, md, lg, xl, full

### 5. Shadows

- xs, sm, md, lg, xl + dentro (inset)

### 6. Breakpoints

- mobile (default), sm (640), md (768), lg (1024), xl (1280), 2xl (1536)
- Decidir si mobile-first o desktop-first

### 7. Motion

- Duraciones: fast (150ms), base (250ms), slow (400ms)
- Easings: ease-in, ease-out, ease-in-out, spring

## Output

`docs/design/NN-design-tokens.md` (`NN` = prefijo numérico que asigna el workflow según el orden real de ejecución — tomalo de la tabla del workflow) + opcionalmente `tokens.json` o `tailwind.config.ts` con los valores.

```md
# Design Tokens

## Colors

### Primary
- 50: #...
- 500: #... (base)
- 900: #...

### Semantic
- success: #22c55e
- error: #ef4444

## Typography
- Font: Inter, sans-serif
- Scale: ...

## Spacing
- Base: 4px

## Breakpoints
- Strategy: mobile-first
- sm: 640px, md: 768px, lg: 1024px
```

## Regla

Si un color, tamaño o spacing aparece más de una vez, **debe ser un token**. No hardcodear valores en componentes.
