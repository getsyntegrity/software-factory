# AIF-001 · Fuentes, contratos y matriz de paridad

Documento canónico de la historia [#376](https://github.com/getsyntegrity/software-factory/issues/376),
parte del épico [#375](https://github.com/getsyntegrity/software-factory/issues/375).
Congela las fuentes que el port Go (`aifactory`) tiene que igualar, el inventario
de la Software Factory v2, la matriz de paridad que demuestra esa igualdad, las
extensiones que no forman parte de la paridad y la prueba de fase 0 con Gentle y
MCP. No describe código existente: las piezas Go que nombra están previstas, no
implementadas.

## Fuentes y SHA

Estos valores se copian tal cual de las decisiones del autor en #376 y no se
cambian sin una nueva historia.

| Fuente | Valor |
|---|---|
| Repositorio destino | `getsyntegrity/software-factory` |
| Commit de la línea base funcional (v2 en `main`) | `4153c2bb686a551cb293e295430aee38fd3857cb` |
| Árbol raíz de ese commit | `694f953b7941adc4398565bbdd0de222b71a7968` |
| Fuente v2 canónica | `zazencodes/zazencodes-season-3`, rama `main`, commit `de4598a8fab54425366952d886bbdb083f075af6`, ruta `src/software-factory-claude-code-v2` |
| HEAD observado al copiar la fuente | `94f7ae7b9dced7ffa87fb4a58312c89522ee120c` |
| Tamaño | 53 archivos, 3.435.304 bytes |

En los dos commits de la fuente, la ruta `src/software-factory-claude-code-v2`
apunta al mismo árbol `694f953b7941adc4398565bbdd0de222b71a7968`, así que no hay
divergencia entre el commit del plan y el HEAD usado en la copia.

`arch/factoryd-go@7baecc95d7172bebccba8f592b4eac891db5f945` es la cáscara Go
donde vive este documento. **No es la línea base funcional**: esa es v2 en el
commit `4153c2bb686a551cb293e295430aee38fd3857cb` de `main`. La rama base de la
fábrica sigue siendo `arch/factoryd-go` (no se cambia `factory/config.json`): el
documento se entrega ahí porque `main` va a ser reemplazada y lo que se entregue
en `main` se pierde.

El árbol de v2 se lee en solo lectura desde ese commit del mismo repo, sin
checkout ni copia de archivos. Ese commit es la fuente del inventario y de la
matriz de paridad:

```sh
git rev-parse 4153c2bb686a551cb293e295430aee38fd3857cb^{tree}
# 694f953b7941adc4398565bbdd0de222b71a7968
git ls-tree -r -l 4153c2bb686a551cb293e295430aee38fd3857cb
# 53 blobs; la suma de tamaños es 3435304
git show 4153c2bb686a551cb293e295430aee38fd3857cb:<ruta>
```

## Alcance / Fuera de alcance

**Alcance de esta historia:** fijar fuentes y SHA; inventariar los 53 archivos de
v2; asignar a cada comportamiento v2 un identificador de paridad, su futura pieza
Go y una prueba prevista; listar aparte las extensiones ODD, MCP y PDF; fijar la
versión de Gentle y el contrato MCP; describir la prueba de fase 0.

**Fuera de alcance:**

- Implementación Go (paquetes, tests o directorios vacíos para las piezas futuras).
- UI.
- Scanners.
- Cambios en Gentle.
- Ejecutar la prueba de fase 0 o registrar la salida real de `gentle-ai version`.
- Copiar archivos de v2 (incluidos PNG y jobs de ejemplo) a esta rama.
- Versionar o transcribir el PDF de la guía, o crear `docs/sources/guide-p9-10-notes.md`.
- Historias posteriores de #375, entre ellas #377 (AIF-002) y el walking slice de #379.

## Inventario v2

Los 53 archivos de `4153c2bb686a551cb293e295430aee38fd3857cb`, en el orden de
`git ls-tree -r`. Cada fila lleva su ruta y su blob SHA, para que el recuento sea
reproducible, y apunta a uno o más identificadores de la matriz de paridad o
lleva la etiqueta `sin comportamiento de runtime` con su motivo.

| Ruta | Blob SHA | Mapeo |
|---|---|---|
| `.claude/agents/approver.md` | `0383132ebe10e480e632c1d4d4e31ce06c11db05` | PAR-ENT-03, PAR-ENT-04 |
| `.claude/agents/builder.md` | `c5118242408a6a9116db4c5affc7f636dbc79d8b` | PAR-ENT-01, PAR-ENT-02, PAR-COR-04, PAR-GATE-04, PAR-REA-03, PAR-REV-04 |
| `.claude/agents/intake.md` | `f8b545727c47a236b5ae4ed55e1a72678e582948` | PAR-INT-03 |
| `.claude/agents/reviewer-code.md` | `22bd6206c23ac8afdade9c4d71a18ce42db077af` | PAR-REV-01, PAR-REV-03 |
| `.claude/agents/reviewer-security.md` | `f6fe7e49376339e86313c7538a38d4851880d02f` | PAR-REV-01, PAR-REV-03 |
| `.claude/agents/reviewer-ui.md` | `6607cd84c2a54c7a2298bb35fbb07e8007b1832c` | PAR-REV-01, PAR-REV-03, PAR-REV-04 |
| `.claude/agents/reviewer-ux.md` | `ed46401ad0551e159d5f87976aa92b2973ab56f0` | PAR-REV-01, PAR-REV-03, PAR-REV-04 |
| `.claude/agents/spec-writer.md` | `5a3c7a52742b27bb3b49325073caaa927dd19d07` | PAR-ART-01, PAR-ART-02 |
| `.claude/skills/factory/SKILL.md` | `b37895c07a50a4da725860658034341c58b30426` | PAR-INT-05, PAR-REV-01, PAR-REV-04, PAR-COR-01, PAR-COR-03, PAR-REA-01, PAR-REA-03 |
| `DESIGN.md` | `4ce2e35fbfe51271509c340aa6c860d569fea9fd` | sin comportamiento de runtime: prosa que describe el pipeline; los comportamientos que cita los poseen las filas de `factory/bin/*`, agentes y `SKILL.md` |
| `INSTALL.md` | `9c91869f3d4613404b2b3ce61bcd9a3b71e8b5d8` | PAR-INT-06 (etiquetas `factory`, `factory:needs-info`, `factory:needs-human`), PAR-ENT-03 (`merge_method` permitido por el repo), PAR-EST-04 (`concurrency`), PAR-GATE-01 (reglas de `checks.sh`), PAR-ART-01 (`/factory/jobs/` y `/.worktrees/` ignorados); los pasos de copia son instrucciones para una persona |
| `LICENSE` | `f288702d2fa16d3cdf0035b15a9fcbc552cd88e7` | sin comportamiento de runtime: licencia GPLv3 |
| `README.md` | `f0e64f4d033ed85dcb420a7b5675ace935bd7b2b` | sin comportamiento de runtime: presentación del proyecto |
| `docs/assets/software-factories-guide-promo.png` | `6e199102a33647413934c4252e83fb2edfdf6248` | sin comportamiento de runtime: imagen promocional |
| `docs/assets/zazencodes-banner.png` | `3aa8f522ef494da53a74d0f4548c0e8aa57d6d87` | sin comportamiento de runtime: imagen de cabecera |
| `examples/jobs/001-add-a-subtract-function/build.md` | `29333e1af3b05735954522efc5cc925a31c40c67` | sin comportamiento de runtime: fixture del job 001 (aprobado y mergeado en la ronda 1); dato de prueba para PAR-ART-01 y PAR-ART-02 |
| `examples/jobs/001-add-a-subtract-function/decision.md` | `2028266044a405d6a566810f8c6b95899b312025` | sin comportamiento de runtime: fixture del job 001 (aprobado y mergeado en la ronda 1); dato de prueba para PAR-ART-01 y PAR-ART-02 |
| `examples/jobs/001-add-a-subtract-function/intake.md` | `3e8864c700b4134c3554c43b6d811dd0b920e52a` | sin comportamiento de runtime: fixture del job 001 (aprobado y mergeado en la ronda 1); dato de prueba para PAR-ART-01 y PAR-ART-02 |
| `examples/jobs/001-add-a-subtract-function/issue.md` | `6384c00cd2cd494a4ce29c323f5bf2aec3ce0003` | sin comportamiento de runtime: fixture del job 001 (aprobado y mergeado en la ronda 1); dato de prueba para PAR-ART-01 y PAR-ART-02 |
| `examples/jobs/001-add-a-subtract-function/job.json` | `32fe13f31b60f0e2df25d6249200df86764ff899` | sin comportamiento de runtime: fixture del job 001 (aprobado y mergeado en la ronda 1); dato de prueba para PAR-ART-01 y PAR-ART-02 |
| `examples/jobs/001-add-a-subtract-function/round-1/checks.md` | `053fd001722f7a80bfa0c6fb0ba1da8558b58bb0` | sin comportamiento de runtime: fixture del job 001 (aprobado y mergeado en la ronda 1); dato de prueba para PAR-ART-01 y PAR-ART-02 |
| `examples/jobs/001-add-a-subtract-function/round-1/merge-checks.md` | `738850dc05aaaa1286dbc1b6975845482c9291b2` | sin comportamiento de runtime: fixture del job 001 (aprobado y mergeado en la ronda 1); dato de prueba para PAR-ART-01 y PAR-ART-02 |
| `examples/jobs/001-add-a-subtract-function/round-1/review-code.md` | `34b5ee27250e642f4cc953d27d6b3be35e637b30` | sin comportamiento de runtime: fixture del job 001 (aprobado y mergeado en la ronda 1); dato de prueba para PAR-ART-01 y PAR-ART-02 |
| `examples/jobs/001-add-a-subtract-function/spec.md` | `87fda89f6bc19c95c6f9f1583549d9a991b0e65e` | sin comportamiento de runtime: fixture del job 001 (aprobado y mergeado en la ronda 1); dato de prueba para PAR-ART-01 y PAR-ART-02 |
| `examples/jobs/002-show-a-history-of/build.md` | `b44b6aefbe154ef264488d4910102c3523d7cf3e` | sin comportamiento de runtime: fixture del job 002 (superficie `ui`, tres reviewers); dato de prueba para PAR-INT-04 y PAR-REV-01 |
| `examples/jobs/002-show-a-history-of/decision.md` | `8445e183431834e7bfce354a5cc99b8e51f60e5a` | sin comportamiento de runtime: fixture del job 002 (superficie `ui`, tres reviewers); dato de prueba para PAR-INT-04 y PAR-REV-01 |
| `examples/jobs/002-show-a-history-of/intake.md` | `4a1cd55fbdd80e6d56c796bdd463d91c9749c4d9` | sin comportamiento de runtime: fixture del job 002 (superficie `ui`, tres reviewers); dato de prueba para PAR-INT-04 y PAR-REV-01 |
| `examples/jobs/002-show-a-history-of/issue.md` | `29727c82b68143ff5734a63b8b1359d3ecd521ad` | sin comportamiento de runtime: fixture del job 002 (superficie `ui`, tres reviewers); dato de prueba para PAR-INT-04 y PAR-REV-01 |
| `examples/jobs/002-show-a-history-of/job.json` | `e8fa389f2fc333084d7e1ae78f1d97ff80c7814a` | sin comportamiento de runtime: fixture del job 002 (superficie `ui`, tres reviewers); dato de prueba para PAR-INT-04 y PAR-REV-01 |
| `examples/jobs/002-show-a-history-of/round-1/checks.md` | `c7dd4d04123d860df540ff6df7cb258e3b6778b3` | sin comportamiento de runtime: fixture del job 002 (superficie `ui`, tres reviewers); dato de prueba para PAR-INT-04 y PAR-REV-01 |
| `examples/jobs/002-show-a-history-of/round-1/merge-checks.md` | `b16a1b6ce717fc61ce316dd38a45e630fcdfede1` | sin comportamiento de runtime: fixture del job 002 (superficie `ui`, tres reviewers); dato de prueba para PAR-INT-04 y PAR-REV-01 |
| `examples/jobs/002-show-a-history-of/round-1/review-code.md` | `78828675d6cecce6a982b6733b411466a87152fd` | sin comportamiento de runtime: fixture del job 002 (superficie `ui`, tres reviewers); dato de prueba para PAR-INT-04 y PAR-REV-01 |
| `examples/jobs/002-show-a-history-of/round-1/review-ui.md` | `0bb74916abbdfcfb603fac1f6244d339fdb3df58` | sin comportamiento de runtime: fixture del job 002 (superficie `ui`, tres reviewers); dato de prueba para PAR-INT-04 y PAR-REV-01 |
| `examples/jobs/002-show-a-history-of/round-1/review-ux.md` | `6f96cb9b53271a94db3e856f876c3749c176500a` | sin comportamiento de runtime: fixture del job 002 (superficie `ui`, tres reviewers); dato de prueba para PAR-INT-04 y PAR-REV-01 |
| `examples/jobs/002-show-a-history-of/spec.md` | `75f50cb0a3e25e3f6ac2cafa5dc1436a20b8bdbb` | sin comportamiento de runtime: fixture del job 002 (superficie `ui`, tres reviewers); dato de prueba para PAR-INT-04 y PAR-REV-01 |
| `examples/jobs/003-add-multiply-divide-power/build.md` | `ef1c44d3475c768cc6c42749524fb67c821289b6` | sin comportamiento de runtime: fixture del job 003 (gate FAIL en la ronda 1 y `needs-human` al agotar la ronda 3); dato de prueba para PAR-GATE-03, PAR-COR-01 y PAR-COR-02 |
| `examples/jobs/003-add-multiply-divide-power/decision.md` | `1ec38c6db172c09c0cf138a176d16364e2b67fd1` | sin comportamiento de runtime: fixture del job 003 (gate FAIL en la ronda 1 y `needs-human` al agotar la ronda 3); dato de prueba para PAR-GATE-03, PAR-COR-01 y PAR-COR-02 |
| `examples/jobs/003-add-multiply-divide-power/intake.md` | `11b4e8be48866d15648fa2841a8f47be9d998427` | sin comportamiento de runtime: fixture del job 003 (gate FAIL en la ronda 1 y `needs-human` al agotar la ronda 3); dato de prueba para PAR-GATE-03, PAR-COR-01 y PAR-COR-02 |
| `examples/jobs/003-add-multiply-divide-power/issue.md` | `c708230109e2d5266048adad4bcc6336afe44736` | sin comportamiento de runtime: fixture del job 003 (gate FAIL en la ronda 1 y `needs-human` al agotar la ronda 3); dato de prueba para PAR-GATE-03, PAR-COR-01 y PAR-COR-02 |
| `examples/jobs/003-add-multiply-divide-power/job.json` | `79313674b8522ddf0a4ea7ee97bc91caf85e6ffb` | sin comportamiento de runtime: fixture del job 003 (gate FAIL en la ronda 1 y `needs-human` al agotar la ronda 3); dato de prueba para PAR-GATE-03, PAR-COR-01 y PAR-COR-02 |
| `examples/jobs/003-add-multiply-divide-power/round-1/checks.md` | `debc5ca0b72ec67e170084990be35c62a230c8a8` | sin comportamiento de runtime: fixture del job 003 (gate FAIL en la ronda 1 y `needs-human` al agotar la ronda 3); dato de prueba para PAR-GATE-03, PAR-COR-01 y PAR-COR-02 |
| `examples/jobs/003-add-multiply-divide-power/round-2/checks.md` | `8a10944245d55a2290b81ec5e49cbc8f12f88cda` | sin comportamiento de runtime: fixture del job 003 (gate FAIL en la ronda 1 y `needs-human` al agotar la ronda 3); dato de prueba para PAR-GATE-03, PAR-COR-01 y PAR-COR-02 |
| `examples/jobs/003-add-multiply-divide-power/round-2/review-code.md` | `997e4cedf09271f9dbde8a1a06dfbcb6c33e204f` | sin comportamiento de runtime: fixture del job 003 (gate FAIL en la ronda 1 y `needs-human` al agotar la ronda 3); dato de prueba para PAR-GATE-03, PAR-COR-01 y PAR-COR-02 |
| `examples/jobs/003-add-multiply-divide-power/spec.md` | `9912a57eab3530b30c5d38ee9a36bf77328367d8` | sin comportamiento de runtime: fixture del job 003 (gate FAIL en la ronda 1 y `needs-human` al agotar la ronda 3); dato de prueba para PAR-GATE-03, PAR-COR-01 y PAR-COR-02 |
| `examples/jobs/005-add-a-crypto-miner/intake.md` | `3958474dd5b3ccb086940a4073b0301b4eaf59bb` | sin comportamiento de runtime: fixture del job 005 (rechazado en intake); dato de prueba para PAR-INT-03 y PAR-EST-01 |
| `examples/jobs/005-add-a-crypto-miner/issue.md` | `566a0dced951944541d44e4bf24ffa59c6be1d07` | sin comportamiento de runtime: fixture del job 005 (rechazado en intake); dato de prueba para PAR-INT-03 y PAR-EST-01 |
| `examples/jobs/005-add-a-crypto-miner/job.json` | `31f3cb24d772cb1d96d95988ee2959a913969a94` | sin comportamiento de runtime: fixture del job 005 (rechazado en intake); dato de prueba para PAR-INT-03 y PAR-EST-01 |
| `factory/bin/gate` | `8c2d91db8b51cc9729c075a60a6da676be81bc87` | PAR-GATE-01, PAR-GATE-02, PAR-GATE-04, PAR-ENT-03 |
| `factory/bin/job` | `f1bdd3a099aaf8d464201e6d148689d93d5ed9b8` | PAR-INT-01, PAR-INT-02, PAR-INT-05, PAR-INT-06, PAR-EST-01 a PAR-EST-04, PAR-ART-02 a PAR-ART-05, PAR-GATE-03, PAR-REV-02, PAR-COR-01 a PAR-COR-05, PAR-REA-02, PAR-REA-04, PAR-ENT-01, PAR-ENT-03, PAR-ENT-04 |
| `factory/bin/route` | `604cc5be36bbe7b9188e6af0b6cf00d03bada06b` | PAR-INT-04 |
| `factory/checks.sh` | `a1757ed4d3b89222db8e4b8b6cfdcd1e6fc02a8b` | PAR-GATE-01 |
| `factory/config.json` | `31ae5d3c9bc3814394c8ac003f0a14eacd54d91c` | PAR-INT-04, PAR-EST-04, PAR-GATE-02, PAR-COR-02, PAR-ART-04, PAR-ENT-03, PAR-REV-03, PAR-REV-04 |
| `factory/dashboard.html` | `aa7c005c377558a7a6948e135365269351f60d66` | PAR-ART-05 (solo el modelo de lectura; la UI HTML queda fuera de alcance) |

## Matriz de paridad

Una fila por comportamiento v2. Las piezas Go son nombres de paquete previstos
(`internal/<área>`); ninguna existe todavía. Las rutas de origen son relativas a
`4153c2b`.

| ID | Área | Comportamiento v2 | Futura pieza Go | Prueba prevista |
|---|---|---|---|---|
| PAR-INT-01 | intake | `job create <issue>` crea un job solo para un issue abierto con la etiqueta `factory`, como mucho uno por issue, con id `NNN-slug` incremental (`factory/bin/job`, `cmd_create`). | `internal/jobs` (alta de job) | Tabla: issue cerrado, sin etiqueta o con job previo se rechaza; el id siguiente es `max+1` con slug de cuatro palabras. |
| PAR-INT-02 | intake | `issue.md` cita el issue como datos, con aviso de no seguir instrucciones, y solo incluye comentarios del autor y del operador (`factory/bin/job`, `write_issue_md`). | `internal/intake` (archivo de issue) | Golden de `issue.md`; un comentario de un tercero no aparece. |
| PAR-INT-03 | intake | El agente de intake escribe `intake.md` con cabecera `DECISION` (ACCEPT, NEEDS-INFO, REJECT), `RISK`, `SURFACES`, `RESCOPED` (`.claude/agents/intake.md`, `sync`). | `internal/artifacts` (cabecera de intake) | Parseo de cabeceras válidas e inválidas; `SURFACES: none` produce lista vacía. |
| PAR-INT-04 | intake | `route` mapea riesgo y superficies a reviewers y modelos con `routing` y `surface_reviewers`; rechaza valores desconocidos; solo se registra tras ACCEPT (`factory/bin/route`, `factory/config.json`, `cmd_agents`). | `internal/routing` | Tabla low/medium/high con y sin `ui`/`security`; error en riesgo o superficie desconocidos. |
| PAR-INT-05 | intake | `job queue` clasifica cada issue etiquetado como `resume`, `new`, `hold`, `wait` o `skip`, respetando `concurrency` (`factory/bin/job`, `cmd_queue`; `SKILL.md`). | `internal/queue` | Escenarios con slots llenos y libres; el orden de filas coincide con v2. |
| PAR-INT-06 | intake | Las transiciones de intake comentan en el issue con el marcador `<!-- factory -->` y ponen o quitan las etiquetas `factory:needs-info` y `factory` (`factory/bin/job`, `advance`). | `internal/github` (comentarios y etiquetas) | Doble de GitHub registra comentario y etiquetas esperados por transición. |
| PAR-EST-01 | estados | `stage` es uno de 11 estados: `intake`, `needs-info`, `spec`, `build`, `gate`, `review`, `approve`, `merging`, `needs-human`, `merged`, `rejected`. Los movimientos legales salen de `TRANSITIONS` en `factory/bin/job`; además se pasa a `merged` desde cualquier estado no terminal si el PR está mergeado. | `internal/state` | Tabla de todos los pares origen→destino: solo los de `TRANSITIONS` (y `merged`) se aceptan. |
| PAR-EST-02 | estados | `merged` y `rejected` son terminales y borran worktree y rama; `needs-info` y `needs-human` son de espera; ambos fijan `ended` (`factory/bin/job`, `advance`). | `internal/state` | Al entrar en terminal se elimina el worktree; `ended` se fija y se limpia al salir de espera. |
| PAR-EST-03 | estados | `job` es el único escritor de `job.json`; `load` toma un `flock` por job y lo mantiene hasta que termina el proceso, así que cubre todo el comando (`factory/bin/job`, `load`/`save`). | `internal/store` | Dos escrituras concurrentes se serializan; ninguna se pierde. |
| PAR-EST-04 | estados | Un job ocupa slot si no es terminal ni de espera; `create` y `needs-info → intake` se rechazan sin slot libre (`require_slot`, `concurrency` en `factory/config.json`). | `internal/queue` | Con `concurrency` agotada, crear o reanudar se rechaza con el mensaje de v2. |
| PAR-ART-01 | artefactos | Cada job vive en `factory/jobs/<id>/`: `job.json`, `issue.md`, `intake.md`, `spec.md`, `build.md`, `round-<N>/checks.md`, `round-<N>/review-<r>.md`, `decision.md`, `feedback-<k>.md` (`SKILL.md`, `.claude/agents/spec-writer.md`). | `internal/store` (layout de job) | Recorrer un fixture de `examples/jobs/` y resolver cada ruta esperada. |
| PAR-ART-02 | artefactos | Contratos de primera línea: `STATUS: READY` o `STATUS: BLOCKED`, `VERDICT: PASS` o `VERDICT: CHANGES`, `APPROVE` o `ESCALATE`, `GATE: PASS` o `GATE: FAIL`, y cabeceras `CLAVE: valor` (`factory/bin/job`, `header`/`first_line`). Cada archivo de agente abre con `## Summary` de 3 a 5 viñetas, por convención de los prompts (`.claude/agents/*.md`); `factory/bin/job` no lo comprueba. | `internal/artifacts` | Parseo de fixtures. Un veredicto de review fuera de contrato queda como `invalid` en `sync`; una primera línea fuera de contrato en `intake.md`, `build.md`, `checks.md` o `decision.md` hace que `advance` rechace la transición con su mensaje. |
| PAR-ART-03 | artefactos | `job sync` deriva `intake`, `gate` y `reviews` de los archivos, nunca de memoria (`factory/bin/job`, `sync`). | `internal/jobs` (sincronización) | Modificar un archivo y sincronizar actualiza solo el campo derivado. |
| PAR-ART-04 | artefactos | El costo por job suma las transcripciones de subagentes que nombran su `Job folder:` y las valora con `prices_usd_per_mtok` (`factory/bin/job`, `cost_usd`; `factory/config.json`). | `internal/cost` | Transcripciones de fixture dan el costo esperado; las de otro job no cuentan. |
| PAR-ART-05 | artefactos | `job summary` y el tablero leen `job.json` y los archivos de ronda para mostrar etapa, ronda, PR, costo y tiempo (`factory/bin/job`, `cmd_summary`; `factory/dashboard.html`). | `internal/jobs` (modelo de lectura) | Golden de la tabla de resumen con jobs de fixture. |
| PAR-GATE-01 | gate | El gate corre el `factory/checks.sh` del checkout principal dentro del worktree y escribe `round-N/checks.log` con la salida completa y `round-N/checks.md` con `GATE`, `COMMIT` y las últimas 80 líneas del log (`factory/bin/gate`, `factory/checks.sh`). | `internal/gate` | Un checks que falla da `GATE: FAIL` con el commit actual; un checks editado en la rama no se usa. |
| PAR-GATE-02 | gate | Falla si la rama modifica o borra un archivo de `protected_paths` (fnmatch, `*` cruza `/`); añadir archivos es válido (`factory/bin/gate`, `factory/config.json`). | `internal/gate` (rutas protegidas) | Casos M, D, R y A sobre rutas protegidas; solo A pasa. |
| PAR-GATE-03 | gate | `gate → review` exige `COMMIT` igual a HEAD y `GATE: PASS`; con FAIL vuelve a `build`, o a `needs-human` en la última ronda (`factory/bin/job`, `advance`). | `internal/state` (guardas del gate) | Un `checks.md` de otro commit se rechaza; FAIL en la última ronda va a `needs-human`. |
| PAR-GATE-04 | gate | `gate --hook` corre los checks al parar el builder y sale con 2 si fallan, como aviso; el gate es quien decide (`factory/bin/gate`, `.claude/agents/builder.md`). | `internal/gate` (hook) | Hook con checks rojos sale 2; con `stop_hook_active` sale 0. |
| PAR-REV-01 | reviewers | Los reviewers enrutados corren en paralelo; se espera a todos y cada uno escribe `round-N/review-<r>.md` (`SKILL.md`, `.claude/agents/reviewer-*.md`). | `internal/review` | `review → approve` se rechaza mientras falte algún `review-<r>.md`. |
| PAR-REV-02 | reviewers | Solo se aprueba si todas las reviews dicen PASS; una primera línea que no es veredicto bloquea el avance (`factory/bin/job`, `check_reviews_done`). | `internal/review` | Una CHANGES impide `approve`; un veredicto inválido se rechaza. |
| PAR-REV-03 | reviewers | Cuatro lentes (`code`, `security`, `ui`, `ux`), cada una con su prompt y el modelo que le asigna `factory/config.json` (`.claude/agents/reviewer-*.md`). | `internal/roles` | Registro de las cuatro lentes con su modelo según la tabla de ruteo. |
| PAR-REV-04 | reviewers | Puertos por job, con `n` el número del job: el builder recibe `5100 + 10n`, `reviewer-ux` `5101 + 10n` y `reviewer-ui` `5102 + 10n`; `code` y `security` no tienen puerto (`ports` en `factory/config.json`; `SKILL.md`; `.claude/agents/builder.md`). | `internal/roles` (puertos) | Para el job `004`, builder 5140, ux 5141, ui 5142; code y security sin puerto. |
| PAR-COR-01 | correcciones | Con CHANGES o gate FAIL se abre la ronda siguiente: nueva carpeta `round-N`, `gate` y `reviews` vacíos (`factory/bin/job`, `next_round`; `SKILL.md`). | `internal/state` (rondas) | Tras CHANGES la ronda sube en 1 y la carpeta existe vacía. |
| PAR-COR-02 | correcciones | Como mucho `max_rounds` rondas; agotar la última escribe `decision.md` ESCALATE y va a `needs-human` (`escalate`, `factory/config.json`). | `internal/state` (límite de rondas) | Última ronda con CHANGES produce `decision.md` y `needs-human`. |
| PAR-COR-03 | correcciones | `job feedback` guarda `feedback-<k>.md`, pasa `needs-human → build`, da `max_rounds` rondas nuevas y quita la etiqueta (`cmd_feedback`; `SKILL.md`). | `internal/jobs` (feedback) | Feedback vacío se rechaza; con texto, `round_limit = round + max_rounds`. |
| PAR-COR-04 | correcciones | `STATUS: BLOCKED` en `build.md` lleva a `needs-human` con la sección `## Blocked` en `decision.md` (`advance`; `.claude/agents/builder.md`). | `internal/state` (bloqueo) | `build.md` BLOCKED produce `decision.md` con el texto bloqueante. |
| PAR-COR-05 | correcciones | `job reject` en espera escribe `rejection.md`, comenta el motivo, quita etiquetas, cierra el PR y borra la rama (`cmd_reject`, `advance`). | `internal/jobs` (rechazo) | Rechazar fuera de espera falla; en espera se registran cierre de PR y etiquetas. |
| PAR-REA-01 | reanudación | El estado vive en archivos: una sesión nueva reanuda cada job desde su etapa, respawneando solo lo que falta (`SKILL.md`, `cmd_queue`). | `internal/queue` (reanudación) | Reiniciar con jobs a medio camino devuelve `resume` con la etapa correcta. |
| PAR-REA-02 | reanudación | Un job `needs-info` se reanuda si el autor o el operador comentó después del último comentario de la fábrica; `intake.md` pasa a `intake-<k>.md` e `issue.md` se reescribe (`replied_since_question`, `advance`). | `internal/intake` (respuesta del autor) | Sin respuesta queda `wait`; con respuesta, `resume` y archivo renombrado. |
| PAR-REA-03 | reanudación | Cada prompt empieza con la cabecera del job (`Job folder:` incluido) y el builder se reanuda por nombre `builder-<id>` (`SKILL.md`, `.claude/agents/builder.md`). | `internal/roles` (cabecera de prompt) | Golden de la cabecera generada desde un `job.json`. |
| PAR-REA-04 | reanudación | Si el PR se mergeó a mano en GitHub, `job queue` mueve el job a `merged` (`cmd_queue`). | `internal/queue` | Doble de GitHub con PR MERGED pasa el job a `merged`. |
| PAR-ENT-01 | entrega | `build → gate` exige `STATUS: READY`, árbol limpio, commits sobre la base, rama empujada y un único PR abierto, que queda registrado (`factory/bin/job`, `advance`). | `internal/delivery` | Cada precondición incumplida da su mensaje de rechazo de v2. |
| PAR-ENT-02 | entrega | El builder solo empuja `factory/<job-id>` y abre un PR borrador `Closes #<n>`; la fábrica nunca despliega (`.claude/agents/builder.md`). | `internal/delivery` (PR) | Doble de GitHub: PR borrador con cuerpo `Closes #<n>`; otra rama se rechaza. |
| PAR-ENT-03 | entrega | APPROVE ejecuta `job merge`: sincroniza con la base, corre `gate --merge` (`merge-checks.md`), empuja, marca listo y hace `gh pr merge` con `merge_method`; cualquier fallo va a `needs-human` (`cmd_merge`, `factory/bin/gate`, `.claude/agents/approver.md`). | `internal/delivery` (merge) | Conflicto con la base o gate rojo tras sincronizar terminan en `needs-human`; éxito termina en `merged`. |
| PAR-ENT-04 | entrega | ESCALATE o cualquier paso a `needs-human` marca el PR listo, comenta la decisión en el PR (o en el issue sin PR) y pone `factory:needs-human` (`advance`, `.claude/agents/approver.md`). | `internal/delivery` (escalado) | Doble de GitHub registra PR listo, comentario y etiqueta. |

## Extensiones (no paridad)

Lo que el port añade y v2 no tiene. Se mide aparte y nunca cuenta como paridad.

| ID | Extensión | Contenido congelado |
|---|---|---|
| EXT-ODD-01 | ODD | Gentle coordina la conversación y ODD (Organic Driven Development); `aifactory` valida estado, persistencia, evidencia y efectos. No se crea un segundo orquestador. |
| EXT-ODD-02 | ODD | gentle-shell / gentle-pi entran en el alcance como extensión porque ODD es su flujo. Versión congelada: pendiente (sin valor; se fijará por tag o SHA publicado). |
| EXT-MCP-01 | MCP | Servidor `aifactory`, transporte `stdio`, local; cliente Claude Code con Gentle-AI v4.5.0. |
| EXT-MCP-02 | MCP | Contrato completo de nueve tools: `list_jobs`, `create_job`, `get_job`, `record_artifact`, `advance_job`, `run_checks`, `provide_feedback`, `deliver_job`, `stop_job`. |
| EXT-MCP-03 | MCP | Walking slice de #379: `create_job`, `get_job` y `list_jobs`. |
| EXT-MCP-04 | MCP | Schemas estrictos; un `request_id` repetido es idempotente. |
| EXT-PDF-01 | PDF, página 9 | Tras CI y review, la entrega es una decisión humana. Cuando los agentes abren más PR de los que una persona puede leer, el tablero los ordena por riesgo en P1/P2/P3, los separa en Now/Next y muestra en cada tarjeta una razón breve y la próxima acción (por ejemplo corregir CI o fusionar). |
| EXT-PDF-02 | PDF, página 10 | La vista empieza con un resumen del cambio y luego divide el PR en capas nombradas por propósito, en orden de lectura, para explicar el porqué que el diff solo no muestra. |
| EXT-PDF-03 | PDF, página 10 | La revisión hace visibles los riesgos de seguridad y hallazgos aplicables. En v0.1 se usan únicamente Gitleaks, Semgrep y OSV-Scanner cuando estén configurados. Una herramienta ausente o fallida nunca se presenta como PASS. |
| EXT-PDF-04 | PDF, referencia visual | Página 10 para detalle y recorrido; página 9 para el tablero Now/Next. No se inventan otros módulos de CodeRabbit. |

Fuente de las filas EXT-PDF: la guía *Build Your Own Software Factory in 5 Steps*
(Daniel Moka, copia de 14 páginas del 17/11/2026), páginas 9 y 10. El PDF es
material de terceros: no se versiona ni se transcribe. Las filas resumen lo que
el autor describió en #376. La fuente prevista para esas páginas son las notas
propias `docs/sources/guide-p9-10-notes.md` (pendiente: no existe todavía en
arch/factoryd-go).

## Gentle y contrato MCP

Las versiones se congelan por tag o SHA publicado, no por la versión instalada
en una máquina.

- **gentle-ai:** tag publicado `v4.5.0`, release oficial:
  <https://github.com/Gentleman-Programming/gentle-ai/releases/tag/v4.5.0>.
- **gentle-shell / gentle-pi:** versión pendiente, sin valor fijado todavía. Entran
  en el alcance como extensión porque ODD es su flujo.
- **Verificación local:** `gentle-ai version` (también valen `gentle-ai --version`
  y `gentle-ai -v`) comprueba que la máquina de prueba usa el tag congelado. La
  evidencia guarda la salida real. Si no informa `v4.5.0`, la prueba se bloquea y
  se informa la diferencia, sin asumir compatibilidad.
- **Cliente:** Claude Code con Gentle-AI v4.5.0.
- **Servidor:** `aifactory`, transporte `stdio`, local.
- **Contrato completo:** `list_jobs`, `create_job`, `get_job`, `record_artifact`,
  `advance_job`, `run_checks`, `provide_feedback`, `deliver_job`, `stop_job`.
- **Walking slice de #379:** `create_job`, `get_job`, `list_jobs`. Las otras seis
  tools llegan en historias posteriores.
- **Schemas:** estrictos. Un
  `request_id` repetido es idempotente: devuelve el mismo resultado sin repetir
  efectos.
- **Reparto de responsabilidades:** Gentle coordina la conversación y ODD;
  `aifactory` valida estado, persistencia, evidencia y efectos. No hay un segundo
  orquestador.

## Prueba de fase 0

Prueba Claude Code/Gentle → MCP. Aquí solo se describe; se ejecutará cuando exista
el walking slice de #379. Cada paso guarda su salida como evidencia.

| Paso | Entrada | Resultado observable |
|---|---|---|
| 1 | `gentle-ai version` en la máquina de prueba. | La salida informa `v4.5.0`. Si no, la prueba se bloquea y se informa la versión real. |
| 2 | Desde Claude Code con Gentle, descubrir los servidores MCP configurados. | Aparece el servidor `aifactory` con transporte `stdio`, conectado. |
| 3 | Listar las tools de `aifactory`. | Aparecen `create_job`, `get_job` y `list_jobs`. Las otras seis del contrato aún no existen en esta fase. |
| 4 | `create_job` con un job mínimo y `request_id` `fase0-001`. | Devuelve un ID de job nuevo; el schema estricto acepta la entrada. |
| 5 | `get_job` con el ID del paso 4. | Devuelve ese job con los datos enviados en el paso 4. |
| 6 | `list_jobs`. | La lista contiene exactamente un job, el del paso 4. |
| 7 | Repetir `create_job` con la misma entrada y el mismo `request_id` `fase0-001`. | Devuelve el mismo ID de job del paso 4, y `list_jobs` sigue mostrando un solo job: no se duplican efectos. |
