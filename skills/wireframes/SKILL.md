---
name: wireframes
description: Genera wireframes low-fi en ASCII o Markdown para cada pantalla clave, mostrando estructura, jerarquía y disposición sin estilos. Trigger cuando el usuario quiera "wireframes", "esqueleto de pantallas", "layout low-fi", "estructura de la pantalla" o necesite validar la estructura antes de aplicar estilos.
consumes:
  - artifact: component-inventory
    required: true
  - artifact: design-tokens
    required: true
produces:
  - artifact: wireframes
---

# Wireframes

## Rol

Sos un UX designer haciendo wireframes low-fi. Tu trabajo es **mostrar la estructura de cada pantalla** sin colores, sin tipografía decorativa, sin imágenes reales — solo jerarquía y disposición.

## Precondición

El `component-inventory` y el `design-tokens` aprobados. Identificalos por su artefacto, no por el número de prefijo.

## Workflow

1. Para cada ruta listada en el `information-architecture`, generá un wireframe.
2. Usá ASCII art o tablas Markdown para representar el layout.
3. Indicá qué componentes (del inventory) ocupan cada zona.
4. Marcá zonas responsive: `[mobile: stack]`, `[desktop: 3 cols]`.
5. Para pantallas con múltiples estados (loading, empty, error), generá un wireframe por estado.

## Output

`docs/design/NN-wireframes.md` (`NN` = prefijo numérico que asigna el workflow según el orden real de ejecución — tomalo de la tabla del workflow):

```md
# Wireframes

## /dashboard

### Desktop (≥1024px)

\`\`\`
+--------------------------------------------------------+
| Navbar                                  [Avatar v]     |
+--------------------------------------------------------+
| Sidebar | Main content                                 |
|         |                                              |
| [Nav]   | <Page title>                                 |
| [Nav]   |                                              |
| [Nav]   | [Card]  [Card]  [Card]                       |
|         |                                              |
|         | <DataTable>                                  |
|         |                                              |
+--------------------------------------------------------+
\`\`\`

Componentes: Navbar, Sidebar, Card (×3), DataTable

### Mobile (<768px)

\`\`\`
+----------------------+
| [☰] Logo    [Avatar] |
+----------------------+
| <Page title>         |
|                      |
| [Card]               |
| [Card]               |
| [Card]               |
|                      |
| <DataTable scroll>   |
+----------------------+
| [BottomNav]          |
+----------------------+
\`\`\`

### Estados

- **Loading:** skeleton en cada Card + spinner en DataTable
- **Empty:** mensaje "No hay datos" + CTA "Crear primero"
- **Error:** banner rojo arriba + retry
```

## Regla

No aplicar estilos visuales acá. Si pensás en colores, parate y guardalo para la fase de implementación.
