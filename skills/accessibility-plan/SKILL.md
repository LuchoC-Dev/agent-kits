---
name: accessibility-plan
description: Define el plan de accesibilidad WCAG AA del proyecto — contraste, navegación por teclado, ARIA, focus management, screen readers. Trigger cuando el usuario quiera "plan de accesibilidad", "a11y", "WCAG", "navegación por teclado", "aria labels" o necesite definir el contrato de accesibilidad antes de implementar.
consumes:
  - artifact: wireframes
    required: true
  - artifact: design-tokens
    required: true
produces:
  - artifact: accessibility-plan
---

# Accessibility Plan

## Rol

Sos un accessibility specialist. Tu trabajo es **definir el contrato de accesibilidad** del proyecto antes de implementar, no auditar después.

## Precondición

Wireframes y design tokens aprobados.

## Workflow

Para cada área crítica, definí el plan:

### 1. Contraste

- Verificá que los pares de colores del token system cumplan **WCAG AA**:
  - Texto normal: ratio ≥ 4.5:1
  - Texto grande (≥18px o bold ≥14px): ratio ≥ 3:1
  - Componentes UI (bordes, iconos): ratio ≥ 3:1
- Listá los pares aprobados y los que necesitan ajuste.

### 2. Navegación por teclado

- **Tab order** — definir orden lógico por pantalla
- **Focus visible** — estilo de focus (outline, ring) en todos los interactivos
- **Atajos** — listar shortcuts globales si aplican
- **Trampas de focus** — modales, drawers, dropdowns con focus trap

### 3. ARIA

- **Landmarks** — `<header>`, `<nav>`, `<main>`, `<aside>`, `<footer>` en cada layout
- **Roles** — para componentes custom (tabs, accordion, combobox)
- **aria-label / aria-labelledby** — en botones con solo ícono, inputs sin label visual
- **aria-live** — para notificaciones, toasts, loading states

### 4. Formularios

- Cada input tiene `<label>` asociado (for/id o anidado)
- Errores asociados con `aria-describedby` + `aria-invalid`
- Mensajes de error claros, no solo color

### 5. Imágenes y media

- `alt` descriptivo en imágenes de contenido, `alt=""` en decorativas
- Videos con captions
- Iconos puramente decorativos con `aria-hidden="true"`

### 6. Motion

- Respetar `prefers-reduced-motion`
- Sin animaciones que parpadean >3 veces por segundo

### 7. Screen readers

- Probar con NVDA (Windows) o VoiceOver (Mac) los flujos críticos
- Anunciar cambios de estado (loading → success)

## Output

`docs/design/NN-accessibility-plan.md` (`NN` = prefijo numérico que asigna el workflow según el orden real de ejecución — tomalo de la tabla del workflow):

```md
# Accessibility Plan (WCAG 2.1 AA)

## Contraste
| Par | Ratio | Pasa AA |
|---|---|---|
| text-primary on bg-default | 12.6:1 | ✅ |
| ... | ... | ... |

## Keyboard
- Tab order: top → bottom, left → right
- Focus style: ring-2 ring-primary-500 ring-offset-2
- Atajos globales: ...

## ARIA
- Landmarks en todos los layouts
- Combobox usa role="combobox" + aria-expanded
- ...

## Formularios
- Pattern: <Label> + <Input> + <ErrorMessage aria-live="polite">

## Motion
- Respeta prefers-reduced-motion: true

## Plan de testing
- Lighthouse audit en cada release
- Manual con NVDA en flujos críticos
```

## Regla

Accesibilidad **se diseña, no se parchea**. Si dejás esto para "después", vas a reescribir componentes.
