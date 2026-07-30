# Especificación y plan — transición a CLI única

**Estado:** Fases A–F completadas (2026-07-30) — resta T19 (source privada real) y el cierre T20
**Fecha:** 2026-07-30
**Decisiones:** D-029 a D-033

Este documento es el handoff para el agente que implemente la deprecación de `kits-init`,
la consolidación del estado en el lockfile y la migración posterior a un catálogo mínimo
nativo.

No autoriza a inferir qué recursos deben conservarse. La lista del catálogo es una puerta
humana obligatoria.

## 1. Objetivo

Convertir Agent Kits en un producto con una sola superficie:

- la CLI `agent-kits` descubre, planifica, instala, actualiza, elimina y diagnostica;
- las sources Git continúan siendo el mecanismo de distribución;
- `.agents/agent-kits.lock.json` es la única fuente de verdad del proyecto;
- `workspace.json` se retira mediante una migración explícita, reversible y sin pérdida;
- `kits-init`, `import` y el adaptador heredado desaparecen después de sus ventanas de
  transición;
- el catálogo queda reducido a los recursos que el usuario apruebe y esos recursos usan
  manifests nativos `agent-kit.json`.

### Historias

1. Como usuario con un workspace existente, quiero migrarlo sin perder datos ni archivos.
2. Como agente consumidor, quiero depender de un único archivo de estado y un único
   contrato CLI.
3. Como mantenedor, quiero eliminar el parser y los adaptadores heredados después de
   comprobar que ningún recurso conservado los necesita.
4. Como usuario de sources privadas, quiero conservar la obtención Git y sus garantías de
   privacidad.

## 2. Alcance

### Incluido

- lockfile schema v2;
- comando temporal `migrate`;
- migración de lock v1 y/o `workspace.json`;
- backup lossless de `workspace.json`;
- eliminación de escritura y lectura operativa de `workspace.json`;
- deprecación y posterior retirada de `import`;
- migración de los recursos aprobados a `agent-kit.json`;
- eliminación por lotes de los recursos no aprobados;
- retirada de `kits-init`, loader legacy y parser frontmatter;
- verificación de una source privada real o fixture equivalente autorizado;
- actualización completa de contratos, tests y documentación.

### Fuera de alcance

- elegir recursos sin una lista aprobada;
- eliminar soporte genérico para skill, agent, workflow o kit;
- añadir el adaptador Codex;
- cambiar SemVer, reglas de seguridad o unicidad;
- crear una superficie de autoría o `publish`;
- configurar `origin`;
- eliminar automáticamente backups de usuarios;
- retirar `migrate` en el mismo cambio que lo introduce.

## 3. Stack, comandos y estructura

### Stack

- Go 1.26;
- solo librería estándar;
- Git del sistema, limitado por `internal/git/git.go`;
- JSON para manifests, configuración y lockfile.

### Comandos de verificación

```powershell
go test ./...
go build ./...
git diff --check
git status --short --branch
```

Si se redefine `GOTMPDIR`, debe apuntar fuera del repositorio. De lo contrario,
`internal/git.TestHeadCommitOnNonRepository` puede detectar el `.git` padre y producir un
falso negativo.

### Estructura relevante

```text
cmd/agent-kits/          entrada de la CLI
internal/cli/            comandos y contratos JSON
internal/model/          lockfile y recursos canónicos
internal/workspace/      lock v1/v2; lector del workspace heredado en legacy.go
internal/migrate/        transición a lock v2 (temporal)
internal/journal/        backup y rollback de una operación
internal/install/        aplicación, lockfile y doctor
internal/catalog/        loader de manifests nativos
internal/source/         sources Git y cache
docs/context/            decisiones, spec y plan
skills/, agents/, packs/ catálogo nativo, 75 recursos conservados (D-034)
```

El loader heredado y el parser de frontmatter ya no existen: se eliminaron al completar la
Fase F.

## 4. Contrato objetivo del lockfile v2

El schema exacto se implementará siguiendo esta forma lógica:

```json
{
  "schema_version": 2,
  "project": {
    "id": "uuid-v4",
    "created_at": "2026-07-30T00:00:00Z",
    "stack": {
      "detected": ["go"],
      "source": "user-input",
      "confidence": "high"
    },
    "disciplines": ["tdd"]
  },
  "runtime": "agents",
  "generated_at": "2026-07-30T00:00:00Z",
  "resources": [
    {
      "id": "tdd",
      "type": "skill",
      "source": "public",
      "version": "1.0.0",
      "commit": "0123456789abcdef",
      "checksum": "sha256:...",
      "requested": true,
      "installed_at": "2026-07-30T00:00:00Z",
      "files": [
        {
          "path": ".agents/skills/tdd/SKILL.md",
          "checksum": "sha256:..."
        }
      ]
    }
  ],
  "migration": {
    "source": "workspace.json",
    "source_schema_version": 2,
    "migrated_at": "2026-07-30T00:00:00Z",
    "legacy_updated_at": "2026-07-30T00:00:00Z",
    "legacy_system_version": "0.1.0",
    "legacy_pack": {},
    "legacy_flags": {},
    "legacy_structure": [],
    "extra": {},
    "backup": ".agents/workspace.json.migrated.bak"
  }
}
```

Reglas:

- `project.id` y `project.created_at` son estables;
- `project.disciplines` es explícito: puede afectar comportamiento y no se descarta;
- `resources[].installed_at` preserva la fecha heredada cuando existe;
- `migration` es opcional y solo aparece cuando hubo datos heredados;
- los campos desconocidos de `workspace.json` se conservan en `migration.extra` como
  `json.RawMessage`, sin reinterpretarlos;
- los campos derivables se recalculan para operar, pero su valor heredado se conserva en
  `migration` cuando sea histórico;
- el backup contiene los bytes originales, no una reserialización;
- el backup nunca se sobrescribe si ya existe con contenido distinto;
- el lockfile v1 se lee durante la transición, pero toda escritura produce v2;
- un schema desconocido falla con `workspace_invalid`.

## 5. Matriz de migración

| Estado inicial | Resultado |
|---|---|
| lock v2, sin workspace | no-op idempotente |
| lock v1, sin workspace | upgrade determinístico a lock v2 |
| sin lock, workspace válido | adopción conservadora + lock v2 + backup + retirada |
| lock v1, workspace válido | merge fail-closed + lock v2 + backup + retirada |
| lock v2, workspace válido, sin marca | migrar metadatos compatibles; conflicto bloquea |
| lock v2 migrado, workspace idéntico al backup | completar retirada interrumpida |
| sin lock y sin workspace | no hay nada que migrar; no escribir |
| workspace inválido | abortar con `workspace_invalid` |
| backup distinto ya existente | abortar; nunca sobrescribir |
| recurso sin lock que no coincide con catálogo | informar y abortar |
| archivo administrado divergente | abortar con `local_divergence` |

### Precedencia durante el merge

No se resuelven contradicciones por precedencia silenciosa:

- el lock existente prueba propiedad, versión, checksum y archivos;
- `workspace.json` aporta identidad del proyecto, stack, disciplinas, timestamps y
  metadatos históricos;
- si ambos declaran el mismo hecho con valores incompatibles, el plan se bloquea;
- un dato derivable puede recalcularse, pero el original se conserva en `migration`;
- los campos desconocidos siempre se preservan.

## 6. Contrato de `agent-kits migrate`

```text
agent-kits migrate --project <path> [--yes] [--json]
```

- sin `--yes`, calcula y presenta el plan sin escribir;
- en modo no interactivo o JSON, aplicar requiere `--yes`;
- usa los exit codes existentes;
- usa `workspace_invalid`, `local_divergence`, `integrity_mismatch` y
  `confirmation_required`; no añade códigos sin una decisión nueva;
- no acepta `--force`: una migración de estado no debe descartar datos para continuar;
- JSON emite un único envelope;
- la respuesta informa origen, schema inicial/final, recursos adoptados, campos
  preservados, backup y archivos retirados;
- lockfile, backup y retirada de `workspace.json` se aplican bajo el journal existente;
- al fallar cualquier escritura, se restaura exactamente el estado inicial.

### Transición de `import`

- durante una ventana, `import` delega en la misma lógica que `migrate`;
- la salida humana advierte que está deprecado;
- el JSON agrega datos, no rompe el envelope existente;
- la documentación recomienda exclusivamente `migrate`;
- `import` se elimina solo en una tarea futura aprobada.

## 7. Comportamiento durante la transición

- instalaciones nuevas crean únicamente lock v2;
- ningún comando normal vuelve a crear `workspace.json`;
- si un comando mutante encuentra `workspace.json`, bloquea con `workspace_invalid` y
  sugiere `agent-kits migrate --project <path>`;
- `doctor` informa que la migración es necesaria sin añadir un error público nuevo;
- `list`, `plan` e `info` siguen siendo de solo lectura;
- `migrate` es idempotente;
- `.agents/` se mantiene como layout portable, ahora propiedad de la CLI;
- eliminar `workspace.json` no cambia las rutas de recursos ni del lockfile.

## 8. Catálogo nativo mínimo

### Gate obligatorio

**Superado el 2026-07-30 (D-034).** La tabla aprobada es uniforme, así que se expresa por
clase en lugar de enumerar 75 filas idénticas:

| Clase | Cantidad | Acción | Versión inicial | Dependencias | Motivo |
|---|---:|---|---|---|---|
| skills (`skills/<id>/`) | 50 | keep | declarada, o `1.0.0` | las que declara el frontmatter | podar es barato y reversible; decidir qué se pierde no |
| kits (`packs/<id>/`) | 7 | keep | declarada | ídem | ídem |
| agentes globales (`agents/`) | 4 | keep + mover a directorio propio | `1.0.0` | ídem | el loader nativo asocia un manifest a su directorio |
| agentes de kit (`packs/<kit>/agents/`) | 7 | keep + mover a directorio propio | `1.0.0` | ídem | ídem |
| workflows de kit (`packs/<kit>/workflows/`) | 7 | keep + mover a directorio propio | `1.0.0` | ídem | ídem |
| `backend/feature-development` | 1 | keep + renombrar **archivo** a `backend-feature-development.md` | `1.0.0` | ídem | el destino colisionaba, no la identidad (D-028) |
| `frontend/feature-development` | 1 | keep + renombrar **archivo** a `frontend-feature-development.md` | `1.0.0` | ídem | ídem |

Ningún recurso se elimina. Mover un recurso a su propio directorio cambia el layout de la
source, **no** el destino de instalación: la ruta instalada se deriva del nombre de archivo
declarado, no de su ubicación en el repositorio.

La poda queda para una tarea futura, ya sin bloquear la retirada del legado.

### Reglas para recursos conservados

- cada recurso tiene `agent-kit.json` con `schema_version`;
- IDs y versiones son explícitos;
- toda dependencia usa ID canónico;
- los archivos declarados existen y no escapan del root;
- dos recursos no escriben el mismo destino;
- los recursos del kit usan `<kit>/<name>` cuando corresponde;
- el catálogo no contiene duplicados ni referencias no resueltas;
- cada recurso conserva licencia y atribución aplicables;
- el contenido no depende de `kits-init` ni de `workspace.json`.

## 9. Estilo de código

Mantener tipos de dominio explícitos, errores estructurados y funciones pequeñas:

```go
func (l *Lock) Validate() error {
	if l.SchemaVersion != LockSchemaVersion {
		return errs.New(
			errs.CodeWorkspaceInvalid,
			"unsupported lockfile schema_version %d",
			l.SchemaVersion,
		)
	}
	return nil
}
```

- nombres técnicos y errores en inglés;
- documentación y decisiones en español;
- comentarios explican invariantes;
- salida ordenada y determinística;
- timestamps inyectables en tests;
- no usar `panic` para entradas de usuario;
- ninguna dependencia nueva en `go.mod`.

## 10. Estrategia de testing

### Unit

- parseo y validación de lock v1/v2;
- mapping lossless de cada campo de `workspace.json`;
- preservación de `json.RawMessage`;
- conflictos lock/workspace;
- idempotencia;
- manifests nativos.

### Integration

- cada fila de la matriz de migración;
- backup byte a byte;
- rollback en cada punto de fallo;
- `import` como alias deprecado;
- comandos mutantes bloqueados antes de migrar;
- comandos normales sin escritura de `workspace.json`;
- source local nativa y source Git cacheada.

### Security

- symlink en workspace, lock o backup;
- path traversal;
- backup preexistente divergente;
- workspace con secretos en campos extra;
- tamaño máximo y JSON malicioso;
- ninguna ejecución de contenido.

### End-to-end

1. Crear fixture heredado sin lock.
2. Ejecutar `migrate` sin `--yes` y comprobar cero escrituras.
3. Ejecutar con `--yes`.
4. Comprobar lock v2, backup exacto y ausencia de `workspace.json`.
5. Ejecutar `doctor`, `list`, `plan` e instalación idempotente.
6. Repetir `migrate` y comprobar no-op.
7. Fallar una escritura inyectada y comprobar rollback total.
8. Instalar el catálogo nativo mínimo y verificar que no se carga legacy.

## 11. Límites

### Siempre

- actualizar decisiones/spec antes del comportamiento;
- planificar antes de escribir;
- aplicar migración con journal;
- preservar bytes originales;
- mantener JSON único y determinístico;
- ejecutar tests, build, diff check y revisar Git.

### Preguntar primero

- cambiar el JSON propuesto del lock v2;
- añadir o renombrar errores, exit codes o flags;
- eliminar `migrate` o `import`;
- borrar recursos sin la tabla aprobada;
- renombrar IDs;
- añadir dependencias o runtimes;
- configurar `origin`;
- eliminar backups.

### Nunca

- escribir en un remoto;
- ejecutar contenido del catálogo;
- inferir la lista mínima;
- sobrescribir un backup divergente;
- borrar `workspace.json` antes de validar lock y backup;
- ignorar campos desconocidos;
- resolver conflictos por precedencia;
- modificar `upstream`.

## 12. Plan de implementación

```text
Decisiones/spec
      │
      ▼
Lock v2 ──► lector v1/v2 ──► plan de migración ──► journal
                                                    │
                                                    ▼
                                      CLI migrate + import deprecado
                                                    │
                                                    ▼
                                      CLI deja workspace.json
                                                    │
                              ┌─────────────────────┴─────────────────────┐
                              ▼                                           ▼
                  Gate de catálogo aprobado                    Source privada real
                              │
                              ▼
                  manifests nativos + poda
                              │
                              ▼
                 retirar loader/frontmatter/kits-init
                              │
                              ▼
                     verificación y documentación
```

### Fase A — Contratos

Actualizar especificación, roadmap y CLI antes del comportamiento.

Gate: los documentos distinguen con claridad estado actual, transición y destino.

### Fase B — Lock v2

Implementar structs, lector v1/v2, upgrade determinístico y validación.

Gate: fixtures válidos e inválidos en verde; toda escritura produce v2.

### Fase C — Migración lossless

Implementar planner puro, mapping, backup, journal, `migrate` e `import` deprecado.

Gate: matriz completa, rollback e idempotencia en verde.

### Fase D — CLI independiente

Dejar de escribir workspace, bloquear mutaciones pendientes y operar solo con lock v2.

Gate: ningún test no transicional depende de `workspace.json`.

### Fase E — Catálogo mínimo

Bloqueada hasta recibir la lista. Migrar recursos aprobados, versionarlos y podar el resto.

Gate: catálogo aprobado, único y sin colisiones.

### Fase F — Retirada legacy

Eliminar loader, frontmatter y superficie activa `kits-init`; reclasificar historia.

Gate: `rg` solo encuentra referencias de migración temporal o historia permitida.

### Fase G — Profundización y cierre

Verificar source privada, E2E, documentación, build, tests y Git.

## 13. Tareas para el agente implementador

Cada tarea debe caber en una sesión y tocar, como regla, no más de cinco archivos.

- [x] T01 — Alinear especificación principal y roadmap
  - Acceptance: D-029 a D-033 se reflejan sin presentar la transición como implementada.
  - Verify: `rg -n "kits-init|workspace.json|D-022|D-026" docs`.
  - Files: `docs/context/04-specification.md`, `docs/context/05-roadmap.md`, `docs/cli.md`.

- [x] T02 — Modelar lockfile v2
  - Acceptance: structs y validación cubren `project`, `installed_at` y `migration`.
  - Verify: `go test ./internal/model`.
  - Files: `internal/model/model.go`, `internal/model/model_test.go`.

- [x] T03 — Leer lock v1 y convertirlo en memoria a v2
  - Acceptance: v1 se acepta, schemas desconocidos fallan y escribir produce v2.
  - Verify: `go test ./internal/workspace`.
  - Files: `internal/workspace/workspace.go`, `internal/workspace/workspace_test.go`.

- [x] T04 — Extraer lector lossless de workspace heredado
  - Acceptance: campos conocidos/desconocidos y bytes originales se conservan.
  - Verify: tests round-trip con campos extra.
  - Files: `internal/workspace/workspace.go`, `internal/workspace/workspace_test.go`.

- [x] T05 — Construir planner puro de migración
  - Acceptance: implementa toda la matriz de §5 sin tocar disco.
  - Verify: tests table-driven.
  - Files: `internal/migrate/migrate.go`, `internal/migrate/migrate_test.go`,
    `internal/internaltest/fixtures.go`.

- [x] T06 — Implementar backup y aplicación journalizada
  - Acceptance: lock, backup y retirada son atómicos; backup divergente bloquea.
  - Verify: tests de fallo inyectado.
  - Files: `internal/migrate/apply.go`, `internal/migrate/apply_test.go`,
    `internal/install/journal.go`, `internal/install/journal_test.go`.

- [x] T07 — Exponer `agent-kits migrate`
  - Acceptance: flags aprobados, envelope único, confirmación y salida humana/JSON.
  - Verify: `go test ./internal/cli -run Migrate`.
  - Files: `internal/cli/cli.go`, `internal/cli/migrate_cmd.go`,
    `internal/cli/cli_test.go`.

- [x] T08 — Convertir `import` en alias deprecado
  - Acceptance: delega en migración, no duplica lógica ni escribe workspace.
  - Verify: tests de compatibilidad humana y JSON.
  - Files: `internal/cli/project_cmds.go`, `internal/cli/cli_test.go`,
    `internal/install/import.go`, `internal/install/install_test.go`.

- [x] T09 — Bloquear mutaciones antes de migrar
  - Acceptance: install/update/remove detectan workspace pendiente y no escriben.
  - Verify: tests por comando con hint de migrate.
  - Files: `internal/cli/project_cmds.go`, `internal/cli/cli_test.go`,
    `internal/install/doctor.go`, `internal/install/install_test.go`.

- [x] T10 — Dejar de generar `workspace.json`
  - Acceptance: instalaciones y updates escriben solo lock v2.
  - Verify: E2E confirma ausencia de workspace.
  - Files: `internal/install/install.go`, `internal/install/install_test.go`,
    `internal/plan/plan.go`, `internal/plan/plan_test.go`.

- [x] T11 — Retirar workspace del contrato de adapters
  - Acceptance: `.agents/` y lock path permanecen; workspace solo existe en migración.
  - Verify: `go test ./internal/adapter ./internal/plan`.
  - Files: `internal/adapter/adapter.go`, `internal/adapter/adapter_test.go`,
    `internal/plan/plan.go`, `internal/plan/plan_test.go`.

- [x] T12 — Verificar transición completa
  - Acceptance: matriz, rollback, idempotencia, doctor y comandos verdes.
  - Verify: `go test ./...` y `go build ./...`.
  - Files: solo fixtures/tests; no cambiar contratos aquí.

- [x] GATE — Obtener tabla aprobada del catálogo mínimo
  - Acceptance: cada recurso tiene acción, ID, tipo, versión y dependencias.
  - Verify: aprobación explícita registrada.
  - Files: este documento o inventario dedicado.

- [x] T13+ — Crear manifests nativos para recursos conservados
  - Acceptance: máximo cuatro recursos por tarea y todos cargan nativamente.
  - Verify: tests de catálogo y búsqueda JSON sobre fixture/source.
  - Files: manifests y tests, máximo cinco por tarea.

- [x] T14+ — Eliminar recursos no conservados por lotes
  - Acceptance: cada lote sigue la tabla y no deja referencias colgantes.
  - Verify: auditoría del catálogo después de cada lote.
  - Files: máximo cinco recursos/archivos por tarea.

- [x] T15 — Demostrar que el catálogo ya no necesita legacy
  - Acceptance: todos los recursos finales cargan como nativos; inventario esperado.
  - Verify: test de carga nativa y búsqueda completa.
  - Files: `internal/catalog/catalog_test.go`, fixtures y documento de inventario.

- [x] T16 — Retirar loader legacy
  - Acceptance: `legacy.go` y el campo `Legacy` desaparecen.
  - Verify: `go test ./internal/catalog ./internal/model ./internal/cli`.
  - Files: `internal/catalog/legacy.go`, `internal/catalog/catalog.go`,
    `internal/catalog/catalog_test.go`, `internal/model/model.go`,
    `internal/cli/catalog_cmds.go`.

- [x] T17 — Retirar parser frontmatter si queda huérfano
  - Acceptance: `rg "frontmatter"` no muestra consumidores operativos.
  - Verify: `go test ./...`.
  - Files: `internal/frontmatter/frontmatter.go`,
    `internal/frontmatter/frontmatter_test.go` y referencias restantes.

- [x] T18 — Retirar superficie activa `kits-init`
  - Acceptance: README instala la CLI; no se ofrece `/kits-init`; la historia permanece.
  - Verify: `rg -n "/kits-init|name: kits-init" . -g '!.git/**'`.
  - Files: `SKILL.md`, `repair-upgrade.md`, `workspace-schema.md`, `README.md`,
    `PROJECT-CONTEXT.md`.

- [ ] T19 — Verificar source privada
  - Acceptance: autorizada sincroniza; no autorizada no filtra y devuelve error estable.
  - Verify: integración documentada sin credenciales en logs o fixtures.
  - Files: `internal/source/source_test.go`, fixture/config segura y roadmap.

- [ ] T20 — Cierre documental y de calidad
  - Acceptance: docs describen solo CLI, transición marcada y suite verde.
  - Verify: comandos de §3, diff, inventario y criterios de §15.
  - Files: `README.md`, `docs/cli.md`, `docs/context/README.md`,
    `docs/context/04-specification.md`, `docs/context/05-roadmap.md`.

## 14. Paralelización

Permitido:

- después de T03, T04 puede avanzar en paralelo con el diseño puro de T05;
- T01 puede avanzar en paralelo con lock v2;
- T19 puede ejecutarse en paralelo si no cambia contratos.

Secuencial:

- T02 → T03 → T05 → T06 → T07;
- T07 → T08/T09 → T10/T11 → T12;
- gate humano → manifests → poda → retirada legacy;
- nunca retirar loader antes de demostrar catálogo nativo completo.

## 15. Criterios de éxito

- [x] Todo estado nuevo vive en lockfile v2.
- [x] Ningún comando normal crea o actualiza `workspace.json`.
- [x] La migración conserva datos conocidos, desconocidos y bytes originales.
- [x] Una falla deja lock, workspace, backup y recursos como estaban.
- [x] Repetir `migrate` no cambia archivos.
- [x] Proyectos con lock v1 pueden actualizarse.
- [x] `import` está deprecado y comparte implementación con `migrate`.
- [x] El catálogo final coincide exactamente con la lista aprobada.
- [x] Todos los recursos finales usan `agent-kit.json`.
- [x] Loader legacy y frontmatter se eliminan cuando quedan sin consumidores.
- [x] La CLI conserva los cuatro tipos.
- [ ] Sources Git públicas y privadas mantienen sus garantías. *(privada: pendiente T19)*
- [x] No existe operación de escritura remota.
- [x] `go test ./...`, `go build ./...` y `git diff --check` pasan.
- [ ] Git contiene solo cambios intencionales revisados.

## 16. Riesgos y mitigaciones

| Riesgo | Mitigación |
|---|---|
| perder campos desconocidos | `json.RawMessage` + backup byte a byte |
| borrar workspace antes del lock | journal y validación previa |
| dos fuentes de verdad | comandos normales bloquean hasta migrar |
| romper locks v1 | lector v1 y escritura v2 |
| retirar legacy demasiado pronto | gate y prueba de catálogo nativo |
| eliminar recurso necesario | tabla humana obligatoria |
| deuda temporal permanente | tarea futura explícita de sunset |
| filtrar source privada | prueba autorizada/no autorizada y logs neutros |
| plan demasiado grande | tareas de máximo cinco archivos y gates |

## 17. Sunset posterior

No pertenece a esta implementación inmediata. Cuando el usuario confirme que la ventana
terminó, un cambio nuevo deberá:

1. eliminar `migrate`, `import` y el lector de `workspace.json`;
2. decidir si `migration` permanece como historia o evoluciona a otro schema sin pérdida;
3. mantener los backups como propiedad del usuario;
4. actualizar decisiones, CLI y tests antes de eliminar código.
