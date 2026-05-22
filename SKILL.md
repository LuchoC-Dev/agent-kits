---
name: app-init
description: Inicializa un workspace AI-first en el proyecto actual creando la estructura `.agents/` con memoria, contexto, agentes, skills y disciplinas de desarrollo. Trigger cuando el usuario diga "/app-init", "inicializar workspace AI", "bootstrap AI workspace", "armar workspace", "init ai system" o cuando entre a un proyecto vacío y quiera arrancar el sistema de skills.
---

# app-init — Lanzador del agente de inicialización

## Qué es esta skill

Es un **lanzador delgado** (thin launcher). No contiene el procedimiento de
inicialización: su única responsabilidad es **lanzar el agente `app-init`** en su propio
contexto aislado. Todo el flujo real —detección de stack, elección de packs, disciplinas,
generación de `.agents/`— vive en el agente, no acá.

## Cuándo actuar

Cuando el usuario invoque `/app-init` o pida explícitamente inicializar un workspace
AI-first en el proyecto actual. **No actúes proactivamente** en ningún otro contexto.

## Qué hacer

1. **Resolvé `<global>`** — es el directorio base de esta skill. Aparece al comienzo del
   contexto como `Base directory for this skill: <ruta>`. Ya lo tenés; no ejecutes nada
   para obtenerlo. **`<global>` = `<base_dir>`.** El catálogo (`packs/`, `skills/`,
   `agents/`, `catalog-index.md`) vive ahí mismo.

2. Leé el archivo `agent.md` en la ruta exacta `<base_dir>/agent.md`.

3. **Seguí las instrucciones de `agent.md` como el agente `app-init`, en esta misma
   sesión.** Conducís al usuario con preguntas y creás `.agents/` en el cwd.
   Contexto disponible:
   - El **cwd** — directorio del proyecto a inicializar.
   - `<global>` = `<base_dir>` resuelto en el paso 1.

## No hacer

- No ejecutes el flujo de inicialización vos misma — siempre delegá en el agente.
- No actúes proactivamente fuera del trigger `/app-init`.
