---
description: hawk-sdk-go — Go SDK build and test conventions.
globs: "*.go,*.yaml"
alwaysApply: false
---

# hawk-sdk-go Conventions

Official Go client for the Hawk daemon API.

## Development workflow

When starting any new work (feature, fix, refactor, chore), always create a feature branch from `main` first. Never commit directly to `main`. Use branch naming conventions like `feat/<description>`, `fix/<description>`, or `chore/<description>`. Open a PR, ensure CI is green, then merge.

## Build & Test

```bash
go build ./...                    # Build library
go test ./...                     # Run tests
go test -race ./...               # Race detector
go test -coverprofile=c.out ./...  # Coverage
go vet ./...                      # Static analysis
gofumpt -w .                      # Format
go mod tidy                       # Tidy modules
```

## Design Principles

- Dependency-free: zero third-party runtime imports, pure Go standard library
- `oapi-codegen` is a build-time tool only (see `codegen_tools.go`)
- Local-only: designed for developers running Hawk locally

## Ecosystem Boundaries

- Consumer of Hawk public APIs and `hawk-core-contracts`
- Do not import support engine repos (`eyrie`, `yaad`, `tok`, `trace`, `sight`, `inspect`)

For full hawk-eco extension guidelines, see [hawk/AGENTS.md](https://github.com/GrayCodeAI/hawk/blob/main/AGENTS.md).

<!-- gitnexus:start -->
## GitNexus — Code Intelligence

This project is indexed by GitNexus as **hawk-sdk-go** (602 symbols, 1548 relationships, 48 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> Index stale? Run `node .gitnexus/run.cjs analyze` from the project root — it auto-selects an available runner. No `.gitnexus/run.cjs` yet? `npx gitnexus analyze` (npm 11 crash → `npm i -g gitnexus`; #1939).

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows. For regression review, compare against the default branch: `detect_changes({scope: "compare", base_ref: "main"})`.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `query({search_query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `context({name: "symbolName"})`.
- For security review, `explain({target: "fileOrSymbol"})` lists taint findings (source→sink flows; needs `analyze --pdg`).

## Never Do

- NEVER edit a function, class, or method without first running `impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `rename` which understands the call graph.
- NEVER commit changes without running `detect_changes()` to check affected scope.

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/hawk-sdk-go/context` | Codebase overview, check index freshness |
| `gitnexus://repo/hawk-sdk-go/clusters` | All functional areas |
| `gitnexus://repo/hawk-sdk-go/processes` | All execution flows |
| `gitnexus://repo/hawk-sdk-go/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->
