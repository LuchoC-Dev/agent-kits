# Fase 6 — Workspace existente (Repair / Upgrade / Install / Reindex)

Leé este archivo **solo si llegaste a Fase 6** desde `SKILL.md` — es decir, si
`.agents/workspace.json` ya existe al arrancar `/kits-init`. No es parte del contexto
cargado por defecto.

## Punto de entrada

Mostrá una pregunta estructurada (tool del runtime) con las 4 opciones derivadas del
estado actual del workspace:

- **Repair** — recrea carpetas faltantes sin tocar contenido existente.
- **Upgrade** — actualiza el schema de `workspace.json` si cambió la versión.
- **Install packs** — agrega un pack nuevo encima del workspace actual.
- **Reindex** — regenera índices de skills/agentes (placeholder por ahora).

## Repair

1. Leé `workspace.json` actual.
2. Verificá que existan las carpetas declaradas en `structure[]`. Creá las faltantes
   **vacías** (sin reinstalar contenido).
3. Verificá que cada `skills[].id` tenga su `.agents/skills/<id>/SKILL.md` físico. Si
   falta alguno, copialo desde `<global>/skills/<id>/`.
4. Idem para `agents[]` y para los workflows del pack instalado.
5. Actualizá `flags.repaired_at` con timestamp ISO.
6. **Nunca** borres contenido del usuario. Si hay un archivo donde esperabas una
   carpeta o viceversa, **preguntá** antes de tocar.

## Upgrade

1. Leé `workspace.json` actual.
2. Compará `$schema_version` con la versión del sistema (actual: `2`).
3. Si es la misma → nada que hacer, informá y terminá.
4. Si es menor → migrá:
   - **v1 → v2:** agregá el campo `disciplines: []` si no existe. Ofrecé al usuario
     correr *Fase 4bis* para poblarlo (detección + selección de disciplinas).
5. Actualizá `flags.upgraded_at` con timestamp ISO.

## Install packs

1. Leé el catálogo desde `<global>/catalog-index.md`.
2. Mostrá solo los packs que **aún no** están en `workspace.json → pack` ni se
   instalaron previamente (chequeá `.agents/packs/<id>/pack.md`).
3. El usuario elige uno o más con la tool de preguntas del runtime.
4. Por cada pack elegido, ejecutá *Fase 4* (Instalación de pack) del flujo normal,
   sin sobrescribir lo existente.
5. Si el pack agrega capacidades de desarrollo y aún no hay disciplinas activas,
   ofrecé correr *Fase 4bis*.
6. Actualizá `workspace.json` agregando nuevas entradas a `skills[]`, `agents[]`,
   `disciplines[]`, y actualizando `updated_at`.

## Reindex

Placeholder. Por ahora, informá al usuario: "Reindex todavía no implementado." y
terminá sin tocar nada.

## Regla transversal

**Nunca** borres contenido del usuario. Ante cualquier conflicto (archivo que no
matchea lo esperado, divergencia entre `workspace.json` y el filesystem real),
preguntá antes de modificar.
