# AGENTS.md

aifactory is a Go module; see [README.md](README.md) for what it does, how to use it and how to run `make check`.

Before writing code, load the matching skill from the table below.

## How to Use

1. Find the row whose Trigger matches your task.
2. Read the `SKILL.md` at that row's Path.
3. Follow its rules.
4. More than one skill can apply to the same task; load every one that matches.

## Skills

| Skill | Trigger | Path |
|-------|---------|------|
| `go-unit-testing` | Writing or modifying Go tests (`*_test.go`) | [skills/go-unit-testing/SKILL.md](skills/go-unit-testing/SKILL.md) |
