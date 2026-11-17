# aifactory · Go shell

This branch is intentionally reduced to a buildable Go shell for **aifactory v0.1**.

It contains no product behavior from the former `factoryd` implementation. The old ACP, SQLite, scheduler, outbox/event architecture, role system, pipelines, demos and historical backlog are not part of the new product.

The canonical functional baseline is the Software Factory v2 on `main`. The new implementation is tracked by [epic #375](https://github.com/getsyntegrity/software-factory/issues/375) and will be built incrementally with Gentle, MCP and ODD/XP.

Frozen sources, contracts and the v2 parity matrix: [docs/aif-001-fuentes-y-paridad.md](docs/aif-001-fuentes-y-paridad.md).

The complete former implementation remains recoverable at:

`backup/arch-factoryd-go-full-20261117-4b56aac3`

## Check

```sh
make check
./bin/aifactory
```
