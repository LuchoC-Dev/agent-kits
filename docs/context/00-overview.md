# Agent Kits — de qué se trata todo esto

**Estado:** Active
**Fecha:** 2026-07-30

Si llegás nuevo, este es el único documento que tenés que leer entero. Cuenta qué problema
resuelve el proyecto, cómo funciona y por qué está hecho así. Todo lo demás es referencia
que vas a consultar cuando la necesites.

Toma unos quince minutos. Al final sabés dónde seguir según lo que vayas a hacer.

---

## 1. El problema

Las capacidades que usa un agente —una guía para diseñar interfaces, un rol de revisor, un
proceso de desarrollo de features— terminan desparramadas: copiadas a mano entre proyectos,
sin versión, con nombres que cambian según quién las copió, y sin forma de saber qué
existe. Cuando además hay contenido que no puede ser público, el desorden deja de ser
incómodo y pasa a ser un riesgo.

Agent Kits resuelve eso: **descubrir, planificar e instalar** esas capacidades desde
repositorios Git, de forma reproducible y auditable.

## 2. Qué es, y qué no

Es una **CLI en Go**, sin dependencias externas, que se instala como un binario. Un agente
la usa igual que una persona: todo comando acepta `--json` y devuelve un envelope estable
con códigos de error documentados.

Lo que hace: buscar recursos, mostrarte un plan antes de tocar nada, instalarlos en tu
proyecto, actualizarlos, quitarlos y diagnosticar el resultado.

Lo que **deliberadamente no hace**, y no por falta de tiempo:

- **No crea recursos.** Obtener e instalar es un problema; autorear es otro. Mezclarlos
  hace mal las dos cosas (D-003).
- **No publica ni escribe en ningún remoto.** `git` se invoca con una lista blanca de
  subcomandos de sólo lectura. La garantía se verifica leyendo `internal/git/git.go`, no
  confiando en una promesa (D-004).
- **No ejecuta contenido del catálogo.** Nunca, bajo ninguna circunstancia (D-025).
- **No sobrescribe en silencio.** Si tocaste un archivo que la CLI instaló, se detiene
  (D-023).

## 3. Vocabulario

| Término | Qué es |
|---|---|
| **Recurso** | La unidad instalable. Es de uno de cuatro tipos. |
| **Skill** | Un conocimiento o procedimiento reutilizable. Vive en `skills/<name>/`. |
| **Agent** | Un actor definido por un rol y un contrato de entrada/salida. |
| **Workflow** | Un proceso que ordena actividades y actores. |
| **Kit** | Una composición instalable de skills, agentes y workflows. |
| **Source** | Un repositorio Git que ofrece recursos. Puede ser público o privado. |
| **Catálogo** | La vista agregada de todas las sources configuradas. |
| **Manifest** | El `agent-kit.json` que describe un recurso. |
| **Lockfile** | `.agents/agent-kits.lock.json`: qué tiene instalado un proyecto. |
| **Runtime** | Dónde aterrizan los recursos: Claude Code, OpenCode o el layout genérico. |
| **Adapter** | Lo que traduce un recurso a la ruta concreta de un runtime. |

Dos conceptos que cuestan más y explican medio diseño:

**Identidad ≠ nombre.** Un recurso tiene un `id` —un UUID— que se asigna una vez y no
cambia nunca, y un `name` que es dónde se instala y cómo lo pedís. El nombre se puede
cambiar; la identidad no. Por eso renombrar un recurso, o moverlo de kit, no rompe ningún
proyecto que lo tenga instalado.

**Privado ⊇ público.** Hay dos repositorios de catálogo. Todo recurso nace en el privado; el
público es el subconjunto que alguien decidió publicar explícitamente. Nunca al revés.

## 4. Cómo funciona un install, de punta a punta

```powershell
agent-kits install frontend-design --project . --yes
```

1. **Se resuelve la referencia.** `frontend-design` es un nombre. Si una sola source lo
   ofrece, listo. Si varias, el comando falla y te lista los candidatos como
   `acme:frontend-design` — nunca elige por su cuenta.
2. **Se expande el cierre de dependencias.** Si el recurso depende de otros, entran todos.
   Las dependencias apuntan a identidades, no a nombres, así que un rename río arriba no
   rompe nada.
3. **Se planifica.** Se calcula, archivo por archivo, qué pasaría: crear, actualizar,
   adoptar, no tocar. Se comparan tres checksums —el del lockfile, el del disco y el del
   contenido nuevo— y si el archivo cambió localmente, el plan **se bloquea**. El plan no
   escribe nada.
4. **Se escanean secretos** en el contenido a instalar. Un match de alta confianza bloquea.
5. **Se aplica, con journal.** Cada archivo que se sobrescribe o borra se copia antes a un
   directorio temporal fuera del proyecto. Si algo falla a mitad de camino, se restaura
   todo. No existe una instalación a medio aplicar.
6. **Se escribe el lockfile**, que registra qué recurso, de qué source, en qué versión, con
   qué commit y con qué checksum por archivo.

Volver a correr el mismo comando produce un plan vacío y no reescribe nada.

## 5. La arquitectura

El código está en `internal/`, dividido por responsabilidad. Ninguno de estos paquetes
conoce a la CLI; la CLI los orquesta.

| Paquete | Responsabilidad |
|---|---|
| `model` | El vocabulario canónico: recurso, manifest, plan, lockfile. No sabe de disco ni de runtimes. |
| `source` | Las sources configuradas y su cache local. |
| `catalog` | Convierte sources en una vista consultable, y hace cumplir la unicidad. |
| `resolve` | Expande referencias en un conjunto cerrado de dependencias. |
| `plan` | Convierte una resolución en un plan determinístico. No escribe. |
| `install` | Aplica un plan aprobado. Y `doctor`. |
| `journal` | Backup y rollback de una operación. |
| `adapter` | Dónde aterriza cada recurso según el runtime. |
| `workspace` | Leer y escribir el lockfile. |
| `security` | Rutas contenidas, límites de tamaño, detección de secretos. |
| `git` | La lista blanca de subcomandos. Es la garantía de "sin escrituras remotas". |
| `errs` | El vocabulario de errores y su mapeo a exit codes. |
| `semver`, `fsutil` | Versiones y utilidades de filesystem. |

La forma general: **el núcleo es canónico y los bordes traducen**. Un recurso significa lo
mismo venga de donde venga; sólo el adapter sabe de rutas concretas, y sólo `source` sabe
de Git.

## 6. Las decisiones que explican el diseño

Hay 41 decisiones registradas en [`03-decisions.md`](./03-decisions.md). Estas cinco son
las que hacen falta para entender por qué el sistema es así.

**Planificar antes de escribir (D-010).** Toda operación que modifica un proyecto muestra
primero qué haría. En una sesión no interactiva —un agente, un pipe— aprobar sólo puede
venir de `--yes`. Nunca se escribe por suposición.

**Fail closed ante conflictos (D-023).** Si un archivo que la CLI instaló fue modificado,
no se sobrescribe: la operación se detiene. `--force` es la única forma de pasar por
encima, y nunca es implícita.

**Sin precedencia (D-006).** Si dos recursos reclaman la misma identidad, el catálogo
entero se invalida. No se elige uno por orden de source ni por ninguna otra regla
implícita. Un duplicado es un defecto que hay que corregir, no una ambigüedad que la
herramienta deba resolver sola.

**La identidad es un UUID (D-035).** Vino de corregir un error: antes el ID incluía el kit
(`backend/feature-development`), lo que ataba la identidad a una relación que cambia. Mover
un recurso de kit le cambiaba el ID y rompía todo lockfile que lo referenciara.

**Nada nace público (D-038).** El catálogo privado es el origen; el público es un espejo de
lo publicado. Publicar es un workflow de CI que abre un pull request, nunca la CLI. Si un
token se filtra, lo peor que permite es publicar en un repositorio que ya era público.

## 7. Cómo llegamos acá

Agent Kits empezó siendo otra cosa: una **skill conversacional** llamada `kits-init` que un
agente invocaba con `/kits-init`, hacía preguntas y copiaba archivos a `.agents/`. Funcionó,
y su catálogo es el que sigue vivo hoy.

Tenía dos límites que no se arreglaban con más features: no había forma de saber qué
versión tenías instalada ni de actualizarla, y todo dependía de que un agente siguiera
instrucciones en Markdown correctamente.

La CLI se construyó al lado, primero como complemento y después como reemplazo. La
transición se hizo por partes, cada una con su decisión: el lockfile absorbió el estado, la
skill se deprecó y se retiró, el catálogo pasó de Markdown con frontmatter a manifests
JSON, la identidad se separó del nombre, y el catálogo se mudó a repositorios propios con
publicación por CI.

De todo eso queda: el catálogo, el layout `.agents/` y la taxonomía de recursos. Lo demás
se reemplazó. `PROJECT-CONTEXT.md`, en la raíz, conserva la historia del sistema original —
es útil para entender **por qué** el catálogo es como es, y ya no describe cómo funciona
nada.

## 8. Dónde seguir

Según lo que vayas a hacer:

| Si vas a… | Leé |
|---|---|
| **cambiar algo** | [`08-handoff.md`](./08-handoff.md) — estado, pendientes, invariantes y trampas |
| **usar la CLI** | [`../cli.md`](../cli.md) — comandos, contratos JSON, errores |
| **entender una decisión** | [`03-decisions.md`](./03-decisions.md) — es la autoridad |
| **crear un recurso** | `meta/catalog-author.md`, en el catálogo privado |
| **trabajar como agente acá** | [`../../AGENTS.md`](../../AGENTS.md) — reglas y límites |

Y estos son **archivo histórico**: explican cómo se llegó a algo, no cómo funciona hoy.
No los leas para orientarte.

- `01-product-context.md` — la intención original, escrita antes de implementar
- `05-roadmap.md` — las fases del MVP, todas cerradas
- `06-legacy-baseline.md` — la auditoría del sistema anterior
- `07-cli-only-transition-plan.md` — el plan de la transición, completado

---

**Una advertencia sobre esta carpeta.** `docs/context/` describe intención y decisiones;
`docs/cli.md` describe comportamiento real. Cuando difieran, manda `docs/cli.md`: la
intención puede quedar vieja, el comportamiento es lo que la CLI hace.
