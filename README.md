# agent-kits

**Obtención e instalación de capacidades para agentes.** Una CLI en Go, sin dependencias
externas, que descubre, planifica e instala skills, agentes, workflows y kits desde
repositorios Git públicos y privados, de forma reproducible y auditable.

Compatible con **Claude Code**, **OpenCode** y cualquier runtime que lea `.agents/`.

## Instalación

```powershell
go build -o agent-kits.exe ./cmd/agent-kits
```

El binario es estático y no requiere runtime instalado. La única dependencia externa es el
`git` del sistema, y solo para sincronizar sources remotas.

## Uso

```powershell
agent-kits source add public https://github.com/<owner>/<repo>.git
agent-kits source sync public
agent-kits search frontend
agent-kits plan frontend-design --project .
agent-kits install frontend-design --project . --yes
agent-kits doctor --project .
```

Todo comando acepta `--json` y devuelve un envelope estable con códigos de error
documentados, para que un agente pueda consumirlo sin parsear texto.

La referencia completa —comandos, contratos JSON, exit codes, política de conflictos y
seguridad— está en [`docs/cli.md`](./docs/cli.md).

## Qué hay en el catálogo

Este repositorio también es una source: sus 75 recursos se describen con manifests
`agent-kit.json` y se instalan con la CLI.

- **7 kits** — composiciones temáticas de skills, agentes y workflows.
- **50 skills** — unidades de capacidad agnósticas al flujo (incluye el ecosistema SDD).
- **11 agentes** — 4 globales y 7 propiedad de un kit.
- **7 workflows** — procesos completos de extremo a extremo.
- **5 disciplinas** combinables — TDD, BDD, contract-first, trunk-based, SDD.

Ver el [índice del catálogo](./catalog-index.md) para la lista completa.

## Estructura instalada

```text
.agents/
├── agent-kits.lock.json    ← estado del proyecto: qué hay instalado, de dónde y con qué checksum
├── skills/<id>/…
├── agents/<id>.md
├── workflows/<id>.md
└── packs/<id>/pack.md
```

## Migrar desde `kits-init`

La skill conversacional `/kits-init` está **retirada** (D-029): la CLI es la única
superficie que evoluciona. Un workspace creado por ella se adopta una vez, sin pérdida de
datos y dejando una copia de seguridad:

```powershell
agent-kits migrate --project .          # muestra el plan, no escribe
agent-kits migrate --project . --yes    # lo aplica
```

El detalle está en [`docs/cli.md`](./docs/cli.md#migración-desde-kits-init).

## Documentación

| Documento | Contenido |
|---|---|
| [`docs/cli.md`](./docs/cli.md) | Comportamiento de la CLI: contratos, errores, seguridad. |
| [`docs/context/`](./docs/context/README.md) | Decisiones, especificación y roadmap. |
| [`AGENTS.md`](./AGENTS.md) | Lectura obligatoria para agentes que modifiquen el repositorio. |
| [`PROJECT-CONTEXT.md`](./PROJECT-CONTEXT.md) | Historia del sistema `kits-init` que precedió a la CLI. |

## Garantías

- La CLI **nunca** ejecuta contenido del catálogo.
- **Nunca** escribe en un remoto: `git` se invoca con una lista blanca de subcomandos de
  solo lectura, verificable en `internal/git/git.go`.
- No sobrescribe en silencio: un archivo modificado localmente bloquea la operación.
- Toda escritura es journalizada y reversible: un fallo restaura el estado anterior.

## Licencia

MIT
