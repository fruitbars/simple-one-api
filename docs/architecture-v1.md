# v1 Upgrade Architecture

## Accepted decisions

1. The server remains easy to self-host and embeds the production Web bundle into the Go executable.
2. Wails is the desktop shell and consumes the same React application as the Web server.
3. Gateway, control-plane, chat, and provider concerns become explicit Go package boundaries, but may ship in one process.
4. Existing JSON/YAML configuration remains supported for import/export and migration.
5. Runtime configuration uses immutable, versioned snapshots with atomic publication.
6. SQLite is the default durable store; storage interfaces must not prevent a later PostgreSQL implementation.
7. Provider secrets are masked at API boundaries and redacted from normal admin responses. SQLite at-rest encryption is not implemented yet; the database currently relies on `0600` file permissions. Server master-key encryption and desktop OS keychain integration remain future work.
8. Current single-operator deployments use the primary gateway `api_key` to gate `/api/admin/*`. If `api_key` is empty, loopback requests can initialize directly and remote requests require a temporary bootstrap token from startup logs or `SIMPLE_ONE_API_BOOTSTRAP_TOKEN`. Separate admin sessions, CSRF protection, RBAC, and audit events remain future hardening.
9. Client compatibility includes OpenAI Chat Completions, OpenAI Responses, and Anthropic Messages. All three reuse the same provider routing, authentication, limits, proxy policy, cancellation, and error boundary.

## Delivery slices

### Slice 1: safe executable foundation

Status: delivered.

- Extract router construction from `main.go` so it can be tested.
- Embed the shared Web production bundle with `go:embed`.
- Replace the legacy runtime `./static` dependency.
- Establish React chat shell, streaming client, responsive layout, and design tokens.
- Remove high-risk secret logging and unsafe WebSocket/browser defaults.
- Add CI-ready Go and frontend tests.

### Slice 2: versioned configuration and admin API

Status: delivered for the single-operator control plane.

- Introduce stable provider/model identifiers.
- Add configuration repository, validation, revisions, atomic activation, and rollback.
- Add SQLite migrations and legacy config import/export.
- Add `/api/admin/config/draft`, validation, publish, revision list, and activation endpoints.
- Add a visual-first configuration editor with system, Provider, access, proxy, source, and real-time log sections, plus validation and save. Revision list and activation remain available through the Admin API for compatibility and operations.
- Keep secrets masked in drafts and admin summaries; restore unchanged masked values on publish.
- Track restart-required fields for `server_port`, `enable_web`, and `debug`/`log_level`.
- Defer authenticated admin sessions, CSRF protection, RBAC, audit events, and specialized controls for complex nested credentials. Typed Provider/model/access-key/proxy/system screens are now implemented; Configuration Source remains the lossless escape hatch.

### Slice 3: conversation domain and desktop

Status: partly delivered. The shared chat shell, Wails binding, and bounded local multi-conversation storage are delivered; cross-device synchronization, OS credential storage, and signed update distribution remain future work.

- Add conversations/messages, streamed run state, local switching/deletion, and bounded WebView/browser persistence. Delivered.
- Add local and remote connection modes.
- Bind the shared application services into Wails.
- Use OS credential storage for desktop secrets.
- Add signed installers, update verification, and platform smoke tests.

## Compatibility invariants

- Existing OpenAI-compatible routes keep their paths and response formats unless a versioned migration is documented.
- A normal server release does not require a `static` directory beside the executable.
- Provider credentials never appear in normal logs, browser responses, crash reports, or desktop diagnostics.
- SQLite currently stores configuration JSON without at-rest encryption; deployments that need stronger local secret protection must add an external storage control until master-key/keychain support lands.
- Web and desktop render model output as untrusted content.
- File configuration remains readable until a documented major-version removal.

## Verification gates

- `go test ./...`, `go vet ./...`, and `govulncheck ./...` pass.
- `pnpm test`, `pnpm typecheck`, and `pnpm build` pass for the shared Web application.
- A temporary directory containing only the built server executable can serve `/`, `/assets/*`, and `/v1/models`.
- Wails production build starts, loads the shared bundle, and does not require a separately copied Web directory.
- Contract tests cover authentication, streaming, CORS/origin handling, and legacy configuration parsing.
