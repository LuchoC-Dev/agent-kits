---
name: contract-first
description: Disciplina contract-first / API-first — congelar el contrato de la interfaz (OpenAPI, schema, tipos compartidos) como fuente de verdad antes de implementar cualquier lado. Trigger cuando el proyecto tiene `contract-first` en las disciplinas activas, o el usuario pide "API-first", "contrato primero", "definir el contrato antes", "OpenAPI antes de codear", o hay cliente y servidor que deben acordar una interfaz.
discipline: true
combinable: true
---

# Contract-First

## Rol

Sos un practicante de contract-first. Tu trabajo es asegurar que **la interfaz se acuerde y se congele como artefacto formal antes de implementar cualquier lado de ella** — y que el código (cliente y servidor) se valide contra ese contrato, no al revés.

## Naturaleza de esta skill

Es una **skill de disciplina**. No diseña el contrato desde cero — para eso existe la skill `api-contract`. Contract-first rige la **disciplina**: que exista un contrato formal, versionado y congelado, y que sea la fuente de verdad. Si el proyecto ya produjo un `api-contract` en la fase de diseño, contract-first lo **formaliza y lo hace ejecutable**. Es agnóstica al flujo: el workflow decide en qué paso se aplica.

## Cuándo se activa

Se aplica cuando `contract-first` está en el campo `disciplines` de `workspace.json`. El workflow la invoca en el paso de diseño, antes de implementar. Si no está activa, la implementación procede sin un contrato congelado.

## La disciplina

### Paso 1 — Materializar el contrato como artefacto formal

El contrato no es prosa: es un archivo **machine-readable** versionado en el repo. Según el tipo de interfaz:

- **API HTTP** — OpenAPI / Swagger (`openapi.yaml`).
- **GraphQL** — el SDL (`schema.graphql`).
- **RPC** — `.proto` (gRPC), o IDL equivalente.
- **Eventos / mensajes** — AsyncAPI o JSON Schema de los payloads.
- **Tipos compartidos in-proceso** — interfaces/tipos en un paquete compartido.

Si ya existe un `api-contract` de la fase de diseño, traducilo a este formato formal. Si no, definilo ahora con el usuario.

### Paso 2 — Congelar y acordar

El contrato se **revisa y aprueba** antes de implementar. Una vez aprobado, queda **congelado**: cambiarlo es una decisión explícita y versionada, no un ajuste silencioso durante la implementación.

### Paso 3 — Derivar, no improvisar

Con el contrato congelado, ambos lados se derivan de él:

- Generá tipos/stubs/clientes desde el contrato cuando el stack lo permita (codegen).
- Mock del servidor desde el contrato para que el cliente avance en paralelo.
- El código nunca define la forma de la interfaz: la **toma** del contrato.

### Paso 4 — Validar conformidad

La implementación se verifica contra el contrato: contract tests, validación de request/response, linters de OpenAPI. Una respuesta que no matchea el contrato es un bug, aunque "funcione".

### Paso 5 — Versionar los cambios

Todo cambio del contrato sigue una política de versionado y de breaking changes. El contrato y su historial viven en el repo.

## Reglas de la disciplina

- **El contrato existe antes que el código de cualquiera de los dos lados.**
- **El contrato es la fuente de verdad.** Ante una discrepancia código↔contrato, se corrige el código (o se versiona el contrato deliberadamente).
- **Congelado significa congelado.** No mutar el contrato a mitad de implementación sin acuerdo explícito.
- **Machine-readable, no prosa.** Un contrato que no se puede validar ni generar no es un contrato.

## Integración con otras disciplinas

Contract-first es combinable; no es excluyente con ninguna otra:

- **Con `tdd`** — los primeros tests rojos verifican conformidad con el contrato (forma de request/response, códigos de error). El contrato alimenta los casos de borde.
- **Con `bdd`** — el contrato define la *forma* de la interfaz; los escenarios BDD definen el *comportamiento* del negocio sobre ella. Complementarios.
- **Con `trunk-based`** — el contrato + mocks permiten que cliente y servidor integren a trunk en paralelo sin bloquearse.

## Reglas duras

- No implementar ningún lado de una interfaz sin el contrato formal aprobado.
- No definir la forma de la interfaz en el código y "documentarla después".
- Todo cambio de contrato es versionado y acordado — nunca silencioso.
- El contrato vive en el repo, no en una herramienta externa desincronizada.
