package hawksdk

import (
	_ "embed"
	"strings"
)

// versionFile is the canonical version, read at compile time from the VERSION
// file at the repo root. The VERSION file is the single source of truth used
// by release tooling (release-please, goreleaser, CI).
//
//go:embed VERSION
var versionFile string

// Version of the hawk-sdk-go library. Used in the User-Agent header on
// outbound HTTP requests so a misbehaving SDK can be identified by daemon
// logs and operators.
//
// Source of truth: the VERSION file at the repo root. Do not edit this
// constant directly — bump VERSION instead (or let release-please do it).
var Version = strings.TrimSpace(versionFile)

// userAgent returns the User-Agent string for outbound HTTP requests.
func userAgent() string { return "hawk-sdk-go/" + Version }
