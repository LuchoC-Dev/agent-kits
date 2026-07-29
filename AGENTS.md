# Agent Kits Next — guía para agentes

Este repositorio es un **fork de trabajo** de `LuchoC-Dev/agent-kits`. Su propósito es
evolucionar Agent Kits sin modificar directamente el proyecto original.

## Estado actual

- Fase: **MVP implementado y verificado** (Fases 0–8; la 6 queda parcial).
- Rama de trabajo: `planning/agent-kits-next`, mergeada a `main`.
- `upstream`: repositorio original, configurado sin push.
- `origin`: todavía no existe; se definirá cuando haya un repositorio colaborativo.
- Stack: **Go, solo librería estándar** (D-016). `go.mod` no declara `require`.
- El catálogo heredado **no se modificó**: se consume mediante un adaptador (D-026).

La CLI vive en `cmd/agent-kits` e `internal/`. Su referencia es `docs/cli.md`.

```powershell
go build ./...
go test ./...
```

## Lectura obligatoria

Antes de proponer o realizar cambios, lee en este orden:

1. `docs/context/README.md`
2. `docs/context/01-product-context.md`
3. `docs/context/02-architecture-direction.md`
4. `docs/context/03-decisions.md`
5. `docs/context/04-specification.md`
6. `docs/context/05-roadmap.md`
7. `docs/context/06-legacy-baseline.md`

Después `docs/cli.md` para el contrato implementado, y `PROJECT-CONTEXT.md` para el sistema
heredado. Cuando el contexto heredado contradiga los documentos de `docs/context/`,
prevalece `docs/context/` para la nueva versión.

## Propósito del proyecto

Agent Kits será la superficie de **descubrimiento, obtención e instalación** de recursos
reutilizables para agentes:

- skills;
- agents;
- workflows;
- kits;
- y, si se aprueba posteriormente, otros tipos expresamente documentados.

La creación de recursos pertenece a otra superficie de autoría. Agent Kits no tendrá
comandos de publicación ni escribirá en los repositorios remotos que consume.

## Reglas de trabajo

### Siempre

- Trata los documentos de contexto como especificaciones vivas.
- Actualiza primero la especificación cuando cambie una decisión.
- Conserva la historia y el comportamiento heredado hasta que una tarea aprobada indique
  explícitamente reemplazarlo.
- Mantén separados el catálogo público y las fuentes privadas.
- Usa IDs canónicos globalmente únicos.
- Verifica el estado Git y revisa el diff antes de cerrar una tarea.
- Documenta supuestos, riesgos y preguntas abiertas.

### Pregunta antes

- Añadir la primera dependencia externa a `go.mod` (hoy no hay ninguna, y es deliberado).
- Cambiar el formato actual de `SKILL.md`, `pack.md` o `workspace.json`.
- Cambiar el esquema del lockfile o de los manifests, o su `schema_version`.
- Añadir o renombrar un código de error, un exit code o un flag: son contrato público.
- Eliminar contenido heredado.
- Crear o configurar un remoto `origin`.
- Cambiar reglas de versionado, compatibilidad o seguridad.
- Incorporar un nuevo tipo de recurso o un nuevo runtime.

### Nunca

- Hagas push a `upstream`.
- Añadas un comando `publish` a Agent Kits.
- Añadas un subcomando de Git que escriba a un remoto a la lista blanca de
  `internal/git/git.go`. Esa lista **es** la garantía de "sin escrituras remotas".
- Hagas que la CLI ejecute contenido del catálogo.
- Resuelvas IDs duplicados por precedencia o sobrescritura.
- Sobrescribas un archivo divergente sin `--force` explícito del usuario.
- Mezcles recursos privados dentro del repositorio público.
- Guardes secretos, tokens o credenciales en manifiestos.
- Modifiques el repositorio original desde este fork.
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

