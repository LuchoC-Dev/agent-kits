---
name: app-init
description: Inicializa un workspace AI-first en el proyecto actual creando la estructura `.my-system/` con memoria, contexto, agentes, skills y disciplinas de desarrollo. Trigger cuando el usuario diga "/app-init", "inicializar workspace AI", "bootstrap AI workspace", "armar workspace", "init ai system" o cuando entre a un proyecto vacío y quiera arrancar el sistema de skills.
---

# app-init — Lanzador del agente de inicialización

## Qué es esta skill

Es un **lanzador delgado** (thin launcher). No contiene el procedimiento de
inicialización: su única responsabilidad es **lanzar el agente `app-init`** en su propio
contexto aislado. Todo el flujo real —detección de stack, elección de packs, disciplinas,
generación de `.my-system/`— vive en el agente, no acá.

## Cuándo actuar

Cuando el usuario invoque `/app-init` o pida explícitamente inicializar un workspace
AI-first en el proyecto actual. **No actúes proactivamente** en ningún otro contexto.

## Qué hacer

1. **PRIMER PASO OBLIGATORIO — resolvé `<global>` ahora, antes de cualquier otra cosa.**

   `<base_dir>` es el directorio base de esta skill — aparece al comienzo del contexto
   como `Base directory for this skill: <ruta>`. Ya lo tenés; no ejecutes nada para
   obtenerlo.

   Con `<base_dir>` en mano, resolvé `<global>` en este orden:

   a. Ejecutá con Bash: `echo "$MY_SYSTEM_HOME"`
      - Si retorna una ruta no vacía → `<global>` = ese valor.
   b. Si retornó vacío → verificá si `<base_dir>/packs/` existe con Bash:
      `ls "<base_dir>/packs"` (usá comillas — la ruta puede tener espacios).
      - Si existe → `<global>` = `<base_dir>`.
   c. Si ninguno funcionó → `<global>` = `~/.my-system/`.

   **No uses `$env:MY_SYSTEM_HOME` ni PowerShell — solo Bash.**

2. Leé el archivo `agent.md` en la ruta exacta `<base_dir>/agent.md`.

3. **Seguí las instrucciones de `agent.md` como el agente `app-init`, en esta misma
   sesión.** Conducís al usuario con preguntas y creás `.my-system/` en el cwd.
   Contexto disponible:
   - El **cwd** — directorio del proyecto a inicializar.
   - `<global>` — ya resuelta en el paso 1. **No la vuelvas a resolver.** Usá ese valor
     cada vez que `agent.md` diga `<global>`. El agente NO debe ejecutar
     `echo "$MY_SYSTEM_HOME"` ni ningún comando de env var de nuevo.

## No hacer

- No ejecutes el flujo de inicialización vos misma — siempre delegá en el agente.
- No actúes proactivamente fuera del trigger `/app-init`.
