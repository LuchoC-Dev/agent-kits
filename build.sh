#!/usr/bin/env bash
# build.sh — empaqueta kits-init para distribución
#
# Genera dist/kits-init/ con solo los archivos que el usuario final necesita.
# Excluye: tests/, meta/, PROJECT-CONTEXT.md, .git/, .gitignore, build.sh, dist/.
#
# Uso:
#   ./build.sh              → genera dist/kits-init/
#   ./build.sh --zip        → además genera dist/kits-init.zip
#   ./build.sh --tar        → además genera dist/kits-init.tar.gz
#   ./build.sh --clean      → borra dist/ y sale

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DIST="$ROOT/dist"
OUT="$DIST/kits-init"

# --clean
if [[ "${1:-}" == "--clean" ]]; then
  rm -rf "$DIST"
  echo "✓ dist/ eliminado"
  exit 0
fi

echo "→ Building kits-init distribution…"

# Limpiar build previo
rm -rf "$OUT"
mkdir -p "$OUT"

# Archivos raíz a distribuir
ROOT_FILES=(
  "SKILL.md"
  "agent.md"
  "catalog-index.md"
)

# Carpetas a distribuir
ROOT_DIRS=(
  "skills"
  "packs"
  "agents"
)

for f in "${ROOT_FILES[@]}"; do
  if [[ -f "$ROOT/$f" ]]; then
    cp "$ROOT/$f" "$OUT/$f"
  else
    echo "✗ Falta archivo requerido: $f" >&2
    exit 1
  fi
done

for d in "${ROOT_DIRS[@]}"; do
  if [[ -d "$ROOT/$d" ]]; then
    cp -r "$ROOT/$d" "$OUT/$d"
  else
    echo "✗ Falta carpeta requerida: $d" >&2
    exit 1
  fi
done

# Stats
SKILLS_COUNT=$(ls -1 "$OUT/skills" 2>/dev/null | wc -l)
PACKS_COUNT=$(ls -1 "$OUT/packs" 2>/dev/null | wc -l)
AGENTS_COUNT=$(ls -1 "$OUT/agents" 2>/dev/null | wc -l)
TOTAL_FILES=$(find "$OUT" -type f | wc -l)
SIZE=$(du -sh "$OUT" | cut -f1)

echo "✓ dist/kits-init/ generado"
echo "  ├ skills: $SKILLS_COUNT"
echo "  ├ packs: $PACKS_COUNT"
echo "  ├ agents (pool global): $AGENTS_COUNT"
echo "  ├ archivos totales: $TOTAL_FILES"
echo "  └ tamaño: $SIZE"

# Empaquetado opcional
case "${1:-}" in
  --zip)
    if command -v zip >/dev/null 2>&1; then
      (cd "$DIST" && zip -rq kits-init.zip kits-init)
      echo "✓ dist/kits-init.zip"
    else
      echo "✗ zip no instalado, salteado" >&2
    fi
    ;;
  --tar)
    (cd "$DIST" && tar -czf kits-init.tar.gz kits-init)
    echo "✓ dist/kits-init.tar.gz"
    ;;
esac

echo ""
echo "Instalación: copiar dist/kits-init/ a ~/.claude/skills/ (Claude Code)"
echo "             o ~/.config/opencode/skills/ (OpenCode)"
