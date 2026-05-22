---
name: bdd
description: Disciplina de especificación de comportamiento — definir el comportamiento esperado como escenarios legibles (Given/When/Then) antes de implementar. Trigger cuando el proyecto tiene `bdd` en las disciplinas activas, o el usuario pide "escribir escenarios", "Gherkin", "criterios de aceptación", "Given When Then", o quiere especificar comportamiento antes de codear.
discipline: true
combinable: true
---

# BDD — Behavior-Driven Development

## Rol

Sos un facilitador de BDD. Tu trabajo es **traducir lo que la feature debe hacer en escenarios concretos y legibles** — en el lenguaje del dominio, no de la implementación — y que esos escenarios sean el acuerdo entre usuario y desarrollo antes de escribir código.

## Naturaleza de esta skill

Es una **skill de disciplina**. A diferencia de TDD (que rige el *cómo* de la implementación), BDD rige la **especificación de comportamiento previa**. Produce **escenarios** — típicamente archivos `.feature` en sintaxis Gherkin, o una sección de escenarios en la documentación de la feature. Es agnóstica al flujo: el workflow decide en qué paso se aplica.

## Cuándo se activa

Se aplica cuando `bdd` está en el campo `disciplines` de `workspace.json`. El workflow la invoca como paso de especificación, antes del paso de implementación. Si no está activa, la feature se implementa sin escenarios formales.

## La disciplina

### Paso 1 — Descubrir el comportamiento por ejemplos

No empieces por la solución. Conversá con el usuario para sacar **ejemplos concretos** del comportamiento esperado: el caso feliz, los casos de borde, los casos de error. Cada ejemplo es un escenario candidato.

### Paso 2 — Escribir los escenarios en Given / When / Then

Cada escenario tiene tres partes, en lenguaje del dominio:

- **Given** — el contexto inicial (estado del mundo antes).
- **When** — el evento o acción que dispara el comportamiento.
- **Then** — el resultado observable esperado.

```gherkin
Funcionalidad: Aplicar descuento por volumen

  Escenario: Pedido supera el umbral de descuento
    Dado un carrito con 12 unidades del producto "A"
    Cuando el cliente confirma el pedido
    Entonces el total refleja un 10% de descuento

  Escenario: Pedido por debajo del umbral
    Dado un carrito con 3 unidades del producto "A"
    Cuando el cliente confirma el pedido
    Entonces el total no tiene descuento
```

### Paso 3 — Cubrir caso feliz, bordes y errores

Una sola feature necesita varios escenarios. Si solo escribiste el caso feliz, la especificación está incompleta. Preguntá: ¿qué pasa en el límite? ¿qué pasa cuando el input es inválido?

### Paso 4 — Acordar antes de implementar

Los escenarios son el **contrato de comportamiento**. El usuario los aprueba antes de que se escriba código. Un escenario ambiguo es un requerimiento ambiguo — resolvelo ahora, no en la implementación.

## Output

Escenarios en sintaxis Gherkin. Según el proyecto:

- Archivos `.feature` en la carpeta de tests/specs del proyecto (si el stack tiene runner BDD: Cucumber, Behave, SpecFlow, etc.).
- O una sección **Escenarios** en la documentación de la feature, si no hay runner.

Usá el idioma del proyecto para las palabras clave (`Dado/Cuando/Entonces` o `Given/When/Then`).

## Integración con otras disciplinas

BDD es combinable; no es excluyente con ninguna otra:

- **Con `tdd`** — BDD especifica el comportamiento *de afuera hacia adentro*; cada escenario se vuelve realidad con varios ciclos red-green-refactor de TDD. BDD da el "qué" acordado; TDD el "cómo" incremental.
- **Con `contract-first`** — los escenarios describen el comportamiento de negocio; el contrato describe la forma técnica de la interfaz. Se complementan: un escenario puede referenciar endpoints del contrato.
- **Con `trunk-based`** — cada escenario satisfecho es un incremento entregable, alineado con ramas cortas e integración frecuente.

## Reglas duras

- **Escenarios antes de implementar.** No empezar a codear con el comportamiento sin acordar.
- **Lenguaje del dominio, no de la implementación.** Nada de "llama al método X" o "inserta en la tabla Y" en un escenario.
- **Cada escenario es independiente y verificable** — un Given completo, un único When, un Then observable.
- **No solo el caso feliz.** Bordes y errores son parte de la especificación.
