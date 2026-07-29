# Especificación — Agent Kits Next

**Estado:** Draft — requiere revisión humana
**Fecha:** 2026-07-29

## 1. Objetivo

Construir una CLI y núcleo de resolución que permitan a una persona o agente descubrir,
planificar e instalar recursos desde sources Git públicas y privadas, de manera
reproducible y segura.

### Historias principales

1. Como usuario, quiero instalar un recurso por ID sin conocer su ubicación física.
2. Como agente, quiero obtener resultados JSON y códigos de error estables.
3. Como usuario, quiero ver un plan antes de modificar mi proyecto.
4. Como usuario, quiero que una segunda instalación sea idempotente.
5. Como propietario, quiero mantener recursos privados invisibles para terceros.
6. Como mantenedor, quiero que un ID duplicado bloquee la resolución.
7. Como usuario, quiero conocer qué versión y archivos fueron instalados.

## 2. Stack

**Decidido: Go, solo librería estándar** (D-016).

| Criterio | Resolución |
|---|---|
| Distribución | `go build` → binario estático en Windows, macOS y Linux |
| Filesystem y Git | `os`, `io/fs`, `path/filepath` y `os/exec` sobre el `git` del sistema |
| Salida JSON | `encoding/json` |
| Checksums | `crypto/sha256` |
| Testing | `go test` sin configuración |
| Dependencias | ninguna; `go.mod` no declara `require` |

Git se invoca como proceso externo con una whitelist de subcomandos de solo lectura, en
lugar de embeber una implementación de Git. Esto mantiene la garantía de "no remote
writes" verificable por inspección.

## 3. Comandos

### Comandos actuales de diagnóstico

```powershell
git status --short --branch
git remote -v
rg --files --hidden -g '!.git/**'
git diff --check
```

### Contrato de la CLI

```text
agent-kits source list [--json]
agent-kits source add <name> <url> [--access public|private] [--trust trusted|review] [--ref <ref>]
agent-kits source remove <name>
agent-kits source sync [<name>] [--json]
agent-kits search [<query>] [--type <type>] [--source <name>] [--json]
agent-kits info <id> [--json]
agent-kits plan <id>... --project <path> [--runtime auto] [--json]
agent-kits install <id>... --project <path> [--runtime auto] [--yes] [--force] [--json]
agent-kits update [<id>...] --project <path> [--yes] [--force] [--json]
agent-kits remove <id>... --project <path> [--yes] [--json]
agent-kits list --project <path> [--json]
agent-kits doctor --project <path> [--json]
agent-kits import --project <path> [--yes] [--json]
agent-kits version [--json]
```

`import` adopta un workspace creado por `kits-init` (D-022). Todo comando mutante acepta
`--json` y requiere `--yes` cuando el plan escribe archivos.

### Códigos de salida

| Código | Significado |
|---|---:|
| 0 | operación completada |
| 1 | fallo genérico |
| 2 | error de uso |
| 3 | integridad del registro (duplicados, ambigüedad, manifest inválido) |
| 4 | conflicto que requiere decisión (divergencia local, confirmación) |
| 5 | source inaccesible |
| 6 | violación de seguridad |

## 4. Estructura de proyecto prevista

La estructura definitiva depende del stack. La separación lógica mínima será:

```text
src/
├── cli/          command parsing and presentation
├── registry/     sources, catalogs, IDs and dependency resolution
├── install/      plans, filesystem changes and lockfile
├── adapters/     runtime-specific destinations
└── security/     integrity and trust checks

tests/
├── unit/
├── integration/
└── fixtures/

docs/
└── context/
```

El catálogo heredado permanecerá sin mover hasta que una tarea de migración aprobada
indique la estructura de destino.

## 5. Estilo y naming

Los identificadores técnicos usan inglés y kebab-case para IDs:

```yaml
id: frontend-design
type: skill
name: Frontend Design
```

Los agentes nombran roles:

```yaml
id: frontend-designer
type: agent
name: Frontend Designer
skills:
  - frontend-design
```

Reglas:

- funciones y tipos con nombres orientados al dominio;
- errores estructurados con código estable;
- no codificar una source en el ID;
- no usar nombres de archivos como identidad de artefactos;
- evitar abreviaturas no documentadas.

## 6. Estrategia de testing

El framework se decidirá con el stack, pero los niveles son obligatorios.

### Unit

- parseo y validación de manifests;
- agregación de catálogos;
- detección de IDs duplicados;
- resolución de dependencias;
- comparación de checksums;
- planificación de cambios.

### Integration

- source pública local;
- source privada simulada;
- instalación nueva;
- instalación repetida;
- actualización sin cambios locales;
- actualización con divergencia;
- recurso no encontrado;
- dependencia pública desde privado;
- dependencia privada desde público;
- lockfile reproducible.

### Security

- path traversal;
- symlinks fuera del proyecto;
- archivos inesperadamente grandes;
- scripts no autorizados;
- checksum inválido;
- manifest malicioso;
- credenciales incluidas por error.

### End-to-end

- instalar un kit completo en un fixture vacío;
- ejecutar `doctor`;
- repetir instalación;
- comprobar que el segundo plan no escribe;
- retirar el kit sin borrar archivos ajenos.

## 7. Límites

### Siempre

- validar antes de escribir;
- generar plan;
- verificar IDs globales;
- registrar origen y checksum;
- preservar archivos ajenos;
- emitir errores accionables;
- mantener sources remotas en solo lectura.

### Preguntar primero

- añadir dependencias;
- cambiar formatos heredados;
- introducir migraciones;
- habilitar scripts;
- añadir runtimes;
- cambiar compatibilidad;
- crear remotos o publicar.

### Nunca

- publicar desde Agent Kits;
- guardar secretos;
- sobrescribir silenciosamente;
- resolver duplicados por orden;
- exponer contenido privado;
- escribir fuera del proyecto destino;
- hacer push a upstream.

## 8. Requisitos funcionales

### RF-01 — Sources

El sistema permite configurar, listar y sincronizar sources públicas y privadas.

### RF-02 — Catálogo agregado

El sistema genera una vista consultable de recursos disponibles.

### RF-03 — Unicidad

La vista se invalida si un ID activo aparece más de una vez.

### RF-04 — Búsqueda

La búsqueda devuelve ID, tipo, descripción, source, versión y compatibilidad.

### RF-05 — Resolución

La resolución incluye dependencias transitivas y detecta ciclos.

### RF-06 — Plan

El sistema produce un plan completo sin escribir archivos.

### RF-07 — Instalación

La instalación materializa únicamente los archivos aprobados dentro del destino.

### RF-08 — Idempotencia

Repetir una instalación sin cambios produce un plan vacío.

### RF-09 — Lockfile

Cada recurso instalado queda trazado por origen, versión, commit y checksum.

### RF-10 — Conflictos

Los archivos locales divergentes detienen o requieren resolución explícita.

### RF-11 — Salida estructurada

Los comandos ofrecen JSON y códigos de salida documentados.

### RF-12 — Diagnóstico

`doctor` detecta fuentes inaccesibles, duplicados, archivos faltantes, checksums
divergentes e incompatibilidades.

## 9. Requisitos no funcionales

- Compatibilidad inicial con Windows PowerShell.
- Operaciones determinísticas y auditables.
- Sin telemetría por defecto.
- Mensajes comprensibles para personas y agentes.
- Ninguna credencial escrita en logs.
- Tiempo de respuesta razonable con cache local.
- Documentación versionada junto al código.

## 10. Criterios de éxito del MVP

Verificados el 2026-07-29 contra el catálogo heredado real (75 recursos):

- [x] Una source pública local puede sincronizarse.
- [x] Un recurso se descubre por ID.
- [x] `plan` no modifica el proyecto.
- [x] `install` crea los archivos previstos y un lockfile.
- [x] Repetir `install` no cambia archivos.
- [x] Un ID duplicado bloquea la operación.
- [x] Un checksum divergente se informa sin sobrescribir.
- [x] La CLI devuelve JSON válido.
- [x] No existe ninguna operación remota de escritura.
- [x] Los tests definidos para el alcance aprobado están verdes.

Añadidos durante la implementación:

- [x] Un workspace creado por `kits-init` puede adoptarse con `import` (D-022).
- [x] Dos recursos que escriben el mismo archivo bloquean la operación (D-028).
- [x] Una operación fallida restaura el proyecto a su estado anterior.

## 11. Fuera del MVP

- UI gráfica;
- marketplace público;
- publicación;
- creación de recursos;
- colaboración en tiempo real;
- ejecución persistente de workflows;
- soporte completo de todos los runtimes;
- migración automática de todos los workspaces heredados.

## 12. Preguntas abiertas

Resueltas el 2026-07-29:

| # | Pregunta | Resolución |
|---|---|---|
| 1 | Stack | Go, solo stdlib (D-016) |
| 2 | Primer runtime | `agents` genérico, con `claude-code` y `opencode` (D-021) |
| 3 | Tipos del MVP | los cuatro: skill, agent, workflow, kit (D-020) |
| 5 | SemVer y dependencias | SemVer 2.0.0 con exacto, `^`, `~`, `*` (D-024) |
| 6 | Archivos modificados | tres vías, fail closed, `--force` explícito (D-023) |
| 7 | Formato del lockfile | JSON en `.agents/agent-kits.lock.json` (D-018) |
| 8 | Confianza en sources | `trust` por source, sin ejecución de contenido (D-025) |
| 9 | Compatibilidad `kits-init` | obligatoria, con `import` (D-022) |

Todavía abiertas:

| # | Pregunta | Estado |
|---|---|---|
| 4 | Registro global de reserva de IDs | el MVP valida unicidad sobre la vista agregada; la autoridad de reserva se define al incorporar sources remotas de terceros |
| 10 | Remoto `origin` colaborativo | pendiente de acuerdo |

## 13. Puerta de aprobación

**Superada el 2026-07-29.** El usuario aprobó stack, alcance completo del MVP, manifests
JSON y compatibilidad obligatoria con `kits-init`. La implementación está autorizada.

La auditoría del legado se completó como parte del diseño del adaptador de catálogo
(D-026); sus hallazgos están en `06-legacy-baseline.md §7` y motivaron D-019.
