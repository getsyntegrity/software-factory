# AIF-004 · Conversación de prueba con Claude Code y Gentle

Guía de la historia [#379](https://github.com/getsyntegrity/software-factory/issues/379).
Recorre, sin dashboard, los pasos 2 a 7 de la
[prueba de fase 0](aif-001-fuentes-y-paridad.md#prueba-de-fase-0): Claude Code con
Gentle habla con `aifactory mcp` y crea, muestra y lista un trabajo.

El servidor solo expone `create_job`, `get_job` y `list_jobs`. No ejecuta nada más:
ni GitHub, ni agentes, ni checks, ni merges. Tampoco necesita MCP ni RPC de Gentle;
Claude Code lo arranca como proceso hijo y le habla por stdin y stdout.

Los trabajos viven en `.aifactory/jobs/` bajo el directorio desde el que arranca el
servidor (la raíz del repo), un archivo `<id>.json` por trabajo. Git ignora esa
carpeta. Sobreviven a reiniciar el servidor y Claude Code.

## Requisitos

1. `make build` en la raíz del repo: deja el binario en `bin/aifactory`.
2. Abrir Claude Code con Gentle desde la raíz del repo. El archivo versionado
   `.mcp.json` declara el servidor `aifactory` (`./bin/aifactory mcp`, transporte
   `stdio`).
3. Cuando Claude Code pregunte, aprobar el servidor de proyecto `aifactory`.

## Conversación

| Paso de fase 0 | El usuario dice | Qué pasa |
|---|---|---|
| 2 | "¿Qué servidores MCP tenés?" (o `/mcp`) | Aparece `aifactory`, `stdio`, conectado. |
| 3 | "¿Qué tools tiene aifactory?" | Aparecen `create_job`, `get_job` y `list_jobs`. |
| 4 | "Creá un trabajo" con `request_id` `fase0-001` | Claude llama a `create_job` con `{"request_id":"fase0-001"}`. Devuelve un `id` `job-` más 16 caracteres hex, con `state` `intake` y `next_operation` `run_intake`. Aparece `.aifactory/jobs/<id>.json`. |
| 5 | "Mostrame el trabajo" | Claude llama a `get_job` con ese `id`. La respuesta muestra `state` `intake` y `next_operation` `run_intake`. Funciona también tras reiniciar Claude Code. |
| 6 | "Listá los trabajos" | Claude llama a `list_jobs`. Devuelve `{"jobs":[...]}` con exactamente un trabajo, el del paso 4. |
| 7 | "Creá un trabajo" otra vez, con el mismo `request_id` `fase0-001` | `create_job` devuelve el mismo `id` del paso 4 y `list_jobs` sigue mostrando un solo trabajo: no se duplica. |

Una entrada inválida (un campo que falta, de otro tipo o desconocido, o un `id` que
no existe) devuelve un resultado con `isError: true` que nombra el problema; el
servidor sigue respondiendo.

## Prueba sin Claude Code

Desde un directorio vacío, con la ruta al binario:

```sh
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"create_job","arguments":{"request_id":"fase0-001"}}}' \
  | /ruta/al/repo/bin/aifactory mcp
```

Cada línea de stdout es una respuesta JSON-RPC 2.0.
