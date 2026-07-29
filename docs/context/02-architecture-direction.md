# Dirección de arquitectura — Agent Kits Next

**Estado:** Draft
**Fecha:** 2026-07-29

## 1. Principios

1. **Git-first:** los recursos y sus versiones viven en repositorios.
2. **Filesystem-first:** la instalación materializa archivos auditables.
3. **Agent-friendly:** comandos no interactivos, salida estructurada y errores claros.
4. **Plan before write:** toda operación potencialmente mutante puede previsualizarse.
5. **Fail closed:** ambigüedad, duplicados o integridad inválida detienen la operación.
6. **No remote writes:** la CLI consumidora trata todas las fuentes como solo lectura.
7. **Portable core:** identidad y contratos no dependen de un runtime concreto.
8. **Adapters at the edge:** las diferencias de Codex, Claude Code u otros entornos se
   resuelven en adaptadores.
9. **Evolution by compatibility:** el legado se audita y migra; no se elimina por
   preferencia estética.

## 2. Capas previstas

```text
User or agent intent
        │
        ▼
Agent Kits CLI
        │
        ├── command parser
        ├── planner
        ├── structured output
        │
        ▼
Registry and resolution core
        ├── source manager
        ├── catalog loader
        ├── global ID resolver
        ├── dependency resolver
        └── integrity validator
        │
        ▼
Installation core
        ├── file planner
        ├── conflict detector
        ├── lockfile manager
        └── rollback/recovery strategy
        │
        ▼
Runtime adapters
        ├── generic .agents
        ├── Codex
        └── other approved runtimes
```

Esta descomposición es conceptual. No decide todavía lenguaje, módulos físicos ni
dependencias.

## 3. Fuentes

Una source debe declarar como mínimo:

```yaml
name: public
url: https://github.com/example/agent-kits-public
access: public
```

Una source privada usaría una URL accesible mediante credenciales externas:

```yaml
name: personal
url: git@github.com:example/agent-kits-private.git
access: private
```

El campo `access` es informativo. Git y el proveedor remoto aplican autorización.

### Reglas

- Las sources se sincronizan a un cache local controlado.
- La CLI no modifica sus working trees remotos.
- Una source privada puede depender de una pública.
- Una source pública no puede depender de una privada.
- Los IDs se agregan en una vista global antes de resolver dependencias.
- Cualquier duplicado invalida la vista completa.

## 4. Registro global de IDs

La unicidad entre repositorios requiere una autoridad de reserva. Su ubicación exacta
sigue abierta.

Propiedades requeridas:

- conoce IDs públicos y privados;
- no expone nombres privados involuntariamente;
- permite reservar antes de incorporar contenido;
- soporta transición de visibilidad;
- puede validarse en un entorno autorizado;
- no permite que la CLI consumidora escriba reservas.

La primera implementación no debe simular unicidad global solo con orden de sources.

## 5. Resolución

Entrada:

```text
frontend-design
```

Salida lógica:

```yaml
id: frontend-design
type: skill
source: public
version: 1.2.0
checksum: sha256:...
dependencies: []
```

Si hay cero coincidencias, se devuelve `not_found`. Si hay más de una, se devuelve
`registry_integrity_error`. Nunca se elige la primera.

## 6. Plan de instalación

Antes de escribir, el sistema calcula:

- recurso solicitado;
- dependencias transitivas;
- archivos nuevos;
- archivos sin cambios;
- archivos modificados localmente;
- reemplazos necesarios;
- destino por runtime;
- actualización propuesta del lockfile;
- riesgos y bloqueos.

El plan debe poder emitirse como texto humano y JSON estable.

## 7. Política de escritura

La política exacta para archivos modificados sigue abierta, pero se fijan invariantes:

- nunca sobrescribir silenciosamente;
- distinguir archivos administrados de archivos propios del proyecto;
- comparar checksum instalado, checksum actual y checksum nuevo;
- abortar o pedir decisión cuando el archivo local diverge;
- mantener el proyecto consistente si falla una operación.

No se implementará actualización hasta definir recuperación o atomicidad suficiente.

## 8. Lockfile

El lockfile debe registrar, por recurso:

```yaml
id: frontend-design
type: skill
source: public
version: 1.2.0
commit: 0123456789abcdef
checksum: sha256:...
runtime: codex
files:
  - path: .agents/skills/frontend-design/SKILL.md
    checksum: sha256:...
```

El formato es ilustrativo. Debe decidirse si el lockfile es YAML, JSON u otro formato
antes de implementarlo.

## 9. Adaptadores de runtime

El catálogo describe recursos canónicos. Un adaptador determina:

- carpeta destino;
- archivos de bootstrap;
- sintaxis de invocación;
- capacidades no soportadas;
- transformaciones permitidas;
- verificación posterior.

Un adaptador no cambia el significado del recurso. Si necesita una transformación
semántica, debe declararse como incompatibilidad o paso explícito.

## 10. Relación con el legado

Elementos potencialmente reutilizables:

- pool plano de `skills/`;
- packs como composiciones declarativas;
- separación skill/agent/workflow;
- `produces` y `consumes`;
- catálogo plano;
- estructura `.agents/`;
- filosofía “el agente es el runtime” para kits basados en archivos.

Elementos que no se trasladan automáticamente:

- `SKILL.md` raíz como única interfaz de instalación;
- detección hardcodeada de runtimes;
- copia directa sin lockfile;
- esquema actual de `workspace.json`;
- clasificación histórica de agentes;
- conteos o documentación que no coincidan con el filesystem.

## 11. Seguridad

Antes de instalar contenido público se deberá definir:

- qué archivos pueden incluir scripts ejecutables;
- cómo se declaran permisos y compatibilidad;
- validación de rutas y path traversal;
- límites de tamaño;
- protección frente a symlinks;
- detección de secretos;
- revisión de hooks o instrucciones peligrosas;
- política de confianza por source;
- checksums y trazabilidad.

Un repositorio público no implica confianza automática.

## 12. Topología Git del fork

```text
upstream  fetch → https://github.com/LuchoC-Dev/agent-kits.git
upstream  push  → DISABLED
origin           → pendiente
branch           → planning/agent-kits-next
```

No se añadirá `origin` hasta acordar dónde vivirá el fork colaborativo.

## 13. Preguntas abiertas

Resueltas el 2026-07-29:

| # | Pregunta | Resolución |
|---|---|---|
| 1 | Lenguaje y distribución | Go, solo stdlib, binario estático (D-016) |
| 2 | Formato de manifests e índices | JSON, `agent-kit.json` con `schema_version`; los índices se descubren recorriendo el árbol, sin archivo de índice que pueda desincronizarse (D-017) |
| 4 | Versiones y rangos | SemVer 2.0.0 con exacto, `^`, `~`, `*` (D-024) |
| 5 | Archivos modificados localmente | tres vías, fail closed, `--force` explícito (D-023) |
| 6 | Atomicidad y rollback | journal de respaldos fuera del proyecto; un fallo restaura el estado anterior |
| 7 | Primer runtime | `agents` genérico, más `claude-code` y `opencode` (D-021) |
| 8 | Tipos del MVP | los cuatro (D-020) |
| 9 | Relación con `fast-weekend-core` | ninguna: es solo la carpeta contenedora, sin acoplamiento técnico |
| 10 | Confianza en fuentes públicas | `trust` por source, sin ejecución de contenido, validación previa a escribir (D-025) |

Todavía abierta:

| # | Pregunta | Estado |
|---|---|---|
| 3 | Registro global de reserva de IDs | el MVP valida unicidad sobre la vista agregada, que alcanza para sources conocidas. Una autoridad de reserva hace falta cuando el catálogo acepte sources de terceros |
