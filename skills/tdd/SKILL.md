---
name: tdd
description: Disciplina de ejecución Test-Driven Development — ciclo red → green → refactor. Escribir el test que falla antes que el código que lo hace pasar. Trigger cuando el proyecto tiene `tdd` en las disciplinas activas, o el usuario pide "hacer TDD", "test primero", "red green refactor", o está por implementar lógica con tests.
discipline: true
combinable: true
---

# TDD — Test-Driven Development

## Rol

Sos un practicante de TDD. Tu trabajo **no** es "agregar tests al final": es **conducir la implementación con tests**. Ningún código de producción se escribe sin un test que falle primero y lo justifique.

## Naturaleza de esta skill

Es una **skill de disciplina**, no de diseño. No produce un documento-artefacto en `docs/`; produce **código y tests** y rige *cómo* se ejecuta el paso de implementación. Es agnóstica al flujo: el workflow decide en qué paso se aplica.

## Cuándo se activa

Se aplica cuando `tdd` está en el campo `disciplines` de `workspace.json`. El workflow la invoca para envolver el paso de implementación. Si no está activa, el código se implementa sin esta disciplina.

## La disciplina — el ciclo

Por cada unidad de comportamiento, repetir el ciclo corto:

### 🔴 Red — escribir un test que falle

- Escribí **un solo** test que describa el próximo incremento de comportamiento deseado.
- El test debe **fallar por la razón correcta** (comportamiento ausente), no por un error de compilación o de setup. Corré la suite y confirmá el fallo.
- El test nombra el comportamiento, no la implementación: `calcula_total_con_descuento`, no `llama_a_getDiscount`.

### 🟢 Green — el mínimo código para pasar

- Escribí **lo mínimo** que haga pasar el test. Está permitido hardcodear, duplicar o ser "feo" en este paso.
- No agregues lógica que ningún test exija todavía. Si la querés, primero escribí su test (volvé a Red).
- Corré la suite completa: todo en verde.

### 🔵 Refactor — limpiar con la red puesta

- Con todos los tests en verde, mejorá el diseño: eliminá duplicación, renombrá, extraé funciones.
- No cambies comportamiento. Si la suite se rompe, el refactor estuvo mal — revertí.
- Refactorizá **tanto el código como los tests**.

Repetir hasta que la unidad de comportamiento esté completa.

## Reglas de la disciplina

- **Un test que falla antes que cualquier código de producción.** Sin excepción.
- **Pasos chicos.** Si un ciclo se está volviendo largo, el incremento era demasiado grande — partilo.
- **Suite siempre verde** al cerrar cada ciclo. Nunca dejar tests rojos "para después".
- **Test → razón de fallo correcta.** Un test que pasa apenas se escribe no probó nada.

## Integración con otras disciplinas

TDD es combinable; no es excluyente con ninguna otra disciplina:

- **Con `bdd`** — los escenarios Gherkin definen el comportamiento *de afuera*; TDD conduce las unidades *por dentro*. Un escenario BDD se satisface con varios ciclos red-green-refactor. BDD da el "qué"; TDD el "cómo, en chico".
- **Con `contract-first`** — el contrato congelado es la fuente de verdad de los tests de borde (request/response, errores). Los primeros tests rojos verifican conformidad con el contrato.
- **Con `trunk-based`** — cada ciclo verde es un punto seguro para un commit chico. El ritmo de TDD alimenta naturalmente la cadencia de commits de trunk-based.

## Reglas duras

- No escribir código de producción sin un test rojo que lo pida.
- No avanzar de Green a un nuevo Red sin pasar por Refactor (aunque el refactor sea "nada que limpiar").
- No mezclar refactor con cambio de comportamiento en el mismo paso.
- La cobertura es consecuencia, no objetivo: no escribir tests sin comportamiento que los motive.
