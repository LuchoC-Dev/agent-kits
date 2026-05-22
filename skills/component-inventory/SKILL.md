---
name: component-inventory
description: Lista todos los componentes UI necesarios clasificados como atoms, molecules y organisms (Atomic Design). Trigger cuando el usuario quiera "inventariar componentes", "listar componentes", "atomic design", "qué componentes necesito" o esté por arrancar el desarrollo de UI y necesite la lista exhaustiva antes de codear.
consumes:
  - artifact: user-flows
    required: true
produces:
  - artifact: component-inventory
---

# Component Inventory

## Rol

Sos un design systems engineer. Tu trabajo es **identificar todos los componentes UI necesarios** y clasificarlos en niveles de Atomic Design antes de implementar.

## Precondición

El `user-flows` aprobado (los flows revelan qué componentes hacen falta). Identificalo por su artefacto, no por el número de prefijo.

## Workflow

1. Leé los flows y la IA.
2. Para cada pantalla/estado mencionado, identificá los componentes UI presentes.
3. Clasificalos:
   - **Atoms** — Button, Input, Label, Icon, Avatar, Badge, Spinner
   - **Molecules** — FormField (label+input+error), SearchBar, Card, MenuItem
   - **Organisms** — Navbar, Sidebar, DataTable, Form, Modal, EmptyState
   - **Templates** — DashboardLayout, AuthLayout, MarketingLayout
4. Para cada componente, anotá:
   - **Variantes** (primary/secondary, sm/md/lg)
   - **Estados** (default, hover, focus, disabled, loading, error)
   - **Reutilización** (en cuántas pantallas aparece)
5. Marcá los que ya existen en `shadcn/ui`, librería de terceros, o si hay que construirlos desde cero.

## Output

`docs/design/NN-component-inventory.md` (`NN` = prefijo numérico que asigna el workflow según el orden real de ejecución — tomalo de la tabla del workflow):

```md
# Component Inventory

## Atoms

| Componente | Variantes | Estados | Fuente |
|---|---|---|---|
| Button | primary, secondary, ghost, destructive | default, hover, focus, disabled, loading | shadcn |
| Input | text, email, password | default, focus, error, disabled | shadcn |

## Molecules

| Componente | Composición | Usado en |
|---|---|---|
| FormField | Label + Input + ErrorMessage | login, signup, settings |
| SearchBar | Input + Icon + Clear button | dashboard, lista |

## Organisms

| Componente | Composición | Usado en |
|---|---|---|
| Navbar | Logo + NavItems + UserMenu | layout principal |
| DataTable | Header + Rows + Pagination + Filters | listas, reportes |

## Templates

| Layout | Descripción | Pantallas |
|---|---|---|
| DashboardLayout | Sidebar + topbar + main | /dashboard/* |
| AuthLayout | Centered card | /login, /signup |
```

## Regla

Si dos componentes parecen iguales pero con datos distintos, son **el mismo componente**. No dupliques.
