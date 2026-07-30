# Registro de decisiones — Agent Kits Next

**Estado:** Active
**Fecha inicial:** 2026-07-29

Cada decisión tiene un identificador estable. Las decisiones reemplazadas se conservan
marcadas como `Superseded`.

## D-001 — Evolucionar mediante un fork

**Estado:** Accepted

La nueva versión se implementará en un fork independiente. El repositorio original se
mantiene como referencia y upstream.

**Razón:** permitir cambios profundos, colaboración y experimentación sin alterar la
aplicación existente.

## D-002 — El repositorio es la fuente de verdad

**Estado:** Accepted

Skills, agents, workflows y kits viven en repositorios Git versionados. Los runtimes son
destinos de instalación, no el almacenamiento canónico.

## D-003 — Separar creación y obtención

**Estado:** Accepted

Agent Kits solo obtiene e instala. La creación y validación autoral pertenecen a otra
superficie.

**Consecuencia:** Agent Kits no incluirá `create`, `import` o `publish` como operaciones
de escritura sobre el catálogo.

## D-004 — Prohibir publicación desde Agent Kits

**Estado:** Accepted

La CLI no hace commits, push, ramas ni PR en las sources. La incorporación al catálogo
usa procesos Git separados y controlados.

## D-005 — Separar físicamente lo público y lo privado

**Estado:** Accepted

Los recursos públicos y privados viven en repositorios con permisos distintos.

**Razón:** un campo `visibility` dentro de un repositorio público no proporciona
privacidad.

## D-006 — IDs globalmente únicos

**Estado:** Accepted

Un ID solo puede corresponder a una capacidad activa entre todas las fuentes
administradas.

**Consecuencia:** el origen no forma parte del ID. Los duplicados son errores de
integridad, nunca candidatos a precedencia.

## D-007 — Una visibilidad activa por recurso

**Estado:** Accepted

Un recurso privado que pasa a público se traslada conservando su identidad. No se crean
variantes simultáneas con el mismo ID.

## D-008 — Convención semántica de nombres

**Estado:** Accepted

- skill → disciplina u objeto (`frontend-design`);
- agent → persona o rol (`frontend-designer`);
- workflow → proceso (`design-frontend-interface`);
- kit → colección (`frontend-design-kit`);
- tool → instrumento (`figma-inspector`);
- template → artefacto (`design-brief-template`).

La convención se documenta y puede generar advertencias, pero no es una restricción
estructural rígida.

## D-009 — La CLI está diseñada para agentes

**Estado:** Accepted

Las operaciones deben ser no interactivas cuando se proporcionan flags suficientes,
idempotentes y capaces de devolver JSON estructurado.

## D-010 — Planificar antes de escribir

**Estado:** Accepted

Toda instalación o actualización debe ofrecer un plan o dry-run. Los cambios
significativos requieren confirmación explícita o `--yes`.

## D-011 — No sobrescribir silenciosamente

**Estado:** Accepted

Si un archivo administrado cambió localmente, Agent Kits no lo reemplaza por
precedencia. La política de resolución final sigue abierta.

## D-012 — Orquestación liviana como kit

**Estado:** Accepted

Un sistema compuesto por un agente orquestador, agentes delegados, memoria y guías es un
kit instalable. El agente anfitrión es el runtime.

**No confundir con:** un servicio persistente de orquestación como Event Agent Manager.

## D-013 — Mantener la autoría fuera del MVP

**Estado:** Accepted

La herramienta de autoría puede diseñarse posteriormente. Su nombre provisional no
forma parte de la API ni de la marca de Agent Kits.

## D-014 — Mantener upstream sin push

**Estado:** Superseded por D-040

El remoto `upstream` del fork local tiene `pushurl=DISABLED`. Un `origin` colaborativo se
añadirá solo después de acordar propietario y nombre.

## D-040 — `LuchoC-Dev/agent-kits` pasa a ser el repositorio de la CLI

**Estado:** Accepted
**Fecha:** 2026-07-30
**Reemplaza:** D-014
**Completa:** D-037

El trabajo deja de vivir sólo en local: `LuchoC-Dev/agent-kits` recibe la CLI y se
convierte en el repositorio de la herramienta. El remoto se llama `origin` y admite push.

**Razón:** el fork sin push existía para evolucionar mientras no estaba claro en qué
terminaría el rediseño (D-001). Eso se cerró: la CLI es el producto, y el nombre del
repositorio es el nombre del producto. Mantenerlo congelado obligaría a publicar la
herramienta con otro nombre y a que su propio repositorio siguiera ofreciendo la skill que
D-029 deprecó.

**Consecuencias:**

- el catálogo desaparece de la rama por defecto de ese repositorio, porque se mudó a
  `repository-private` (D-037);
- `SKILL.md`, `repair-upgrade.md` y `workspace-schema.md` desaparecen con él, como ya
  había decidido D-029;
- ambos siguen en el historial de Git, que es donde corresponde que esté la historia;
- la CI del catálogo puede por fin construir la CLI, que es lo que la mantenía en rojo.

**No cambia** la prohibición que importa: la CLI sigue sin poder escribir en un remoto
(D-004). Lo que se publica acá es el código de la herramienta, con Git, por una persona.

**Exposición previa, aceptada el 2026-07-30:** los 75 recursos estuvieron públicos en este
repositorio y siguen en su historial. Sacarlos de la rama por defecto no los saca del
historial, y no se hará nada al respecto.

**Razón:** el valor de lo expuesto es bajo y transitorio. La mayor parte del catálogo no
va a sobrevivir a la poda que D-034 dejó pendiente, así que reescribir el historial
—con el costo y el riesgo que eso tiene— protegería contenido que de todos modos va a
desaparecer. El invariante "nada nace público" (D-038) rige hacia adelante, que es donde
está el contenido que sí importa proteger.

Esto **no** es un permiso general: un recurso que hoy sea sensible sigue naciendo privado
y sale sólo por publicación explícita. Lo que se acepta es una exposición concreta y
acotada, no una excepción a la regla.

## D-015 — Especificar antes de implementar

**Estado:** Accepted

La nueva versión no comenzará con código. Primero se audita el legado, se revisa la
especificación y se aprueban contratos y límites del MVP.

## D-016 — Stack: Go sin dependencias externas

**Estado:** Accepted
**Fecha:** 2026-07-29

La CLI se implementa en Go, usando exclusivamente la librería estándar.

**Razón:** `go build` produce binarios estáticos para Windows, macOS y Linux sin runtime
instalado; la stdlib cubre JSON, SHA-256, filesystem y ejecución de procesos; `go test`
no requiere configuración. Rust era la alternativa considerada, pero habría exigido
crates externos (`serde`, `clap`, `anyhow`) para llegar al mismo punto.

**Consecuencia:** `go.mod` no declara `require`. Añadir la primera dependencia externa
requiere una decisión nueva.

## D-017 — Manifests en JSON

**Estado:** Accepted

Cada recurso canónico se describe con un `agent-kit.json` que incluye `schema_version`.

**Razón:** JSON se parsea con la stdlib, es inequívoco y es el formato que un agente
consume sin ambigüedad. No se admite YAML en los contratos nuevos.

## D-018 — Lockfile en JSON dentro del workspace

**Estado:** Accepted

El lockfile es `.agents/agent-kits.lock.json`, con `schema_version`, y registra por
recurso: `id`, `type`, `source`, `version`, `commit`, `checksum`, `requested` y la lista
de archivos instalados con su checksum individual.

**Razón:** vive junto a lo que describe, se versiona con el proyecto y es legible por el
mismo parser que los manifests.

## D-019 — Gramática de IDs canónicos con propiedad de kit

**Estado:** Superseded por D-035 y D-036

> La propiedad de kit resolvía las colisiones de nombre, pero ataba la identidad a una
> relación que cambia. D-035 la reemplaza por un UUID estable y D-036 convierte el nombre
> en un atributo renombrable.

Un ID canónico tiene dos formas:

- `<name>` para recursos del pool global;
- `<kit>/<name>` para recursos cuya identidad pertenece a un kit.

La unicidad se exige sobre el ID canónico completo. Una referencia corta que coincida con
más de un ID canónico devuelve `ambiguous_id` y detiene la operación.

**Razón:** el catálogo heredado contiene `feature-development` con contenido distinto en
los packs `backend` y `frontend`, y workflows que colisionan con nombres de pack
(`backend-design`, `fullstack-design`). La propiedad de kit resuelve esas colisiones sin
renombrar contenido heredado ni elegir un ganador por precedencia.

**No viola D-006:** el prefijo es el kit propietario, no la source. Mover un kit entre
una source privada y una pública conserva los IDs de sus componentes.

## D-020 — El MVP soporta los cuatro tipos desde el arranque

**Estado:** Accepted

`skill`, `agent`, `workflow` y `kit` son ciudadanos de primera clase en resolución, plan,
instalación, lockfile y doctor. `tool` y `template` siguen fuera del alcance.

## D-021 — Adaptadores iniciales

**Estado:** Accepted

Se implementan tres adaptadores: `agents` (genérico), `claude-code` y `opencode`, con
detección automática por variables de entorno. Comparten el layout de destino `.agents/`
y difieren en el runtime declarado en `workspace.json`.

**Razón:** son los runtimes que el sistema heredado ya soporta. Codex se añade cuando su
layout esté verificado.

## D-022 — Compatibilidad con `kits-init` es obligatoria

**Estado:** Superseded por D-029, D-030 y D-031
**Reemplaza la incógnita de:** `04-specification.md §12.9`

La CLI lee y escribe `workspace.json` v2 preservando los campos que no administra, y
ofrece `agent-kits import` para adoptar un workspace creado por `kits-init` generando su
lockfile a partir de los archivos presentes.

**Consecuencia:** un workspace puede alternar entre el flujo conversacional heredado y la
CLI sin corromperse. El catálogo heredado no se migra ni se mueve.

## D-023 — Conflictos: fail closed con anulación explícita

**Estado:** Accepted
**Refina:** D-011

Se comparan tres checksums: el registrado en el lockfile, el del archivo en disco y el
del contenido nuevo.

| Lockfile | Disco | Nuevo | Resultado |
|---|---|---|---|
| = | = | = | `unchanged` |
| = disco | ≠ nuevo | — | `update` |
| ≠ disco | — | — | `divergent` → bloquea |
| sin registro | existe | — | `adopt` requiere confirmación |

Un `divergent` devuelve `local_divergence` y no escribe nada. `--force` es la única forma
de sobrescribir y nunca es implícita.

## D-024 — Semver con constraints acotados

**Estado:** Accepted

Las versiones son SemVer 2.0.0 sin metadatos de build. Los constraints admitidos son:
exacto (`1.2.0`), caret (`^1.2.0`), tilde (`~1.2.0`) y cualquiera (`*` o ausente).

**Razón:** cubre los casos reales del catálogo sin implementar un solver completo. Rangos
compuestos requieren una decisión nueva.

## D-025 — Confianza por source y prohibición de ejecución

**Estado:** Accepted

Cada source declara `trust`: `trusted` o `review`. La CLI **nunca** ejecuta contenido del
catálogo, en ninguna circunstancia. Antes de escribir valida rutas, rechaza symlinks,
aplica un límite de tamaño por archivo y detecta secretos.

**Razón:** cierra la pregunta abierta sobre el modelo de confianza sin construir firmas
criptográficas en el MVP. Un repositorio público no implica confianza automática.

## D-026 — El catálogo heredado se consume mediante adaptador

**Estado:** Accepted — transición; su retirada está aprobada por D-032

El loader de catálogos detecta el layout heredado (`catalog-index.md` + `skills/` +
`packs/`) y sintetiza manifests canónicos en memoria. No se reescribe ningún archivo del
catálogo actual.

**Razón:** permite que las 50 skills, 7 packs y 4 agentes existentes sean instalables por
la CLI nueva sin una migración destructiva, y mantiene el legado como fuente autoritativa
de su propio contenido.

## D-027 — Las referencias mutuas se informan, no se rechazan

**Estado:** Accepted
**Refina:** `RF-05` de `04-specification.md`

Un ciclo de referencias produce un diagnóstico `dependency_cycle` en el plan, no un error.
La resolución sigue detectando ciclos y los expone; lo que no hace es abortar.

**Razón:** la resolución calcula un *conjunto* de recursos y la instalación escribe
archivos independientes, así que no existe requisito de orden que un ciclo pueda violar.
Además el catálogo heredado contiene referencias mutuas legítimas en cuatro packs: un
workflow nombra al agente que lo orquesta y ese agente nombra el workflow que ejecuta.
Tratarlas como error fatal dejaba `context`, `design`, `backend-design` y
`fullstack-design` sin poder instalarse.

## D-028 — Dos recursos no pueden escribir el mismo archivo

**Estado:** Accepted

Si dos recursos distintos resuelven al mismo archivo de destino, el plan se bloquea con
`destination_conflict` y no escribe nada.

**Razón:** el flujo heredado resolvía esta situación con la regla "si ya fue instalado por
otro pack, no lo copies de nuevo", que conserva silenciosamente el primero. Los packs
`backend` y `frontend` traen ambos un `feature-development.md` con **contenido distinto**,
de modo que instalar los dos perdía una de las dos versiones sin avisar. El conflicto es
un defecto del catálogo que hay que corregir en el catálogo, no una ambigüedad que la
herramienta deba resolver por su cuenta.

## D-029 — La CLI es la única superficie que seguirá evolucionando

**Estado:** Accepted
**Fecha:** 2026-07-30
**Reemplaza:** la obligación futura de compatibilidad bidireccional de D-022

`kits-init` queda deprecado y congelado. No recibirá nuevas funcionalidades ni condicionará
los contratos futuros. Agent Kits evolucionará exclusivamente como CLI.

Las sources Git permanecen: pertenecen a la arquitectura de obtención de la CLI y no al
bootstrap conversacional.

**Transición:** el comportamiento heredado puede permanecer temporalmente para permitir
una migración segura, pero no es una segunda superficie soportada a largo plazo.

## D-030 — El lockfile será la única fuente de verdad del proyecto

**Estado:** Accepted
**Fecha:** 2026-07-30
**Reemplaza parcialmente:** D-018 y D-022

`.agents/agent-kits.lock.json` concentrará todo el estado operativo que Agent Kits necesita.
`workspace.json` se eliminará después de una migración explícita y sin pérdida.

El cambio requiere un nuevo `schema_version` del lockfile. Ningún comando normal podrá
eliminar ni ignorar un `workspace.json` pendiente de migración.

## D-031 — La retirada de `workspace.json` usa una migración temporal y reversible

**Estado:** Accepted
**Fecha:** 2026-07-30

La CLI incorporará temporalmente:

```text
agent-kits migrate --project <path> [--yes] [--json]
```

La migración:

- genera y valida el nuevo lockfile antes de retirar `workspace.json`;
- preserva los datos operativos en el schema nuevo;
- conserva los datos históricos o desconocidos dentro del registro de migración;
- crea `.agents/workspace.json.migrated.bak`;
- usa el journal existente para aplicar lockfile, backup y retirada como una sola
  operación recuperable;
- aborta sin modificar el proyecto si encuentra ambigüedad, divergencia o datos que no
  puede preservar.

`migrate` y el alias heredado `import` se retirarán en un cambio posterior expresamente
aprobado. La copia de seguridad pertenece al usuario y no se eliminará automáticamente.

## D-032 — El catálogo objetivo será mínimo y nativo

**Estado:** Accepted
**Fecha:** 2026-07-30
**Reemplaza al completar la transición:** D-026

Los recursos que el usuario decida conservar se describirán mediante manifests nativos
`agent-kit.json`. Después de migrarlos y verificar el catálogo se eliminarán el loader
Markdown heredado y el parser de frontmatter si ya no tienen consumidores.

No se elimina ningún recurso hasta que el usuario entregue y apruebe la lista exacta.
La selección del catálogo es una puerta bloqueante, no una decisión que el implementador
pueda inferir.

## D-033 — El núcleo conserva los cuatro tipos de recurso

**Estado:** Accepted
**Fecha:** 2026-07-30

La reducción del catálogo no reduce el vocabulario del núcleo. `skill`, `agent`,
`workflow` y `kit` continúan soportados por manifests, resolución, planificación,
instalación y lockfile aunque el catálogo mínimo tenga pocos recursos de algún tipo.

## D-034 — El catálogo mínimo aprobado es el catálogo completo

**Estado:** Accepted
**Fecha:** 2026-07-30
**Cierra el gate de:** D-032

El usuario aprobó conservar los **75 recursos** existentes: 50 skills, 7 kits, 11 agentes y
7 workflows. Ninguno se elimina en este cambio.

**Razón:** podar es barato y reversible —borrar directorios y actualizar
`catalog-index.md`—, mientras que decidir qué se pierde no lo es. Conservar todo permite
migrar a manifests nativos y retirar el loader heredado ahora, y dejar la selección para
cuando el uso real la informe.

Consecuencias operativas, decididas junto con la lista:

1. **Versión inicial `1.0.0`** para los 65 recursos que no declaran ninguna. Los 10 que sí
   la declaran (3 skills y los 7 packs) conservan la suya. El `0.0.0` que sintetizaba el
   loader heredado desaparece: sin versiones reales, `update` no distingue cambios de
   contenido dentro de una misma versión (`06-legacy-baseline.md §8`, Hallazgo 4).

2. **Un directorio por recurso.** El loader nativo asocia un `agent-kit.json` a su
   directorio, así que los 18 recursos que eran un `.md` suelto —4 agentes globales, 7
   agentes de kit y 7 workflows— pasan a tener carpeta propia. Es un cambio del *layout de
   la source*, no del destino: el archivo se instala exactamente en la misma ruta que
   antes, porque el destino se deriva del nombre de archivo declarado, no de su ubicación
   en el repositorio.

3. **Se renombran los archivos, no los IDs.** Los dos `feature-development` pasan a
   `backend-feature-development.md` y `frontend-feature-development.md`. Los IDs canónicos
   `backend/feature-development` y `frontend/feature-development` no cambian: lo que
   colisionaba era el destino `.agents/workflows/feature-development.md`, no la identidad
   (D-028, Hallazgo 2).

**No decide** qué recursos sobrevivirán a una poda futura: esa sigue siendo una decisión
del usuario, ahora sin bloquear la retirada del legado.

## D-035 — La identidad de un recurso es un UUID

**Estado:** Accepted
**Fecha:** 2026-07-30
**Reemplaza:** D-019
**Refina:** D-006

Un recurso se identifica con un **UUID v4** estable, asignado una sola vez y para siempre.
El nombre, el tipo, la versión, la source y la pertenencia a un kit son atributos que
pueden cambiar sin que cambie la identidad.

**Razón:** D-019 ató la identidad a la pertenencia a un kit (`<kit>/<name>`), pero esa
pertenencia es una **relación**, no una propiedad: un recurso puede pasar de un kit a otro,
o dejar de pertenecer a ninguno. Con IDs calificados, mover `frontend-design` del kit
`frontend` al kit `design` cambiaría su identidad y rompería todo lockfile que lo
referencie. Un UUID desacopla las dos cosas.

**Consecuencias:**

- `D-006` se cumple por construcción: dos recursos no pueden compartir UUID.
- La pertenencia a un kit pasa a expresarse como dependencia, igual que cualquier otra.
- Un recurso publicado conserva su UUID, así que "el mismo recurso en el privado y en el
  público" es un hecho verificable y no una coincidencia de nombres (ver D-038).
- Renombrar un recurso deja de ser una operación destructiva.

## D-036 — El nombre es el nombre de instalación

**Estado:** Accepted
**Fecha:** 2026-07-30

Cada recurso declara un `name` en kebab-case. Ese nombre es, a la vez:

- cómo se lo pide (`agent-kits install frontend-design`);
- dónde aterriza al instalar (`.agents/skills/frontend-design/`).

No lleva prefijo de kit ni de source. El nombre legible para humanos vive aparte, en
`title`.

**Unicidad:** el nombre es único **dentro de una source**. Entre sources distintas puede
repetirse: dos organizaciones pueden publicar cada una su `frontend-design` y son recursos
distintos, con UUIDs distintos.

**Referencias:**

| Forma | Cuándo |
|---|---|
| `frontend-design` | el nombre identifica un solo recurso entre las sources configuradas |
| `acme:frontend-design` | varias sources traen ese nombre |
| `9f2c…b41e` | el UUID siempre funciona, sin ambigüedad posible |

Una referencia ambigua devuelve `ambiguous_id` y lista los candidatos calificados por
source. No se reusa ningún código de error nuevo: la ambigüedad ya tenía el suyo.

**Dos recursos con el mismo nombre de instalación no pueden coexistir en un proyecto**,
porque dos archivos no pueden ocupar la misma ruta. Eso sigue siendo `destination_conflict`
(D-028) y se resuelve renombrando, que ahora es barato.

## D-037 — Topología: la herramienta y el catálogo viven separados

**Estado:** Accepted
**Fecha:** 2026-07-30
**Cierra:** la pregunta abierta 10 (`origin`)

Tres repositorios, creados el 2026-07-30:

| Repositorio | Contenido | Visibilidad |
|---|---|---|
| [`LuchoC-Dev/agent-kits`](https://github.com/LuchoC-Dev/agent-kits) | la CLI (`cmd/`, `internal/`, `docs/`) | público |
| [`LuchoC-Dev/repository-private`](https://github.com/LuchoC-Dev/repository-private) | el catálogo completo | privado |
| [`LuchoC-Dev/repository`](https://github.com/LuchoC-Dev/repository) | el subconjunto publicado del catálogo | público |

**Razón:** publicar es una operación sobre **contenido**, no sobre código. Mientras el
catálogo viva junto a la CLI, cada commit de la herramienta es también un commit del
catálogo y viceversa, y la publicación no puede razonarse por separado.

Los 75 recursos se mudaron al repositorio privado el 2026-07-30, con una historia limpia:
la evolución previa del catálogo queda en el historial de `LuchoC-Dev/agent-kits`.

## D-038 — Todo recurso nace privado; el público es un subconjunto publicado

**Estado:** Accepted
**Fecha:** 2026-07-30
**Refina:** D-005, D-007

Invariante: **privado ⊇ público**. Todo lo que existe en el público existe también en el
privado; lo inverso no. Nada entra al público sin una publicación explícita.

**Razón:** es la única disposición en la que una filtración requiere un acto deliberado. Si
un recurso pudiera nacer público, bastaría un error de destino para exponerlo.

**Consecuencia sobre la unicidad:** un recurso publicado existe en las dos sources a la
vez. Eso no es un duplicado: es el mismo recurso, y se prueba porque comparten UUID
(D-035). Para que la vista agregada no lo trate como error de integridad, una source puede
declarar que **es el espejo publicado de otra**:

```json
{ "name": "public", "url": "…/repository.git", "publishes": "private" }
```

Entre dos sources emparentadas así, un UUID repetido es esperado y gana el **origen
privado**, que por construcción está igual o más adelantado. Entre sources **no**
emparentadas, un UUID repetido sigue siendo `registry_integrity_error`.

La precedencia deja de ser un desempate implícito —lo que D-006 prohíbe— y pasa a ser una
relación declarada por quien configura las sources.

## D-039 — La publicación la ejecuta CI, no la CLI

**Estado:** Accepted
**Fecha:** 2026-07-30
**Preserva:** D-003, D-004

Publicar es un workflow de CI en el repositorio privado que abre un **pull request** contra
el público. La CLI no adquiere ninguna capacidad de escritura remota: su lista blanca de
subcomandos de Git sigue siendo de solo lectura y `version --json` sigue informando
`remote_writes: false`.

**Razón:** una CLI capaz de hacer push al repositorio público convierte el compromiso de
una laptop —o de un agente con acceso a la herramienta— en una filtración pública. La
credencial vive en CI, donde es auditable y revocable, y cada publicación queda como un PR
revisable.

**La publicación es transitiva.** Si lo que se publica depende de recursos privados, esos
recursos se publican con él: un recurso público con una dependencia privada sería
ininstalable. Por eso el PR debe enumerar el **cierre completo**, separando lo pedido de lo
arrastrado, para que aprobarlo sea una decisión informada y no un trámite.

Antes de abrir el PR, CI ejecuta el escaneo de secretos sobre el contenido a publicar. Un
match de alta confianza cancela la publicación.

**Credencial y capas de contención** (configuradas el 2026-07-30):

| Capa | Qué impide |
|---|---|
| el token sólo alcanza `repository`, nunca el privado | que una filtración exponga el catálogo privado |
| vive como secret en el repositorio privado | que se lea desde afuera |
| `main` del público está protegida con `enforce_admins` | que una credencial publique sin revisión, incluso siendo del dueño |
| el token no tiene permiso de *Administration* | que pueda desactivar esa protección |
| el PR enumera el cierre completo | que se publique una dependencia privada sin verla |

La tercera capa es la que hace que la primera no tenga que ser perfecta: el token actúa
con la identidad de su dueño, que es administrador, así que sin `enforce_admins` podría
empujar directo a `main` y saltarse el pull request. Con él activado no puede, y como
carece de permiso de administración tampoco puede desactivarlo. Sólo una persona, desde la
interfaz, puede hacerlo.

## Decisiones todavía necesarias

- ubicación definitiva del registro global de reserva de IDs para sources remotas
  (el MVP valida unicidad sobre la vista agregada, que alcanza para sources conocidas);
- rangos de versión compuestos;
- firma criptográfica de sources;
- adaptador de Codex;
- qué recursos podar del catálogo cuando el uso real lo informe (D-034 conservó todos).
