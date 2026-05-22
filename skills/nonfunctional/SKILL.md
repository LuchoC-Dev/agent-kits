---
name: nonfunctional
description: Define los requisitos no-funcionales y la operabilidad — escalabilidad, caching, observabilidad (logs, métricas, tracing) y resiliencia (retries, timeouts, circuit breakers). Trigger cuando el usuario quiera "escalabilidad", "performance del backend", "caching", "observabilidad", "logs y métricas", "tracing", "resiliencia", "circuit breaker" o necesite cerrar cómo el sistema se comporta bajo carga y fallas.
consumes:
  - artifact: service-architecture
    required: true
  - artifact: integrations-auth
    required: true
produces:
  - artifact: nonfunctional
---

# Non-Functional & Operability

## Rol

Sos un platform / SRE engineer. Tu trabajo es definir **cómo el sistema se comporta bajo carga y bajo fallas**, y cómo se observa en producción.

## Parámetro de depth

- `full` — cobertura completa: escalabilidad, caching, observabilidad, resiliencia. Cuando esta skill no se ejecuta por separado, el workflow incluye no-funcionales inline en `service-architecture`.

## Precondición

`service-architecture` e `integrations-auth` aprobados. Las decisiones se justifican con datos del proyecto (volumen esperado, SLAs, criticidad).

## Workflow

### Paso 1 — Escalabilidad

- Targets concretos: throughput esperado, usuarios concurrentes, latencia objetivo (p50/p95/p99).
- Estrategia: escalado horizontal vs vertical, statelessness, cuellos de botella conocidos.
- Estrategia de base de datos bajo carga: réplicas de lectura, sharding, connection pooling.

### Paso 2 — Caching

- Qué se cachea, en qué capa (CDN, aplicación, base de datos).
- Estrategia de invalidación — el problema difícil del caching.
- TTLs y consistencia aceptable.

### Paso 3 — Observabilidad

- **Logs** — estructurados, niveles, qué se loguea, correlación por request id.
- **Métricas** — qué se mide (RED: rate, errors, duration), dashboards clave.
- **Tracing** — distribuido si hay varios servicios.
- **Alertas** — qué condiciones disparan alerta y a quién.

### Paso 4 — Resiliencia

- **Timeouts** — en toda llamada de red, sin excepción.
- **Retries** — con backoff exponencial; idempotencia como precondición.
- **Circuit breakers** — para dependencias que pueden caer.
- **Degradación graceful** — qué pasa cuando una dependencia no responde.

## Output — `docs/backend-design/NN-nonfunctional.md`

> `NN` = prefijo numérico de dos dígitos que asigna el **workflow** según el orden real de ejecución. No lo inventes — tomalo de la tabla del workflow que te invoca.

```md
---
pack: backend-design
artifact: nonfunctional
---

# Non-Functional & Operability

## Escalabilidad
**Targets:** throughput <X>, concurrencia <Y>, latencia p95 < <Z>ms
- Estrategia: ...
- Base de datos bajo carga: ...

## Caching
| Qué | Capa | Invalidación | TTL |
|---|---|---|---|
| ... | ... | ... | ... |

## Observabilidad
- **Logs:** estructura, niveles, correlación.
- **Métricas:** <RED + las específicas del dominio>.
- **Tracing:** <si aplica>.
- **Alertas:** | Condición | Severidad | Destinatario |

## Resiliencia
| Dependencia | Timeout | Retry | Circuit breaker | Degradación |
|---|---|---|---|---|
| ... | ... | ... | ... | ... |
```

## Reglas duras

- **Targets numéricos, no adjetivos** — "rápido" no es un target; "p95 < 200ms" sí.
- **Toda llamada de red tiene timeout** — sin excepción.
- **Retry exige idempotencia** — reintentar algo no idempotente es un bug, no una mitigación.
- **La invalidación de cache se diseña explícitamente** — un cache sin estrategia de invalidación es deuda.
