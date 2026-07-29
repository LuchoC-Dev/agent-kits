# Baseline heredado — Agent Kits

**Estado:** Fact finding inicial
**Fecha de inspección:** 2026-07-29
**Commit base:** `912e1bd`

## 1. Origen

```text
Repositorio remoto: https://github.com/LuchoC-Dev/agent-kits.git
Rama base: main
Commit: 912e1bd
Mensaje: refactor: merge agent.md into SKILL.md (drop thin launcher pattern)
```

El nuevo workspace fue clonado con historial independiente y `upstream` sin push.

## 2. Inventario comprobado

Conteos obtenidos directamente del filesystem:

| Recurso | Cantidad |
|---|---:|
| Directorios de skills | 50 |
| Packs | 7 |
| Agentes globales | 4 |

Packs encontrados:

- `context`;
- `design`;
- `backend-design`;
- `fullstack-design`;
- `frontend`;
- `backend`;
- `tools`.

Agentes globales:

- `artifact-validator`;
- `design-critic`;
- `research-scout`;
- `wireframe-renderer`.

## 3. Estructura heredada

```text
agent-kits/
├── SKILL.md
├── README.md
├── PROJECT-CONTEXT.md
├── catalog-index.md
├── workspace-schema.md
├── repair-upgrade.md
├── skills/
├── agents/
├── packs/
└── meta/
```

## 4. Comportamiento actual

El punto de entrada `kits-init`:

1. detecta runtime y stack;
2. lee el catálogo;
3. pregunta qué packs o skills instalar;
4. copia recursos a `.agents/`;
5. escribe `workspace.json`;
6. ofrece repair/upgrade para workspaces existentes.

El sistema es Markdown-first y asume que el agente anfitrión interpreta y ejecuta las
instrucciones.

## 5. Decisiones heredadas valiosas

- Skills agnósticas al workflow.
- Packs como composiciones declarativas.
- Pool global de skills.
- Agentes de workflow dentro del pack.
- Agentes reutilizables en pool global.
- Workflows responsables del orden.
- `produces` y `consumes` para integrar artefactos.
- `.agents/` como destino portable.
- Meta-agentes separados del contenido distribuido.
- No introducir un motor de compatibilidad innecesario.

## 6. Diferencias con la nueva dirección

| Heredado | Dirección nueva |
|---|---|
| Skill conversacional de bootstrap | CLI determinística para agentes y personas |
| Catálogo embebido en una skill | Sources configurables |
| Copia directa | Plan, checksums y lockfile |
| Runtimes detectados en prompt | Adaptadores explícitos |
| Un catálogo local | Fuentes públicas y privadas |
| Sin identidad global entre sources | IDs globalmente únicos |
| Repair/upgrade guiado por agente | Operaciones reproducibles y estructuradas |
| Sin frontera de publicación explícita | Sources remotas de solo lectura |

## 7. Hallazgos iniciales

### Conteos documentales

El README heredado menciona 51 skills, mientras el filesystem contiene 50 directorios de
skills. Esto debe auditarse; no se corregirá hasta determinar si falta un recurso, si una
entrada es composición o si el conteo quedó obsoleto.

> **Resuelto** en `§8`, Hallazgo 5: el conteo del README está obsoleto. No falta nada.

### Contexto histórico

`PROJECT-CONTEXT.md` conserva decisiones útiles, pero describe rutas y estados de mayo de
2026. Debe tratarse como historia, no como descripción automáticamente vigente.

### Git

El repositorio fuente presenta protección de `safe.directory` dentro del sandbox debido
a la diferencia entre el propietario del archivo y el usuario de ejecución. Esto no es
un defecto del repositorio, pero debe conocerse durante automatizaciones locales.

## 8. Auditoría completada

**Fecha:** 2026-07-29
**Método:** el adaptador de catálogo (D-026) carga el layout heredado y sintetiza
manifests canónicos. La auditoría es ejecutable: `agent-kits search --json` la reproduce.

### Inventario canónico

El catálogo heredado produce **75 recursos**, todos con ID único:

| Tipo | Cantidad | Origen |
|---|---:|---|
| Skill | 50 | `skills/<id>/` |
| Agent | 11 | 4 en `agents/` + 7 en `packs/*/agents/` |
| Workflow | 7 | `packs/*/workflows/` |
| Kit | 7 | `packs/<id>/pack.md` |

Los 75 bloques de frontmatter parsean sin error. El grafo de dependencias se deriva
completo, sin referencias colgantes: cada `skills`, `agents`, `workflows`, `depends_on`,
`uses_agents`, `composes` y `steps[].skill` resuelve a un recurso existente.

### Hallazgo 1 — colisiones de identidad (motivó D-019)

Tres nombres colisionan si se asume un espacio de nombres plano:

| Nombre | Colisión |
|---|---|
| `feature-development` | workflow en `packs/backend` **y** en `packs/frontend`, con contenido distinto |
| `backend-design` | pack `backend-design` y su workflow homónimo |
| `fullstack-design` | pack `fullstack-design` y su workflow homónimo |

La propiedad de kit (`<kit>/<name>`) las resuelve sin renombrar nada. Una referencia corta
a `feature-development` es ambigua y falla cerrada.

### Hallazgo 2 — colisión de destino (motivó D-028)

Instalar `backend` y `frontend` juntos hace que ambos escriban
`.agents/workflows/feature-development.md` con contenido diferente. La regla heredada
("si ya fue instalado por otro pack, no lo copies de nuevo") conservaba silenciosamente el
primero y perdía el otro. **Es un defecto del catálogo, no de la herramienta**, y hay que
corregirlo en el catálogo: renombrar uno de los dos workflows según la convención D-008.

Verificado: `agent-kits install backend frontend` se bloquea con `destination_conflict`.

### Hallazgo 3 — referencias mutuas (motivó D-027)

Cuatro packs contienen un ciclo agente ↔ workflow: `context`, `design`, `backend-design`
y `fullstack-design`. Son asociaciones legítimas, no dependencias de orden.

### Hallazgo 4 — versiones ausentes

Los packs declaran `version`; las skills, los agentes y los workflows no. El adaptador les
asigna `0.0.0`, salvo cuando `metadata.version` trae un valor normalizable (12 skills de
terceros declaran `"2.0"` → `2.0.0`). Sin versiones reales, `update` no puede distinguir
cambios de contenido dentro de una misma versión: **corregir esto requiere versionar el
catálogo**, y es la razón principal para migrar a manifests nativos.

### Hallazgo 5 — conteo del README

La contradicción documental de `§7` queda cerrada: `catalog-index.md` lista exactamente
las **50** skills que existen en el filesystem, sin faltantes, sobrantes ni duplicados. El
único dato incorrecto es el README heredado, que dice 51. No falta ningún recurso.

### Verificación de extremo a extremo

Sobre el catálogo real, en un proyecto vacío:

- instalar los 6 kits compatibles → 56 recursos, 58 archivos;
- repetir la instalación → plan vacío, lockfile intacto;
- `doctor` → sin problemas;
- quitar `fullstack-design` → borra sus 4 archivos propios y conserva las 19 skills
  compartidas con `design` y `backend-design`;
- importar un workspace creado por `kits-init` → adopta sus recursos y preserva los campos
  ajenos de `workspace.json`.

## 9. Qué sigue pendiente

- Renombrar uno de los dos `feature-development` para eliminar la colisión de destino.
- Versionar los recursos heredados (Hallazgo 4).
- Corregir el conteo del README y las omisiones de `catalog-index.md`.
- Revisar licencia y atribución de las 12 skills de terceros (`metadata.author`).
- Decidir si el catálogo migra a manifests nativos o permanece en el layout heredado.
- Determinar qué pruebas históricas existen fuera del repositorio distribuido.
