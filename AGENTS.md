# Agent Kits Next — guía para agentes

Este repositorio **es** `LuchoC-Dev/agent-kits`: la CLI de Agent Kits (D-040). El catálogo
vive aparte, en `repository-private` y su espejo `repository` (D-037).

## Estado actual

- Fase: **transición a CLI única completada**, incluida la verificación de la source
  privada real (`docs/context/07-cli-only-transition-plan.md`).
- Rama de trabajo: `main`.
- `origin`: `LuchoC-Dev/agent-kits`, con push habilitado desde D-040.
- Stack: **Go, solo librería estándar** (D-016). `go.mod` no declara `require`.
- **La CLI es la única superficie.** `kits-init`, `workspace.json`, el adaptador del
  catálogo heredado y la migración que los retiraba ya no existen (D-029, D-030, D-034,
  D-041). El estado de un proyecto vive solo en `.agents/agent-kits.lock.json`.
- **El catálogo no vive acá** (D-037). Está en `LuchoC-Dev/repository-private`, y su
  espejo publicado en `LuchoC-Dev/repository`. Este repositorio es sólo la herramienta.

La CLI vive en `cmd/agent-kits` e `internal/`. Su referencia es `docs/cli.md`.

```powershell
go build ./...
go test ./...
```

## Lectura obligatoria

Dos documentos, en este orden:

1. **`docs/context/00-overview.md`** — qué es el proyecto, cómo funciona y por qué está
   hecho así. Es la puerta de entrada y se lee entero.
2. **`docs/context/08-handoff.md`** — el estado operativo: qué falta, qué invariantes no se
   pueden romper y qué trampas cuestan tiempo.

Con eso alcanza para trabajar. El resto se consulta cuando hace falta:

- `docs/cli.md` — el comportamiento real de la CLI. **Manda sobre `docs/context/`** cuando
  difieran: la intención puede quedar vieja, el comportamiento es lo que la CLI hace.
- `docs/context/03-decisions.md` — la autoridad sobre por qué algo es como es.
- `docs/context/02-architecture-direction.md` y `04-specification.md` — detalle de diseño.

Y esto es archivo histórico, útil para entender cómo se llegó a algo pero no para
orientarse: `01-product-context.md`, `05-roadmap.md`, `06-legacy-baseline.md`,
`07-cli-only-transition-plan.md` y `PROJECT-CONTEXT.md`.

## Propósito del proyecto

Agent Kits será la superficie de **descubrimiento, obtención e instalación** de recursos
reutilizables para agentes:

- skills;
- agents;
- workflows;
- kits;
- y, si se aprueba posteriormente, otros tipos expresamente documentados.

La creación de recursos pertenece a otra superficie de autoría. Agent Kits no tiene
comandos de publicación ni escribe en los repositorios remotos que consume: publicar es un
workflow de CI en el repositorio privado del catálogo (D-039).

## Reglas de trabajo

### Siempre

- Trata los documentos de contexto como especificaciones vivas.
- Actualiza primero la especificación cuando cambie una decisión.
- Conserva la historia y el comportamiento heredado hasta que una tarea aprobada indique
  explícitamente reemplazarlo.
- Mantén separados el catálogo público y las fuentes privadas.
- Distingue identidad de nombre: el `id` es un UUID que no cambia nunca, el `name` es el
  nombre de instalación y se puede renombrar (D-035, D-036).
- Verifica el estado Git y revisa el diff antes de cerrar una tarea.
- Documenta supuestos, riesgos y preguntas abiertas.

### Pregunta antes

- Añadir la primera dependencia externa a `go.mod` (hoy no hay ninguna, y es deliberado).
- Cambiar el formato de `agent-kit.json` o del lockfile. El bloque `migration` es historia
  de proyectos ya migrados: se conserva intacto, no se reinterpreta ni se descarta.
- Cambiar el esquema del lockfile o de los manifests, o su `schema_version`.
- Añadir o renombrar un código de error, un exit code o un flag: son contrato público.
- Eliminar contenido heredado.
- Crear un repositorio remoto o configurar un remoto nuevo.
- Cambiar reglas de versionado, compatibilidad o seguridad.
- Incorporar un nuevo tipo de recurso o un nuevo runtime.

### Nunca

- Añadas un comando `publish` a Agent Kits.
- Añadas un subcomando de Git que escriba a un remoto a la lista blanca de
  `internal/git/git.go`. Esa lista **es** la garantía de "sin escrituras remotas".
- Hagas que la CLI ejecute contenido del catálogo.
- Resuelvas IDs duplicados por precedencia o sobrescritura.
- Sobrescribas un archivo divergente sin `--force` explícito del usuario.
- Mezcles recursos privados dentro del repositorio público.
- Guardes secretos, tokens o credenciales en manifiestos.
- Presentes decisiones todavía abiertas como contratos definitivos.

## Convención semántica de nombres

La convención orienta a autores y agentes, pero no forma parte rígida del esquema:

| Tipo | Representa | Ejemplo |
|---|---|---|
| Skill | disciplina, objeto o capacidad | `frontend-design` |
| Agent | persona, profesión o rol | `frontend-designer` |
| Workflow | proceso o acción | `design-frontend-interface` |
| Kit | colección o solución | `frontend-design-kit` |
| Tool | instrumento o integración | `figma-inspector` |
| Template | artefacto reutilizable | `design-brief-template` |

Una advertencia de naming puede sugerir una alternativa, pero no debe bloquear por sí
sola una contribución válida.

## Idioma

- Documentación y conversación: español.
- IDs, nombres de archivos técnicos, código, mensajes de error y contratos de API:
  inglés.

## Cierre de tareas

Toda tarea debe terminar con:

1. archivos modificados;
2. decisiones tomadas;
3. verificaciones ejecutadas;
4. riesgos o preguntas pendientes;
5. estado Git resultante.

