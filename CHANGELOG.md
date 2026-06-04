# Changelog

All notable changes to `hawk-sdk-go` are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **`const Version = "0.1.0"`** in `version.go`, exposed as the package-level
  source of truth for the SDK version. Aligns hawk-sdk-go with the rest of
  the hawk-eco ecosystem (`hawk`, `tok`, `eyrie`, `yaad`, `trace`, `sight`,
  `inspect`).
- **`User-Agent: hawk-sdk-go/<Version>` header** on every outbound HTTP
  request — `Health`, `Chat`, `ChatStream`, `Sessions`, `Session`,
  `Messages`, `DeleteSession`, `Stats`. Lets daemon operators identify
  SDK clients in logs and reject misbehaving versions cleanly.
- **OSS standard files** (this is the first PR to add them):
  - `README.md` — install, usage, API surface, versioning, contributing.
  - `LICENSE` — MIT.
  - `CONTRIBUTING.md` — quick start, branch flow (this repo branches
    from `main`), conventional commits, code standards, testing.
  - `SECURITY.md` — vulnerability reporting via GitHub Security
    Advisories.
  - `CODE_OF_CONDUCT.md` — Contributor Covenant 2.1.
  - `.gitignore` — Go artifacts, IDE files, OS files, coverage output.
  - `.gitattributes` — LF normalization, binary detection, GitHub
    linguist hint to collapse `go.sum` in PR diffs.
  - `.github/workflows/ci.yml` — race tests + coverage upload + lint
    (golangci-lint v2) + multi-platform build matrix
    (linux/darwin/windows × amd64/arm64) + govulncheck.
  - `.github/dependabot.yml` — weekly `gomod` + `github-actions`
    updates.
  - `.github/PULL_REQUEST_TEMPLATE.md` — Summary / Changes / API impact
    / Daemon compatibility / Testing / Checklist.
  - `.github/ISSUE_TEMPLATE/bug_report.yml` — structured bug report.
  - `.github/ISSUE_TEMPLATE/feature_request.yml` — feature request
    with solo-dev fit checks.
  - `.github/ISSUE_TEMPLATE/config.yml` — routes security to
    advisories, questions to discussions, blocks blank issues.

### Changed
- `gofmt -w` on `client.go` and `retry.go` to remove pre-existing
  trailing-blank-line drift in files I touched.
- Test fixture in `client_test.go` updated to mock daemon version
  `"0.1.0"` (was `"0.3.0"`) for ecosystem consistency. The test does
  not assert on this value; the change is cosmetic.

### Production-hardening pass already on this branch (commit `1028420`)
- Strict `golangci-lint` v2 config with `errcheck`, `staticcheck`,
  `gocritic` (diagnostic + performance), `unused`, `ineffassign`,
  `misspell`, `noctx`, `bodyclose`, `unconvert`, `whitespace`.
- All `errcheck` issues fixed (`resp.Body.Close` deferred and
  error-path closes).
- Initial `Makefile`, `.editorconfig`.

## [0.0.1] — 2026-05-13

### Added
- Initial Go SDK for the hawk daemon API (commits `66e1e1a`, `d04480f`):
  - `Client` with `Health`, `Chat`, `ChatStream`, `Sessions`,
    `Session`, `Messages`, `DeleteSession`, `Stats`.
  - Typed errors with categories (`ErrCategoryRateLimited`,
    `ErrCategoryUnauthorized`, `ErrCategoryNotFound`,
    `ErrCategoryServer`, `ErrCategoryTransport`).
  - Retry with exponential backoff, full jitter, and `Retry-After`
    honouring on 429.
  - SSE stream helpers.
  - `Agent`, `Tool`, `Workflow` orchestration primitives.
