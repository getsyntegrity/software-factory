# aifactory · Go shell

This branch is intentionally reduced to a buildable Go shell for **aifactory v0.1**.

It contains no product behavior from the former `factoryd` implementation. The old ACP, SQLite, scheduler, outbox/event architecture, role system, pipelines, demos and historical backlog are not part of the new product.

The canonical functional baseline is the Software Factory v2 on `main`. The new implementation is tracked by [epic #375](https://github.com/getsyntegrity/software-factory/issues/375) and will be built incrementally with Gentle, MCP and ODD/XP.

Frozen sources, contracts and the v2 parity matrix: [docs/aif-001-fuentes-y-paridad.md](docs/aif-001-fuentes-y-paridad.md).

The complete former implementation remains recoverable at:

`backup/arch-factoryd-go-full-20261117-4b56aac3`

## Usage

```sh
./bin/aifactory --help
./bin/aifactory <command> --help
```

Commands: `mcp`, `serve`, `status` and `import-v2`. `mcp` serves the tools `create_job`, `get_job` and `list_jobs` over MCP on stdio, with jobs stored in `.aifactory/jobs/`; see [docs/aif-004-conversacion-de-prueba.md](docs/aif-004-conversacion-de-prueba.md). The other subcommands are still placeholders: they print their help and do nothing else.

Attribution of the v2 origin: [NOTICE.md](NOTICE.md).

## Check

```sh
make check
./bin/aifactory
```

Testing rules (unit tests with `go-specs` and spy mocks): [skills/go-unit-testing/SKILL.md](skills/go-unit-testing/SKILL.md).
