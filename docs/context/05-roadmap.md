# Roadmap de definición e implementación — Agent Kits Next

**Estado:** Fases 0–7 completadas
**Fecha:** 2026-07-29

> **Roadmap siguiente:** la transición a CLI única, lockfile v2 y catálogo mínimo fue
> aprobada el 2026-07-30. Sus fases y gates están en
> `07-cli-only-transition-plan.md`. Este documento conserva el historial del MVP original.
>
> Esa transición se completó el 2026-07-30. La evolución continúa en D-035 a D-039:
> identidad por UUID, catálogo en su propio repositorio y publicación por CI.

Este roadmap describe puertas de validación.

## Estado por fase

| Fase | Estado | Entregable |
|---|---|---|
| 0 — Baseline y seguridad del fork | ✅ | `docs/context/`, upstream sin push, rama de planificación |
| 1 — Auditoría del legado | ✅ | `06-legacy-baseline.md §8`, auditoría ejecutable |
| 2 — Contratos canónicos | ✅ | `internal/model`, `internal/errs`, D-017 a D-019 |
| 3 — Núcleo de catálogo y resolución | ✅ | `internal/catalog`, `internal/resolve` |
| 4 — Planificador | ✅ | `internal/plan`, plan determinístico y reproducible |
| 5 — Instalación segura | ✅ | `internal/install`, lockfile, rollback, remove, doctor |
| 6 — Sources remotas y privacidad | ✅ | verificada el 2026-07-30 contra `LuchoC-Dev/repository-private`: una identidad autorizada sincroniza; una sin permisos recibe `source_unavailable` y no descubre nada |
| 7 — Adaptadores | ✅ | `agents`, `claude-code`, `opencode` (D-021) |
| 8 — Compatibilidad y adopción | ✅ | adopción de workspaces heredados; hoy vía `migrate` (D-031) |

La Fase 8 se adelantó porque la compatibilidad con `kits-init` pasó a ser un requisito, no
una opción. Todas las fases están cerradas.

## Fase 0 — Baseline y seguridad del fork

### Objetivo

Establecer un workspace independiente y entender el legado.

### Entregables

- fork local con historial;
- upstream de solo lectura;
- rama de planificación;
- contexto y especificación iniciales;
- inventario comprobado;
- mapa de comportamiento heredado.

### Salida

El equipo puede explicar qué existe sin depender de la memoria de una conversación.

## Fase 1 — Auditoría del legado

### Objetivo

Determinar qué componentes se reutilizan, migran, envuelven o descartan.

### Trabajo

- auditar `SKILL.md`;
- auditar `workspace-schema.md`;
- auditar `repair-upgrade.md`;
- validar manifests de packs;
- contar y clasificar skills;
- validar agentes globales y por pack;
- identificar dependencias implícitas;
- comparar documentación con filesystem;
- construir fixtures representativos.

### Puerta

No diseñar migraciones hasta disponer de una matriz:

```text
componente | comportamiento actual | decisión | riesgo | prueba
```

## Fase 2 — Contratos canónicos

### Objetivo

Definir la identidad y los contratos sin elegir todavía todos los detalles operativos.

### Entregables

- schema de resource manifest;
- schema de source;
- schema de catálogo;
- schema de lockfile;
- códigos de error;
- política de IDs;
- semántica de dependencias;
- compatibilidad con recursos heredados.

### Puerta

Fixtures válidos e inválidos revisados por el usuario.

## Fase 3 — Núcleo de catálogo y resolución

### Objetivo

Construir la parte de solo lectura antes de tocar proyectos.

### Capacidades

- cargar sources locales;
- validar catálogos;
- agregar IDs;
- detectar duplicados;
- buscar;
- inspeccionar;
- resolver dependencias;
- emitir JSON.

### Puerta

Ningún comando de esta fase escribe en proyectos ni remotos.

## Fase 4 — Planificador

### Objetivo

Convertir una resolución en un plan determinístico.

### Capacidades

- mapping de destino;
- clasificación de archivos;
- checksums;
- detección de divergencia;
- actualización propuesta del lockfile;
- salida humana y JSON.

### Puerta

El mismo estado produce el mismo plan.

## Fase 5 — Instalación segura

### Objetivo

Aplicar planes aprobados sin corrupción ni sobrescritura silenciosa.

### Capacidades

- escritura contenida;
- lockfile;
- idempotencia;
- recuperación ante fallos;
- remove seguro;
- doctor.

### Puerta

Fixtures end-to-end y escenarios de fallo verificados.

## Fase 6 — Sources remotas y privacidad

### Objetivo

Conectar sources Git reales sin romper los límites de acceso.

### Capacidades

- cache;
- sync;
- autenticación externa;
- source pública y privada;
- registro global autorizado;
- reglas de dependencia por visibilidad.

### Puerta

Pruebas que demuestren que una identidad sin permisos no descubre contenido privado.

## Fase 7 — Adaptadores

### Objetivo

Soportar runtimes de manera explícita y verificable.

### Orden candidato

1. `.agents/` genérico;
2. Codex;
3. runtimes adicionales aprobados.

### Puerta

Cada adaptador declara capacidades, limitaciones y tests propios.

## Fase 8 — Compatibilidad y adopción

### Objetivo

Evaluar migración desde workspaces creados por `kits-init`.

### Trabajo

- detectar schema heredado;
- importar lock inicial;
- mapear packs y skills;
- documentar incompatibilidades;
- ofrecer migración con dry-run.

Esta fase puede quedar fuera de la primera release.

## Trabajo que puede ejecutarse en paralelo

Después de aprobar la Fase 1:

- investigación del stack;
- diseño de schemas;
- diseño de seguridad;
- construcción de fixtures;
- documentación de compatibilidad.

La implementación del instalador depende del planificador y del lockfile; no debe
comenzar en paralelo con contratos todavía inestables.

## Riesgos principales

| Riesgo | Mitigación |
|---|---|
| Sobrediseñar antes del MVP | Mantener tipos y runtimes iniciales acotados. |
| Romper compatibilidad heredada | Fixtures y matriz de migración. |
| Filtrar nombres privados | Registro autorizado y mensajes de error neutros. |
| Ejecutar contenido malicioso | Modelo de confianza y seguridad antes de scripts. |
| Sobrescribir cambios del proyecto | Checksums de tres vías y fail closed. |
| Acoplarse a un runtime | Core canónico y adaptadores de borde. |
| Crear dos productos a la vez | Mantener autoría fuera del alcance. |

## Próxima acción recomendada

Las cinco acciones que este roadmap enumeraba se resolvieron: la colisión de destino
(D-034), las versiones reales (D-034), la topología y la verificación de la source privada
(D-037, T19) y la migración a manifests nativos (D-034). La autoría sigue fuera de alcance
(D-003, D-013).

Lo que sigue vive en las decisiones D-037 a D-039: el workflow de publicación que lleva
recursos del catálogo privado a su espejo público.
