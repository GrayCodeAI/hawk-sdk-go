# Contributing to hawk-sdk-go

Thanks for considering a contribution. `hawk-sdk-go` is the Go SDK for the
hawk daemon API. It is built for **solo developers** running their own hawk
daemon locally — small surface area, zero external dependencies, fast feedback.

## Quick start

```bash
git clone https://github.com/GrayCodeAI/hawk-sdk-go.git
cd hawk-sdk-go
make test       # race detector, -count=1, -timeout=60s
make lint       # golangci-lint v2
```

Go 1.26.1 is the targeted toolchain.

## Branch flow

This repo does **not** have a `dev` branch. Branch from `main`:

```bash
git checkout main
git pull origin main
git checkout -b feat/<short-description>
```

Open the PR against `main`. Do **not** push directly to `main`.

One PR per logical change. Do not mix unrelated changes in a single PR.

## Commit messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(client): add ChatStream backpressure
fix(retry): respect Retry-After when value is a date
perf(stream): re-use buffer in SSE parser
docs(readme): document agent orchestration
test(retry): add coverage for context cancellation during backoff
```

Allowed types: `feat`, `fix`, `perf`, `refactor`, `test`, `docs`, `chore`,
`build`, `ci`, `style`. Add a scope when it clarifies the change. Do not add
`Co-authored-by:` trailers — this is solo-developer work.

## Code standards

- `gofmt -l .` must be empty for files you touch.
- `go vet ./...` must be clean.
- `golangci-lint run ./...` must surface no new findings. The repo enables
  `errcheck`, `staticcheck`, `gocritic`, `unused`, `ineffassign`, `misspell`,
  `noctx`, `bodyclose`, `unconvert`, `whitespace`. Use `//nolint:<linter>`
  only with a one-line justification.
- Public types and functions must have godoc comments.
- Prefer table-driven tests with `t.Parallel()` where independent.
- Wrap errors with context: `fmt.Errorf("hawk-sdk: <action>: %w", err)`.
- Propagate `context.Context` everywhere; never call out to a network without it.
- Set `User-Agent: hawk-sdk-go/<Version>` on every new HTTP request via
  the `userAgent()` helper.

## Bumping the SDK version

The single source of truth is `version.go`. Bumping it automatically
updates the User-Agent header. When bumping:

1. Edit `version.go`.
2. Add a `## [X.Y.Z] — YYYY-MM-DD` entry at the top of `CHANGELOG.md`.
3. Tag the release: `git tag vX.Y.Z && git push origin vX.Y.Z`.

This SDK adheres to [SemVer](https://semver.org/spec/v2.0.0.html).
Breaking changes to the daemon API or the SDK surface bump the major
version. New SDK methods or daemon endpoints bump the minor. Bug fixes
and internal improvements bump the patch.

## Testing

```bash
make test          # full suite with race detector
make test-coverage # coverage totals
```

When adding a new client method, cover: success path, retryable error
(429 with `Retry-After`), non-retryable error, context cancellation,
and (for streaming) early `Close()` and EOF.

## Reporting bugs / requesting features

- Bug: open an issue using the bug-report template.
- Feature: open an issue using the feature-request template.
- Security: do **not** file a public issue. Use a GitHub Security Advisory
  per `SECURITY.md`.
