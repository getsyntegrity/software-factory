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
commit `4153c2b` de `main`. El inventario se lee de ese commit con git, sin
cambiar de rama ni copiar archivos:

```sh
git rev-parse 4153c2bb686a551cb293e295430aee38fd3857cb^{tree}
# 694f953b7941adc4398565bbdd0de222b71a7968
git ls-tree -r -l 4153c2b
# 53 blobs; la suma de tamaños es 3435304
```

Para leer un archivo: `git show 4153c2b:<ruta>`.

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
- Historias posteriores de #375, entre ellas #377 (AIF-002) y el walking slice de #379.

## Inventario v2

Los 53 archivos de `4153c2b`, en el orden de `git ls-tree -r`. Cada fila apunta a
uno o más identificadores de la matriz de paridad, o lleva la etiqueta
`sin comportamiento de runtime` con su motivo.

| Archivo(s) v2 | Mapeo |
|---|---|
| `.claude/agents/approver.md` | PAR-ENT-03, PAR-ENT-04 |
| `.claude/agents/builder.md` | PAR-ENT-01, PAR-ENT-02, PAR-COR-04, PAR-GATE-04, PAR-REA-03 |
| `.claude/agents/intake.md` | PAR-INT-03 |
| `.claude/agents/reviewer-code.md` | PAR-REV-01, PAR-REV-03 |
| `.claude/agents/reviewer-security.md` | PAR-REV-01, PAR-REV-03 |
| `.claude/agents/reviewer-ui.md` | PAR-REV-01, PAR-REV-03 |
| `.claude/agents/reviewer-ux.md` | PAR-REV-01, PAR-REV-03 |
| `.claude/agents/spec-writer.md` | PAR-ART-01 |
| `.claude/skills/factory/SKILL.md` | PAR-INT-05, PAR-REV-01, PAR-COR-01, PAR-COR-03, PAR-REA-01, PAR-REA-03 |
| `DESIGN.md` | sin comportamiento de runtime: prosa que describe el pipeline; los comportamientos que cita los poseen las filas de `factory/bin/*`, agentes y `SKILL.md` |
| `INSTALL.md` | sin comportamiento de runtime: instrucciones de instalación para una persona |
| `LICENSE` | sin comportamiento de runtime: licencia GPLv3 |
| `README.md` | sin comportamiento de runtime: presentación del proyecto |
| `docs/assets/software-factories-guide-promo.png` | sin comportamiento de runtime: imagen promocional |
| `docs/assets/zazencodes-banner.png` | sin comportamiento de runtime: imagen de cabecera |
| `examples/jobs/001-add-a-subtract-function/build.md`, `examples/jobs/001-add-a-subtract-function/decision.md`, `examples/jobs/001-add-a-subtract-function/intake.md`, `examples/jobs/001-add-a-subtract-function/issue.md`, `examples/jobs/001-add-a-subtract-function/job.json`, `examples/jobs/001-add-a-subtract-function/round-1/checks.md`, `examples/jobs/001-add-a-subtract-function/round-1/merge-checks.md`, `examples/jobs/001-add-a-subtract-function/round-1/review-code.md`, `examples/jobs/001-add-a-subtract-function/spec.md` | sin comportamiento de runtime: fixture de un job aprobado y mergeado en una ronda; sirve de dato de prueba para PAR-ART-01 y PAR-ART-02 |
| `examples/jobs/002-show-a-history-of/build.md`, `examples/jobs/002-show-a-history-of/decision.md`, `examples/jobs/002-show-a-history-of/intake.md`, `examples/jobs/002-show-a-history-of/issue.md`, `examples/jobs/002-show-a-history-of/job.json`, `examples/jobs/002-show-a-history-of/round-1/checks.md`, `examples/jobs/002-show-a-history-of/round-1/merge-checks.md`, `examples/jobs/002-show-a-history-of/round-1/review-code.md`, `examples/jobs/002-show-a-history-of/round-1/review-ui.md`, `examples/jobs/002-show-a-history-of/round-1/review-ux.md`, `examples/jobs/002-show-a-history-of/spec.md` | sin comportamiento de runtime: fixture de un job con superficie `ui` y tres reviewers; dato de prueba para PAR-INT-04 y PAR-REV-01 |
| `examples/jobs/003-add-multiply-divide-power/build.md`, `examples/jobs/003-add-multiply-divide-power/decision.md`, `examples/jobs/003-add-multiply-divide-power/intake.md`, `examples/jobs/003-add-multiply-divide-power/issue.md`, `examples/jobs/003-add-multiply-divide-power/job.json`, `examples/jobs/003-add-multiply-divide-power/round-1/checks.md`, `examples/jobs/003-add-multiply-divide-power/round-2/checks.md`, `examples/jobs/003-add-multiply-divide-power/round-2/review-code.md`, `examples/jobs/003-add-multiply-divide-power/spec.md` | sin comportamiento de runtime: fixture de un gate fallido en la ronda 1 corregido en la ronda 2; dato de prueba para PAR-GATE-03 y PAR-COR-01 |
| `examples/jobs/005-add-a-crypto-miner/intake.md`, `examples/jobs/005-add-a-crypto-miner/issue.md`, `examples/jobs/005-add-a-crypto-miner/job.json` | sin comportamiento de runtime: fixture de un issue rechazado en intake; dato de prueba para PAR-INT-03 y PAR-EST-01 |
| `factory/bin/gate` | PAR-GATE-01, PAR-GATE-02, PAR-GATE-04, PAR-ENT-03 |
| `factory/bin/job` | PAR-INT-01, PAR-INT-02, PAR-INT-05, PAR-INT-06, PAR-EST-01 a PAR-EST-04, PAR-ART-02 a PAR-ART-05, PAR-GATE-03, PAR-REV-02, PAR-COR-01 a PAR-COR-05, PAR-REA-02, PAR-REA-04, PAR-ENT-01, PAR-ENT-03, PAR-ENT-04 |
| `factory/bin/route` | PAR-INT-04 |
| `factory/checks.sh` | PAR-GATE-01 |
| `factory/config.json` | PAR-INT-04, PAR-EST-04, PAR-GATE-02, PAR-COR-02, PAR-ART-04, PAR-ENT-03, PAR-REV-03 |
| `factory/dashboard.html` | PAR-ART-05 (solo el modelo de lectura; la UI HTML queda fuera de alcance) |

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
| PAR-EST-03 | estados | `job` es el único escritor de `job.json` y toma un lock por job mientras escribe (`factory/bin/job`, `load`/`save`). | `internal/store` | Dos escrituras concurrentes se serializan; ninguna se pierde. |
| PAR-EST-04 | estados | Un job ocupa slot si no es terminal ni de espera; `create` y `needs-info → intake` se rechazan sin slot libre (`require_slot`, `concurrency` en `factory/config.json`). | `internal/queue` | Con `concurrency` agotada, crear o reanudar se rechaza con el mensaje de v2. |
| PAR-ART-01 | artefactos | Cada job vive en `factory/jobs/<id>/`: `job.json`, `issue.md`, `intake.md`, `spec.md`, `build.md`, `round-<N>/checks.md`, `round-<N>/review-<r>.md`, `decision.md`, `feedback-<k>.md` (`SKILL.md`, `.claude/agents/spec-writer.md`). | `internal/store` (layout de job) | Recorrer un fixture de `examples/jobs/` y resolver cada ruta esperada. |
| PAR-ART-02 | artefactos | Contratos de primera línea: `STATUS: READY` o `STATUS: BLOCKED`, `VERDICT: PASS` o `VERDICT: CHANGES`, `APPROVE` o `ESCALATE`, `GATE: PASS` o `GATE: FAIL`; cabeceras `CLAVE: valor` y sección `## Summary` (`factory/bin/job`, `header`/`first_line`). | `internal/artifacts` | Parseo de fixtures; una primera línea fuera de contrato da `invalid`. |
| PAR-ART-03 | artefactos | `job sync` deriva `intake`, `gate` y `reviews` de los archivos, nunca de memoria (`factory/bin/job`, `sync`). | `internal/jobs` (sincronización) | Modificar un archivo y sincronizar actualiza solo el campo derivado. |
| PAR-ART-04 | artefactos | El costo por job suma las transcripciones de subagentes que nombran su `Job folder:` y las valora con `prices_usd_per_mtok` (`factory/bin/job`, `cost_usd`; `factory/config.json`). | `internal/cost` | Transcripciones de fixture dan el costo esperado; las de otro job no cuentan. |
| PAR-ART-05 | artefactos | `job summary` y el tablero leen `job.json` y los archivos de ronda para mostrar etapa, ronda, PR, costo y tiempo (`factory/bin/job`, `cmd_summary`; `factory/dashboard.html`). | `internal/jobs` (modelo de lectura) | Golden de la tabla de resumen con jobs de fixture. |
| PAR-GATE-01 | gate | El gate corre el `factory/checks.sh` del checkout principal dentro del worktree y escribe `round-N/checks.md` (con `GATE` y `COMMIT`) y `checks.log`, con las últimas 80 líneas (`factory/bin/gate`, `factory/checks.sh`). | `internal/gate` | Un checks que falla da `GATE: FAIL` con el commit actual; un checks editado en la rama no se usa. |
| PAR-GATE-02 | gate | Falla si la rama modifica o borra un archivo de `protected_paths` (fnmatch, `*` cruza `/`); añadir archivos es válido (`factory/bin/gate`, `factory/config.json`). | `internal/gate` (rutas protegidas) | Casos M, D, R y A sobre rutas protegidas; solo A pasa. |
| PAR-GATE-03 | gate | `gate → review` exige `COMMIT` igual a HEAD y `GATE: PASS`; con FAIL vuelve a `build`, o a `needs-human` en la última ronda (`factory/bin/job`, `advance`). | `internal/state` (guardas del gate) | Un `checks.md` de otro commit se rechaza; FAIL en la última ronda va a `needs-human`. |
| PAR-GATE-04 | gate | `gate --hook` corre los checks al parar el builder y sale con 2 si fallan, como aviso; el gate es quien decide (`factory/bin/gate`, `.claude/agents/builder.md`). | `internal/gate` (hook) | Hook con checks rojos sale 2; con `stop_hook_active` sale 0. |
| PAR-REV-01 | reviewers | Los reviewers enrutados corren en paralelo; se espera a todos y cada uno escribe `round-N/review-<r>.md` (`SKILL.md`, `.claude/agents/reviewer-*.md`). | `internal/review` | `review → approve` se rechaza mientras falte algún `review-<r>.md`. |
| PAR-REV-02 | reviewers | Solo se aprueba si todas las reviews dicen PASS; una primera línea que no es veredicto bloquea el avance (`factory/bin/job`, `check_reviews_done`). | `internal/review` | Una CHANGES impide `approve`; un veredicto inválido se rechaza. |
| PAR-REV-03 | reviewers | Cuatro lentes (`code`, `security`, `ui`, `ux`) con su modelo y puertos `5100 + 10n`, `5101 + 10n`, `5102 + 10n` (`.claude/agents/reviewer-*.md`, `factory/config.json`). | `internal/roles` | Registro de roles y cálculo de puertos para un id de job dado. |
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
| EXT-MCP-01 | MCP | Servidor `aifactory`, transporte `stdio`, local; cliente Claude Code con Gentle-AI v4.5.0. |
| EXT-MCP-02 | MCP | Contrato completo de nueve tools: `list_jobs`, `create_job`, `get_job`, `record_artifact`, `advance_job`, `run_checks`, `provide_feedback`, `deliver_job`, `stop_job`. |
| EXT-MCP-03 | MCP | Walking slice de #379: `create_job`, `get_job` y `list_jobs`. |
| EXT-MCP-04 | MCP | Schemas estrictos; un `request_id` repetido es idempotente. |
| EXT-PDF-01 | PDF, página 9 | Tras CI y review, la entrega es una decisión humana. Cuando los agentes abren más PR de los que una persona puede leer, el tablero los ordena por riesgo en P1/P2/P3, los separa en Now/Next y muestra en cada tarjeta una razón breve y la próxima acción (por ejemplo corregir CI o fusionar). |
| EXT-PDF-02 | PDF, página 10 | La vista empieza con un resumen del cambio y luego divide el PR en capas nombradas por propósito, en orden de lectura, para explicar el porqué que el diff solo no muestra. |
| EXT-PDF-03 | PDF, página 10 | La revisión hace visibles los riesgos de seguridad y hallazgos aplicables. En v0.1 se usan únicamente Gitleaks, Semgrep y OSV-Scanner cuando estén configurados. Una herramienta ausente o fallida nunca se presenta como PASS. |
| EXT-PDF-04 | PDF, referencia visual | Página 10 para detalle y recorrido; página 9 para el tablero Now/Next. No se inventan otros módulos de CodeRabbit. |

Fuente de las filas EXT-PDF: *Build Your Own Software Factory in 5 Steps*, Daniel
Moka, copia de 14 páginas del 17/11/2026, tal como la describió el autor en #376.

## Gentle y contrato MCP

- **Versión congelada:** Gentle-AI `v4.5.0`, release oficial:
  <https://github.com/Gentleman-Programming/gentle-ai/releases/tag/v4.5.0>.
- **Verificación local:** `gentle-ai version` (también valen `gentle-ai --version`
  y `gentle-ai -v`). La evidencia guarda la salida real. Si no informa `v4.5.0`,
  la prueba se bloquea y se informa la diferencia, sin asumir compatibilidad.
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
