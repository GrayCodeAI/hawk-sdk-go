# Hawk daemon endpoint support

`openapi.yaml` is an exact snapshot of Hawk's authoritative daemon contract.
The SDK supports chat, health, sessions, session messages, and statistics.

The following operational/product-specific endpoints are intentionally not
wrapped in v0.1:

- `GET /v1/ready` — daemon deployment readiness probe;
- `POST /v1/review` — asynchronous review orchestration;
- `GET /v1/review/status` — review worker status.

They remain available through the generic HTTP transport and must be classified
here before the server contract can add another path.
