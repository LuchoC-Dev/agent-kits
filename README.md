# agent-kits

**AI-first workspace bootstrapper.** Inicializa un workspace `.agents/` en cualquier proyecto con un catálogo curado de skills, packs, agentes y disciplinas de desarrollo. Cross-compatible **Claude Code** ↔ **OpenCode**.

## Instalación

### Con `npx skills` (recomendado)

```bash
npx skills add LuchoC-Dev/agent-kits
```

### Manual

Cloná y copiá la carpeta a tu directorio de skills:

```bash
git clone https://github.com/LuchoC-Dev/agent-kits.git
# Claude Code:
cp -r agent-kits ~/.claude/skills/kits-init
# OpenCode:
cp -r agent-kits ~/.config/opencode/skills/kits-init
```

## Uso

Desde un proyecto cualquiera, invocá el comando:

```
/kits-init
```

El agente te guía con preguntas para:

1. Detectar el stack del proyecto (React, Node, Python, Java, etc.).
2. Elegir packs (`context`, `design`, `backend-design`, `fullstack-design`, `backend`, `frontend`, `tools`).
3. Elegir skills sueltas si querés custom.
4. Activar disciplinas de desarrollo (`tdd`, `bdd`, `contract-first`, `trunk-based`, `sdd`).

Genera `.agents/` con `workspace.json`, skills, agentes y workflows listos para trabajar.

## Qué hay adentro

- **7 packs** — composiciones temáticas de skills+agentes+workflows.
- **51 skills** — unidades de capacidad agnósticas al flujo (incluye ecosistema SDD).
- **4 agentes globales** — `artifact-validator`, `design-critic`, `research-scout`, `wireframe-renderer`.
- **5 disciplinas** combinables — TDD, BDD, contract-first, trunk-based, SDD.

Ver el [índice del catálogo](./catalog-index.md) para la lista completa.

## Runtimes soportados

| Runtime | Tool de preguntas usada |
|---|---|
| **Claude Code** | `AskUserQuestion` (nativa) |
| **OpenCode** | `question` (nativa) |
| Otros | Chat plano numerado (fallback) |

La detección es automática en Fase 1 del agente.

## Estructura del workspace generado

```
.agents/
├── workspace.json          ← runtime, packs, skills, agentes, disciplinas
├── skills/<id>/SKILL.md    ← skills instaladas (solo las elegidas)
├── agents/<id>.md          ← agentes orquestadores y de tarea
├── workflows/<id>.md       ← workflows del/los pack(s) elegido(s)
└── packs/<id>/pack.md      ← registro de los packs instalados
```

## Documentación interna

Para entender la arquitectura, las decisiones de diseño y el glosario completo, ver [PROJECT-CONTEXT.md](./PROJECT-CONTEXT.md).

## Licencia

MIT
