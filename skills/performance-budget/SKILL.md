---
name: performance-budget
description: Define los presupuestos de performance del proyecto — Core Web Vitals, bundle size, tiempos de carga. Trigger cuando el usuario quiera "performance budget", "core web vitals", "LCP", "bundle size", "presupuesto de performance" o necesite establecer límites medibles antes de implementar.
consumes:
  - artifact: component-inventory
    required: true
  - artifact: wireframes
    required: true
produces:
  - artifact: performance-budget
---

# Performance Budget

## Rol

Sos un performance engineer. Tu trabajo es **establecer límites medibles** antes de implementar, no optimizar después de que ya está lento.

## Precondición

Component inventory y wireframes aprobados (ayudan a estimar el tamaño esperado).

## Métricas a presupuestar

### Core Web Vitals

| Métrica | Target | Máximo aceptable |
|---|---|---|
| LCP (Largest Contentful Paint) | < 2.5s | 4s |
| INP (Interaction to Next Paint) | < 200ms | 500ms |
| CLS (Cumulative Layout Shift) | < 0.1 | 0.25 |
| FCP (First Contentful Paint) | < 1.8s | 3s |
| TTFB (Time to First Byte) | < 800ms | 1.8s |

### Bundle size

- **Initial JS** (route raíz): < 150 KB gzipped
- **Initial CSS**: < 50 KB gzipped
- **Route chunks** (lazy loaded): < 100 KB cada uno
- **Imágenes** above-the-fold: < 200 KB total

### Network

- **Requests críticos** (above-the-fold): < 20
- **Total requests** (page load): < 60
- **Fonts**: máximo 2 familias, preload de las críticas

### Runtime

- **Re-renders** por interacción: < 5 (en dev tools profiler)
- **Long tasks** (> 50ms): 0 en interacciones críticas

## Decisiones de arquitectura derivadas

A partir de los targets, decidir:

- **Code splitting** — por ruta sí/no
- **SSR / SSG / CSR** — qué pantallas requieren cada uno
- **Image strategy** — formatos (webp, avif), lazy loading, responsive images
- **Font strategy** — self-hosted vs CDN, subset, font-display
- **Third-party scripts** — qué se carga, cuándo, con qué prioridad

## Output

`docs/design/NN-performance-budget.md` (`NN` = prefijo numérico que asigna el workflow según el orden real de ejecución — tomalo de la tabla del workflow):

```md
# Performance Budget

## Core Web Vitals targets

| Métrica | Target |
|---|---|
| LCP | < 2.5s |
| INP | < 200ms |
| CLS | < 0.1 |

## Bundle budgets

| Asset | Budget |
|---|---|
| Initial JS | 150 KB |
| Initial CSS | 50 KB |
| Route chunks | 100 KB |

## Decisiones derivadas

- SSR para /landing y /producto/[slug] (SEO + LCP)
- CSR para /dashboard/* (interactivo)
- Code splitting por ruta
- Imágenes: WebP con fallback, responsive, lazy
- Fonts: Inter self-hosted, subset latin, font-display: swap

## Monitoreo

- Lighthouse CI en cada PR
- Web Vitals reporting en producción → analytics
- Budget enforced via webpack/vite plugin
```

## Regla

Si un target no se va a medir en CI o en producción, no es un budget — es un deseo.
