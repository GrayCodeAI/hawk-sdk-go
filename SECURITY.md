# Security policy

## Supported versions

Only the latest minor version of `hawk-sdk-go` receives security updates.
The current supported version is the most recent `v0.x` release.

## Reporting a vulnerability

Please **do not** file a public GitHub issue for security vulnerabilities.

Instead, open a **private** GitHub Security Advisory:

> https://github.com/GrayCodeAI/hawk-sdk-go/security/advisories/new

Include, where possible:

- A clear description of the issue and impact.
- Steps to reproduce (a minimal Go snippet is ideal).
- The affected `hawk-sdk-go` version (`hawksdk.Version`) and Go version.
- The hawk daemon version you were targeting, if relevant.
- Any mitigations or patches you have already explored.

We aim to respond to advisories within **5 business days** and to release
a fix within **30 days** for high-severity issues.

## What counts as a security issue

Examples of in-scope issues:

- The SDK leaking secrets, API tokens, or session IDs into logs, errors,
  or metrics.
- The SDK accepting and forwarding data that bypasses daemon-side
  authentication, authorization, or rate limiting.
- TLS misuse — accepting untrusted certificates, downgrade to HTTP, or
  ignoring `https_proxy` rules.
- Memory-safety issues (panics on attacker-controlled input, unbounded
  allocations on stream or response).
- Path / URL handling that lets a malicious daemon URL escape the
  expected host (e.g. via redirects).

Out of scope:

- Issues in the hawk daemon itself — please report those at
  https://github.com/GrayCodeAI/hawk/security/advisories/new.
- Issues in third-party Go modules — please report those upstream.

## Disclosure

Once a fix is released, we will publish the advisory with credit to the
reporter (unless they request anonymity).
