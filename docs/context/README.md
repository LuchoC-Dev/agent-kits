# Contexto de Agent Kits Next

Esta carpeta reúne la fuente de verdad inicial para reconstruir Agent Kits como un
sistema de obtención e instalación de capacidades para agentes.

## Por qué existe

El repositorio heredado contiene un catálogo funcional y muchas decisiones valiosas,
pero fue diseñado principalmente como una skill de bootstrap que copia composiciones a
`.agents/`. La nueva dirección amplía el problema:

- acceso sencillo desde agentes;
- comandos determinísticos;
- recursos públicos y privados;
- IDs únicos entre fuentes;
- instalación, actualización y trazabilidad;
- separación estricta entre creación y obtención;
- compatibilidad con diferentes runtimes.

Estos documentos evitan que la implementación comience interpretando conversaciones
aisladas o revirtiendo decisiones históricas sin advertirlo.

## Mapa documental

| Documento | Contenido |
|---|---|
| `00-overview.md` | **Empezá acá.** Qué es el proyecto, cómo funciona y por qué. |
| `01-product-context.md` | Problema, usuarios, propuesta de valor, alcance y glosario. |
| `02-architecture-direction.md` | Componentes previstos, flujos, invariantes y relación con el legado. |
| `03-decisions.md` | Registro de decisiones tomadas y su justificación. |
| `04-specification.md` | Especificación verificable previa a la implementación. |
| `05-roadmap.md` | Fases, entregables, dependencias y puertas de validación. |
| `06-legacy-baseline.md` | Estado comprobado del repositorio heredado y elementos reutilizables. |
| `07-cli-only-transition-plan.md` | Especificación, migración y tareas para la transición a CLI única. |
| `08-handoff.md` | Estado actual, qué falta, invariantes y trampas. Leelo antes de cambiar algo. |

## Orden de precedencia

Cuando haya contradicciones:

1. Decisiones aprobadas en `03-decisions.md`.
2. Especificación vigente en `04-specification.md`.
3. Dirección de arquitectura en `02-architecture-direction.md`.
4. Contexto de producto en `01-product-context.md`.
5. Documentación heredada (`PROJECT-CONTEXT.md`), que es histórica.

`docs/cli.md` describe el comportamiento real y tiene precedencia sobre cualquier
documento de esta carpeta cuando difieran: acá vive la intención, allá lo que la CLI hace.

## Estado de madurez

**Actualizado el 2026-07-30.** El MVP original y la transición a CLI única están
completos. Agent Kits es hoy una CLI, con el catálogo en repositorios propios y una
publicación privado → público por CI.

Para el estado operativo, lo que falta y las trampas conocidas, leé
[`08-handoff.md`](./08-handoff.md). Esta sección sólo registra la madurez de los
documentos.

| Documento | Estado |
|---|---|
| `00-overview.md` | Active — la puerta de entrada |
| `01-product-context.md` | Histórico — la intención previa a implementar |
| `02-architecture-direction.md` | Implementado, con las preguntas abiertas resueltas en `§13` |
| `03-decisions.md` | Active — D-001 a D-041 |
| `04-specification.md` | Implementado y verificado |
| `05-roadmap.md` | Histórico — fases del MVP, todas cerradas |
| `06-legacy-baseline.md` | Histórico — auditoría del sistema anterior |
| `07-cli-only-transition-plan.md` | Histórico — la transición, completada |
| `08-handoff.md` | Active — estado operativo |

El contrato de la CLI implementada vive en [`../cli.md`](../cli.md), fuera de esta carpeta:
`docs/context/` describe intención y decisiones, `docs/cli.md` describe comportamiento.

## Cómo mantener estos documentos

- Actualizar decisiones antes de modificar el comportamiento que describen.
- No borrar decisiones reemplazadas: marcarlas como `Superseded` y enlazar su reemplazo.
- Separar hechos comprobados, decisiones y propuestas.
- Usar fechas absolutas.
- Mantener ejemplos como ilustrativos mientras el esquema no esté aprobado.

