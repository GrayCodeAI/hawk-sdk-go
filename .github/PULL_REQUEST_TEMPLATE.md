<!--
  Thanks for your contribution! Please fill out this template so reviewers can
  understand the change quickly. Anything that does not apply can be left in
  place; do not delete unanswered sections — write "n/a".
-->

## Summary

<!--
  One paragraph describing what this PR does and why. Link the related
  issue(s) with `Fixes #N` or `Refs #N` if applicable.
-->

## Changes

<!--
  Bullet list of what changed, grouped by area (client, retry, stream,
  errors, agent, tools, workflow, version, CI, docs). Reviewers should be
  able to skim this and know what to look at first.
-->

-

## API impact

<!--
  Did you add, remove, rename, or change the signature of any exported
  symbol? List them here. If yes, confirm whether this is a breaking
  change and bump the version accordingly in `version.go` + `CHANGELOG.md`.
  If no exported surface changed, write "n/a".
-->

## Daemon compatibility

<!--
  This SDK targets the hawk daemon `v1` API. Did you change endpoints,
  request/response shapes, headers, or status-code handling?

  - Which daemon versions did you test against (commit SHA / tag)?
  - Is the change wire-compatible with the latest released daemon?
  - If not, link the corresponding daemon PR.
-->

## Testing

<!--
  Describe how you tested. Paste output of `make test` and `make lint`. If
  you added new tests, list them.
-->

```text
$ make test
...
$ make lint
...
```

## Checklist

- [ ] Commits follow [Conventional Commits](https://www.conventionalcommits.org/)
      (`feat:`, `fix:`, `perf:`, `refactor:`, `docs:`, `test:`, etc.)
- [ ] `make build` (or `go build ./...`) passes
- [ ] `make lint` passes (no new lint findings, no `nolint:…` without justification)
- [ ] `make test` passes locally with `-race` enabled
- [ ] New or changed code has tests (table-driven where appropriate)
- [ ] Public APIs have godoc comments and runnable examples where helpful
- [ ] `CHANGELOG.md` updated under `## [Unreleased]` if user-visible
- [ ] `version.go` bumped if this is a release-eligible change
- [ ] Every new outbound HTTP request sets `User-Agent: hawk-sdk-go/<Version>`
      via the `userAgent()` helper
- [ ] No secrets, tokens, or PII added to the repo
- [ ] No `Co-authored-by:` trailers (this is solo-developer work)
