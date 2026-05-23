# Schema de `workspace.json` (v2)

Leé este archivo **solo cuando vayas a escribir `workspace.json`** (Fase 4, 4.5 o Fase 6
Upgrade). No es parte del contexto cargado por defecto del agente.

```json
{
  "$schema_version": 2,
  "id": "<generá un UUID v4 directamente como texto, formato xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx — sin scripts>",
  "created_at": "<ISO 8601>",
  "updated_at": "<ISO 8601>",
  "system_version": "0.1.0",
  "runtime": "claude-code | opencode | unknown",
  "pack": {
    "name": "<pack-id o 'custom' o 'custom-empty'>",
    "source": "<ruta al pack en global, o null si custom>",
    "installed_at": "<ISO 8601>"
  },
  "stack": {
    "detected": ["react", "typescript", "vite"],
    "source": "package.json | user-input | none",
    "confidence": "high | medium | low"
  },
  "skills": [
    { "id": "<skill-id>", "source": "<ruta global>", "installed_at": "<ISO>" }
  ],
  "agents": [
    { "id": "<agent-id>", "class": 1, "source": "<ruta global>", "installed_at": "<ISO>" },
    { "id": "<agent-id>", "class": 2, "source": "<ruta global>", "installed_at": "<ISO>" }
  ],
  "disciplines": ["tdd", "bdd"],
  "flags": {
    "initialized": true,
    "repaired_at": null,
    "upgraded_at": null
  },
  "structure": ["agents", "skills", "workflows"]
}
```

## Notas por campo

- **`runtime`** — entorno donde se inicializó el workspace. Detectado en Fase 1 paso 1 del
  agente vía env vars (`CLAUDECODE`, `OPENCODE`). Lo usan agentes y workflows para elegir
  mecanismos: `claude-code` habilita `AskUserQuestion`, `ScheduleWakeup`, `TaskCreate`;
  `opencode` y `unknown` caen a alternativas (`question` o chat plano).

- **`disciplines`** — array de ids de skills de disciplina activas en el proyecto. Lo
  leen los workflows de implementación (`feature-development`) para invocar
  condicionalmente cada disciplina. Vacío `[]` si el proyecto no usa ninguna. Es la
  **única fuente de verdad** de qué disciplinas están activas.

- **`structure`** — lista de las carpetas top-level que el workspace realmente tiene
  (las que se crearon por instalar contenido). Para custom-empty queda `[]`.

## Versionado

**Cambios de v1 → v2:** se agregó el campo `disciplines`. Un workspace v1 sin el campo
se trata como `disciplines: []` al hacer Upgrade.
