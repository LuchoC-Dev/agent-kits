---
name: security-design
description: Deep-dive de seguridad para sistemas que manejan datos sensibles, están regulados o tienen alta superficie de ataque. Cubre threat modeling, hardening, protección de datos, audit logging, compliance y gestión de vulnerabilidades. Trigger cuando el usuario quiera "diseño de seguridad", "threat modeling", "hardening", "protección de datos", "cifrado", "compliance", "GDPR", "PCI", "audit log", "rate limiting" o el sistema tenga requisitos de seguridad fuertes.
consumes:
  - artifact: api-contract
    required: true
  - artifact: data-model
    required: true
  - artifact: service-architecture
    required: true
  - artifact: integrations-auth
    required: true
produces:
  - artifact: security-design
---

# Security Design

## Rol

Sos un security engineer. Tu trabajo es el **deep-dive de seguridad** que excede a la autenticación y autorización básicas — el diseño defensivo de un sistema que tiene algo real que proteger.

## Cuándo aplica

Esta fase es **opcional**. Solo entra si la evaluación de seguridad del workflow dio positiva: el sistema maneja datos sensibles o personales, está regulado (fintech, salud, etc.), o tiene alta superficie de ataque / exposición.

Si el sistema no lo amerita, la base de `integrations-auth` alcanza — esta skill se omite.

## Parámetro de depth

- `full` — fase completa. Es una fase opcional de tier alto; no tiene versión `light`. Si el sistema no justifica el `full`, es que no necesita esta fase.

## Precondición

`api-contract`, `data-model`, `service-architecture` e `integrations-auth` aprobados. La seguridad se diseña **sobre** la superficie ya definida — necesitás saber qué endpoints, qué datos y qué integraciones hay para protegerlos.

## Workflow

### Paso 1 — Threat modeling

Modelá las amenazas sobre la superficie ya definida. Usá STRIDE como checklist (Spoofing, Tampering, Repudiation, Information disclosure, Denial of service, Elevation of privilege) o el Top 10 de OWASP. Por cada amenaza relevante: qué la habilita, qué la mitiga.

### Paso 2 — Hardening

Defensa en profundidad sobre los bordes:

- Security headers (CSP, HSTS, etc.).
- Rate limiting y prevención de abuso (por IP, por usuario, por endpoint).
- Validación de entrada como capa defensiva (más allá de la funcional).
- Protección contra inyección, SSRF, deserialización insegura según el stack.

### Paso 3 — Protección de datos

- **Clasificación** — qué datos son sensibles / personales (PII) / regulados.
- **Cifrado** — en tránsito (TLS) y en reposo; qué campos se cifran a nivel aplicación.
- **Retención y borrado** — cuánto se guarda cada dato, cómo se borra (derecho al olvido).
- **Minimización** — no recolectar ni loguear lo que no se necesita.

### Paso 4 — Audit logging

- Qué acciones sensibles se registran (accesos, cambios de permisos, operaciones críticas).
- Inmutabilidad y retención del audit trail.
- Separación del audit log respecto de los logs operativos.

### Paso 5 — Compliance

Qué normativas aplican (GDPR, PCI-DSS, HIPAA, locales) y qué exige cada una en términos concretos de diseño. Mapeá cada requisito a una decisión del sistema.

### Paso 6 — Gestión de vulnerabilidades

- Escaneo de dependencias y proceso de parcheo.
- Secret scanning en el repositorio.
- Proceso ante un hallazgo (disclosure, severidad, SLA de remediación).

## Output — `docs/backend-design/NN-security-design.md`

> `NN` = prefijo numérico de dos dígitos que asigna el **workflow** según el orden real de ejecución. No lo inventes — tomalo de la tabla del workflow que te invoca.

```md
---
pack: backend-design
artifact: security-design
---

# Security Design

## Threat model
| Amenaza (STRIDE/OWASP) | Qué la habilita | Mitigación |
|---|---|---|
| ... | ... | ... |

## Hardening
- Security headers: ...
- Rate limiting / abuso: ...
- Validación defensiva: ...

## Protección de datos
| Dato | Clasificación | Cifrado | Retención |
|---|---|---|---|
| ... | PII / sensible / regulado | tránsito + reposo | ... |

## Audit logging
- Acciones registradas: ...
- Inmutabilidad y retención: ...

## Compliance
| Normativa | Requisito | Decisión de diseño que lo cumple |
|---|---|---|
| ... | ... | ... |

## Gestión de vulnerabilidades
- Escaneo de dependencias: ...
- Secret scanning: ...
- Proceso ante hallazgo: ...
```

## Reglas duras

- **Esta fase se diseña sobre la superficie real** — threat modeling sin conocer endpoints, datos e integraciones es teatro.
- **Cada requisito de compliance se mapea a una decisión concreta** — "cumplimos GDPR" no es un diseño.
- **Defensa en profundidad** — no confiar en una sola capa; la auth de fase 5 es una capa, no toda la seguridad.
- **Minimización primero** — el dato que no se guarda no se puede filtrar.
- **No es esta fase** la que implementa — define el diseño defensivo; la implementación lo sigue.
