# Contexto de producto — Agent Kits Next

**Estado:** Draft
**Fecha:** 2026-07-29

## 1. Problema

Las capacidades utilizadas por agentes —skills, agentes especializados, workflows y
kits— suelen quedar distribuidas entre carpetas personales, proyectos y herramientas.
Esto provoca:

- dificultad para descubrir qué existe;
- copias manuales;
- versiones desconocidas;
- nombres inconsistentes;
- dependencias implícitas;
- riesgo de sobrescribir archivos;
- mezcla accidental entre contenido público y privado;
- instrucciones diferentes según el runtime.

El usuario necesita poder pedirle a un agente:

> Instala `frontend-design` en este proyecto desde mis fuentes configuradas.

El agente debe traducir esa intención a una operación determinística, verificable e
idempotente.

## 2. Objetivo

Construir una nueva evolución de Agent Kits que funcione como **gestor de obtención e
instalación de capacidades para agentes**.

El repositorio y sus índices serán fuentes de verdad versionadas. La CLI resolverá
recursos y dependencias, planificará cambios, instalará en el proyecto activo y
registrará el resultado.

## 3. Usuarios

### Usuario propietario

Mantiene fuentes públicas y privadas, decide qué capacidades confía e instala recursos
en sus proyectos mediante lenguaje natural o comandos.

### Agente consumidor

Consulta el catálogo, resuelve un pedido del usuario, muestra un plan y ejecuta la
instalación sin necesitar conocer la estructura física de los repositorios.

### Colaborador

Trabaja sobre el nuevo fork y desarrolla la CLI, los contratos y adaptadores. No obtiene
por ello permisos automáticos para modificar el catálogo original.

### Autor de recursos

Crea y valida recursos mediante una superficie separada. La herramienta de autoría no
forma parte del alcance de la CLI consumidora.

## 4. Propuesta de valor

- Una orden estable para instalar cualquier recurso soportado.
- Catálogos Git versionados y auditables.
- Uso transparente de fuentes públicas y privadas autorizadas.
- IDs únicos sin precedencia ni sobrescritura silenciosa.
- Instalaciones reproducibles mediante lockfile.
- Separación clara entre contenido, autoría, distribución y runtime.
- Compatibilidad extensible con Codex, Claude Code y otros entornos.

## 5. Casos de uso principales

### Descubrir

```text
agent-kits search frontend
agent-kits info frontend-design
```

### Planificar

```text
agent-kits plan frontend-design --project .
```

### Instalar

```text
agent-kits install frontend-design --project . --runtime auto --yes
```

### Actualizar

```text
agent-kits update frontend-design --project .
```

### Diagnosticar

```text
agent-kits doctor --project .
```

Los nombres y flags son candidatos; deberán validarse antes de convertirse en API
pública.

## 6. Tipos de recurso iniciales

| Tipo | Función |
|---|---|
| Skill | Conocimiento, procedimiento o capacidad reutilizable. |
| Agent | Actor definido por un rol y un contrato de entrada/salida. |
| Workflow | Proceso que ordena actividades, gates y actores. |
| Kit | Composición instalable de skills, agents y workflows. |

Tools y templates aparecen en la taxonomía conceptual, pero su soporte en el MVP sigue
abierto.

## 7. Visibilidad

La visibilidad se implementa mediante permisos reales y repositorios separados:

- fuente pública: visible sin autenticación;
- fuente privada: visible solo con credenciales autorizadas.

`visibility` puede existir como metadato, pero nunca será la frontera de seguridad.

Una sola instalación de Agent Kits podrá consultar varias fuentes. Las credenciales
permanecerán en Git, SSH o el proveedor correspondiente, nunca en los manifiestos.

## 8. Identidad

Cada recurso tendrá un ID canónico único entre todas las fuentes administradas.

No se admitirá:

```text
public:frontend-design
personal:frontend-design
```

La fuente describe ubicación, no identidad. Si un ID aparece en más de una fuente
activa, el sistema falla de forma cerrada y no instala nada.

Mover un recurso de privado a público será una migración de visibilidad del mismo
recurso, no la creación de una variante paralela.

## 9. Separación entre creación y obtención

Agent Kits:

- busca;
- inspecciona;
- planifica;
- instala;
- actualiza;
- elimina;
- diagnostica.

Agent Kits no:

- crea recursos;
- hace commits;
- abre PR;
- publica;
- escribe en los repositorios remotos.

La autoría pertenecerá a una herramienta o proceso separado. El nombre provisional
`Kit Forge` no está aprobado como marca definitiva.

## 10. Kits de orquestación

Una orquestación liviana basada en archivos es un kit, no un runtime independiente.

Ejemplo:

```text
project-orchestration-kit
├── project-orchestrator
├── implementation-agent
├── verification-agent
├── memory templates
└── delegation guide
```

El agente anfitrión continúa siendo el runtime. El kit solo instala roles, memoria,
prompts y convenciones.

## 11. No objetivos

- Reemplazar Git como sistema de versionado.
- Construir un Event Agent Manager nuevo.
- Ejecutar una plataforma multiagente persistente.
- Publicar recursos desde la CLI consumidora.
- Crear un marketplace comercial en el MVP.
- Guardar credenciales.
- Resolver automáticamente contenido no confiable sin revisión.
- Soportar todos los runtimes desde la primera entrega.

## 12. Relación con fast-weekend-core

La nueva versión vive dentro del área de trabajo de `fast-weekend-core`, pero debe
mantener límites claros. Agent Kits será una dependencia consumible mediante contratos
estables; no se mezclarán internamente ambos productos sin una decisión explícita.

## 13. Éxito del producto

La primera versión será útil cuando:

1. un agente pueda descubrir e instalar un recurso por ID;
2. la misma operación repetida no duplique ni corrompa archivos;
3. el sistema registre origen, versión y checksum;
4. un ID duplicado detenga la operación;
5. una fuente privada sea invisible sin autorización;
6. ningún comando pueda publicar o escribir en fuentes remotas;
7. los cambios puedan previsualizarse antes de escribir.

## 14. Glosario

| Término | Definición |
|---|---|
| Source | Repositorio o índice desde el que se obtienen recursos. |
| Catalog | Vista consultable de recursos disponibles en una source. |
| Canonical ID | Identidad estable y global de un recurso. |
| Manifest | Metadatos y contratos declarativos de un recurso. |
| Lockfile | Registro reproducible de lo instalado, su origen y versión. |
| Runtime adapter | Traducción entre el recurso canónico y un entorno concreto. |
| Kit | Composición distribuible de recursos. |
| Legacy | Implementación heredada del bootstrap `kits-init`. |
