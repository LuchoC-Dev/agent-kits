# Handoff — estado actual y trabajo pendiente

**Estado:** Active
**Fecha:** 2026-07-30

Este documento existe para que alguien que llega —persona o agente— entienda dónde está el
proyecto sin reconstruirlo leyendo commits. Es el punto de entrada después de
[`README.md`](./README.md) de esta carpeta.

Lo que **no** hace: repetir las decisiones. Están en [`03-decisions.md`](./03-decisions.md)
y son la autoridad. Acá se las cita, no se las resume.

## 1. Qué es Agent Kits hoy

Una CLI en Go que descubre, planifica e instala recursos —skills, agentes, workflows y
kits— desde repositorios Git, de forma reproducible y auditable. Nada más. No crea
recursos, no publica, no ejecuta contenido.

Tres ideas explican casi todo el diseño:

**La identidad no es el nombre.** Un recurso se identifica con un UUID que se asigna una
vez y no cambia jamás (D-035). Su nombre es dónde se instala y cómo se lo pide, y se puede
renombrar sin romper nada (D-036). Pertenecer a un kit es una relación, no parte de la
identidad. Esto salió de un error anterior: los IDs calificados por kit (`backend/feature-
development`) ataban la identidad a algo que cambia, así que mover un recurso de kit
rompía todo lockfile que lo referenciara.

**El lockfile es el único estado.** `.agents/agent-kits.lock.json` concentra todo lo que
Agent Kits sabe de un proyecto (D-030). No hay un segundo archivo. Hubo uno —
`workspace.json`, de la skill conversacional que precedió a la CLI— y se retiró con una
migración que ya se eliminó también (D-041).

**Nada nace público.** Un recurso se crea en el catálogo privado y sólo llega al público
por una publicación explícita, que abre un pull request desde CI (D-038, D-039). La CLI no
participa: no puede escribir en un remoto, y esa garantía es verificable leyendo la lista
blanca de subcomandos de Git en `internal/git/git.go`.

## 2. Dónde está cada cosa

| Repositorio | Qué contiene | Visibilidad |
|---|---|---|
| `LuchoC-Dev/agent-kits` | la CLI: `cmd/`, `internal/`, `docs/` | pública |
| `LuchoC-Dev/repository-private` | el catálogo: 75 recursos, y los workflows de CI | privada |
| `LuchoC-Dev/repository` | el espejo publicado; hoy vacío | pública |

En el repositorio privado hay dos workflows:

- **`validate`** — construye la CLI y comprueba que el catálogo carga: identidades y
  nombres únicos, dependencias que resuelven, archivos declarados presentes. Corre en cada
  push. Reemplaza al test de inventario que antes vivía en el repo de la CLI.
- **`publish`** — toma nombres de recursos, calcula el cierre de dependencias y abre un
  pull request contra el espejo. `dry_run` viene activado por defecto.

El espejo tiene el mismo `validate`, para que una publicación malformada se detenga antes
de entrar. Su rama `main` está protegida con *include administrators*, así que ni un token
de administrador puede empujar directo.

**`meta/catalog-author.md`**, en el catálogo privado, es la guía para crear recursos
nuevos. Está al día pero **nunca se ejercitó**: nadie creó un recurso desde que se
reescribió.

## 3. Qué falta

### Lo único con una decisión detrás

**Podar el catálogo.** D-034 conservó los 75 recursos deliberadamente: podar es barato y
reversible, decidir qué se pierde no lo es, y no había información para decidirlo. Cuando
el uso real lo informe, podar es borrar directorios en el privado y publicar el
subconjunto que sobreviva.

Abre una pregunta que hoy no tiene respuesta: **qué hacer con un recurso ya publicado que
se elimina.** ¿Se retira del espejo, dejando proyectos instalados sin origen? ¿Se queda
como versión congelada? Es una decisión de producto, no técnica.

### Features sin nada que las empuje

- **Rangos de versión compuestos** (`>=1.2 <2`). Hoy se admite exacto, `^`, `~` y `*`
  (D-024). Nada en el catálogo los necesita.
- **Firma criptográfica de sources.** D-025 la dejó fuera a propósito: el modelo de
  confianza es `trust` por source más la prohibición de ejecutar contenido.
- **Adaptador de Codex.** D-021 lo condicionó a verificar su layout.

### Un desconocido conocido

**El workflow de publicación nunca corrió de verdad**, sólo en dry-run. El push de la rama
y el `gh pr create` son los dos pasos que el dry-run no ejercita. La primera publicación
real es también su primera prueba: conviene hacerla con algo chico.

## 4. Lo que no se puede romper

Cada uno tiene una decisión detrás. Romperlos no es un bug: es contradecir algo acordado,
y requiere reemplazar la decisión primero.

| Invariante | Dónde |
|---|---|
| La CLI nunca escribe en un remoto ni publica | D-003, D-004 |
| La CLI nunca ejecuta contenido del catálogo | D-025 |
| Un duplicado nunca se resuelve por precedencia | D-006 |
| Un archivo modificado localmente nunca se sobrescribe sin `--force` | D-023 |
| Dos recursos nunca escriben el mismo destino | D-028 |
| Nada nace público | D-038 |
| `go.mod` no declara dependencias externas | D-016 |

Dos matices que parecen excepciones y no lo son:

- **Sources emparentadas comparten identidades.** Cuando una source declara `publishes`,
  un recurso repetido entre ambas es el mismo recurso publicado y gana el origen privado
  (D-038). No viola D-006 porque no hay desempate: hay una relación declarada por quien
  configuró las sources.
- **El bloque `migration` del lockfile se conserva** aunque la migración ya no exista
  (D-041). Es historia de proyectos que la atravesaron, con campos que ninguna versión de
  Agent Kits entendió. Se lee y se reescribe intacto.

## 5. Trampas concretas

Cosas que cuestan tiempo si nadie las avisa.

**El árbol de trabajo está en CRLF** (`core.autocrlf=true`). Dos consecuencias: `gofmt -l`
lista casi todos los archivos y **no es una señal de nada**; y una edición por script que
asuma `\n` falla silenciosamente sin encontrar el patrón. Usá herramientas de edición que
respeten el archivo, o normalizá antes de reemplazar.

**`GOTMPDIR` debe apuntar fuera del repositorio.** Si no, `internal/git.TestHeadCommitOn
NonRepository` encuentra el `.git` del padre y da un falso negativo.

**El catálogo ya no está en este repositorio**, así que `go test ./...` no lo valida. Esa
garantía vive en la CI del repositorio privado. Si tocás el loader del catálogo, corré
también ese workflow.

**El tag `migration-window`** marca la última build capaz de migrar un `workspace.json`
heredado. No lo borres: es la única ruta para un proyecto que se quedó atrás.

**`catalog-index.md`** es un índice para humanos, mantenido a mano, y **nada lo valida**.
Va a divergir del catálogo real. El inventario verdadero lo da `agent-kits search`.

## 6. Cómo verificar que todo sigue bien

```powershell
go build ./...
go test ./...
git diff --check
```

Y para el catálogo, desde el repositorio privado:

```powershell
gh workflow run validate --repo LuchoC-Dev/repository-private
```

Una prueba de extremo a extremo, contra el catálogo real:

```powershell
agent-kits source add private https://github.com/LuchoC-Dev/repository-private.git --access private
agent-kits source sync private
agent-kits install tdd --project <un-directorio-vacio> --yes
agent-kits doctor --project <ese-directorio>
```

## 7. Decisiones que podés cuestionar

Lo que sigue fueron juicios, no requisitos. Si te parecen mal, están para discutirse.

- **Cuatro recursos se renombraron** porque su nombre de instalación chocaba:
  `design-backend` y `design-fullstack` (chocaban con el kit que los contiene), y
  `backend-feature-development` y `frontend-feature-development` (chocaban entre sí).
  Bajo el modelo actual renombrar es barato: la identidad no cambia.
- **Todos los recursos sin versión declarada recibieron `1.0.0`.** La alternativa era
  `0.1.0` o conservar el `0.0.0` sintético que impedía a `update` distinguir cambios.
- **El catálogo arrancó con una historia limpia** en su repositorio nuevo. La evolución
  previa quedó en el historial de la CLI.
- **Los 75 recursos estuvieron públicos** y siguen en el historial de `agent-kits`. Se
  aceptó explícitamente (D-040) porque la mayor parte no va a sobrevivir a la poda.
- **La publicación es transitiva:** publicar algo publica sus dependencias privadas. Es
  necesario —un recurso público con una dependencia privada sería ininstalable— pero
  significa que publicar puede exponer más de lo pedido. Por eso el PR separa lo pedido de
  lo arrastrado; leer esa sección es la mitigación.
