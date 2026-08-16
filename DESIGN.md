# Design

## Source of truth
- Status: Active
- Last refreshed: 2026-08-17
- Primary product surfaces: Web Chat, Web Admin, Wails Desktop
- Evidence reviewed: `README.md`, `web/src/App.tsx`, `web/src/AdminWorkspace.tsx`, `web/src/styles.css`, `pkg/config`, `pkg/apis/admin_config_handler.go`, `samples/*.json`

## Brand
- Personality: calm, capable, direct, privacy-conscious.
- Trust signals: explicit connection state, clear model/provider identity, masked secrets, actionable errors, visible local/remote mode.
- Avoid: OpenAI trademarks or copied assets, decorative gradients that reduce readability, hidden network activity, and exposing raw provider credentials.

## Product goals
- Goals: provide an OpenAI-compatible gateway, a browser chat client, an administration console, and a convenient Wails desktop client from one shared product core.
- Goals: keep server deployment simple; the Web UI must be embedded in the Go executable so the server can run from one file.
- Goals: preserve existing JSON/YAML configuration as an import/export and compatibility format while moving runtime state behind a versioned repository.
- Non-goals: pixel-for-pixel copying of ChatGPT, billing, a public SaaS control plane, or an immediate microservice split.
- Success signals: a new user can configure one provider and complete a streamed chat without editing source files; server releases run without a separate static directory; desktop and Web share the same UI code.

## Personas and jobs
- Primary personas: individual self-hosters, developers integrating OpenAI-compatible APIs, and small-team administrators.
- User jobs: configure providers safely, test model health, chat from a browser or desktop, rotate access keys, and diagnose failures without exposing prompts or secrets.
- Key contexts of use: a local desktop, a LAN-hosted server, and a small public server behind HTTPS.

## Information architecture
- Primary navigation: Chat and Admin are separate top-level surfaces.
- Core routes/screens: `/` configuration workspace, `/chat` chat, and `/admin` as a compatibility route for configuration. Configuration sections cover system settings, Providers, access, network, source, and real-time logs.
- Delivery status: the chat shell, bounded local conversation history, three client protocols, SQLite-backed configuration snapshots, validation/save, real-time logs, and the visual configuration workspace are implemented. Configuration Source remains available as an expert escape hatch rather than the primary editing surface.
- Content hierarchy: conversation content first in Chat; readiness and validation first in Admin, then providers/models, access control, network settings, and revision history.
- First-run flow: when no models are configured, the empty Chat state links directly to Admin. Local users enter immediately; remote users paste the temporary bootstrap token printed at startup. The administrator then sets a permanent gateway key, selects a Provider type, enters endpoint/models/credentials, validates, publishes, and returns to Chat without editing JSON or local files.

## Design principles
- One core, multiple shells: Web and Wails use the same React application and API contracts.
- Safe by default: untrusted model output is rendered as text or sanitized Markdown; secrets are masked; privileged actions are explicit.
- Progressive complexity: common system, Provider, access-key, and proxy controls use typed forms; arbitrary provider-specific fields, redirects, maps, and bulk migration remain available in Configuration Source with JSON/YAML import and export.
- Preserve user context: streaming, reconnecting, errors, and model switches must not silently discard drafts or conversation history.
- Tradeoffs: prefer a small, understandable component system over a large UI framework; prefer a modular monolith over early service decomposition.

## Visual language
- Color: neutral gray surfaces with a restrained green accent; semantic colors are reserved for status and destructive actions.
- Typography: system sans-serif stack for UI; system monospace for code and identifiers.
- Spacing/layout rhythm: 4px base scale, 16px default control gap, 24px section spacing.
- Shape/radius/elevation: 8px primary panel radii, subtle borders, minimal shadows, no decorative glass effects on content surfaces.
- Motion: 120-200ms transitions; streaming content does not animate per token.
- Imagery/iconography: simple line icons with text labels for ambiguous actions; no copied OpenAI marks.

## Components
- Existing components to reuse: shared React App shell, AdminWorkspace forms, Markdown renderer, stream display helpers, and local conversation store.
- Main components: AppShell, ConversationSidebar, ChatTimeline, Message, Composer, ModelPicker, ConnectionStatus, SettingsPanel, AdminShell, AdminSectionNav, SystemForm, ProviderCard, CredentialEditor, AccessKeyEditor, ProxyForm, ConfigurationSourceEditor, LiveLogsPanel, ValidationSummary, and Toast.
- Variants and states: desktop/mobile sidebar, user/assistant/system messages, streaming/stopped/error responses, local/remote connection states, remote bootstrap locked/unlocked, Provider enabled/disabled, secret preserved/replaced, visual/advanced edit modes, dirty/validated/publishing configuration states.
- Token/component ownership: CSS custom properties in the shared Web application; components consume tokens and must not introduce page-local color systems.

## Accessibility
- Target standard: WCAG 2.2 AA.
- Keyboard/focus behavior: all actions keyboard reachable; visible focus rings; composer shortcuts documented; dialogs trap and restore focus.
- Contrast/readability: AA contrast for text and controls; code blocks remain readable in both themes.
- Screen-reader semantics: semantic landmarks, live regions for status rather than every streamed token, labels for icon buttons.
- Reduced motion and sensory considerations: honor `prefers-reduced-motion`; status is never conveyed by color alone.

## Responsive behavior
- Supported breakpoints/devices: 360px mobile Web through wide desktop and native Wails windows.
- Layout adaptations: sidebar becomes a drawer below 768px; message width expands on small screens; admin tables become cards where necessary.
- Touch/hover differences: minimum 44px touch targets; hover-only affordances also appear on focus.

## Interaction states
- Loading: clear connection and streaming indicators; local conversation history loads synchronously from bounded browser storage.
- Empty: guided first-message screen with provider/model readiness information.
- Error: concise user message, expandable technical detail, retry action, request ID when available.
- Success: non-blocking toast for saved settings; connection and provider tests report latency.
- Configuration saving: edits remain a local draft; validation is explicit; saving creates and activates an immutable internal revision, reports restart-required fields, and warns when the active API key changes. Switching to Configuration Source must serialize the current visual draft without losing unknown fields. Revision list/activation remains an API-level compatibility feature rather than a primary UI workflow.
- First-run bootstrap: local same-origin Admin access may initialize directly; remote Admin requires a high-entropy temporary token from the startup log or deployment environment. The unlock screen must direct the operator to set and publish a permanent `api_key`; bootstrap must never mean unauthenticated remote access.
- Disabled: explain why an action is unavailable.
- Offline/slow network: preserve drafts, expose reconnect, and allow stopping a pending stream.

## Content voice
- Tone: concise, neutral, helpful.
- Terminology: Provider, Model, Access Key, Conversation, Local mode, Remote mode.
- Microcopy rules: name the failed operation and next action; never print or echo a complete secret.

## Implementation constraints
- Framework/styling system: React, TypeScript, Vite, and repository-owned CSS tokens; Wails is the desktop shell.
- Design-token constraints: one token source shared by Chat and Admin; light/dark themes use the same semantic token names.
- Performance constraints: initial compressed Web assets target under 500 KB excluding optional syntax highlighters; streaming updates are batched to avoid per-token layout thrash.
- Compatibility constraints: server Web assets are embedded with Go `embed`; the server remains runnable as a single executable; existing `/v1` API behavior remains covered by contract tests.
- Storage constraints: SQLite is the default configuration repository; JSON/YAML remain import/export formats; active revisions are immutable and publishing must not mutate a revision in place. Service entries require stable IDs so reordering or editing a Provider does not reset runtime identity.
- Test/screenshot expectations: unit tests for stream parsing and state reducers, browser E2E for the primary chat flow, and desktop smoke tests for startup and local/remote mode.

## Open questions
- [ ] Replace the temporary letter mark with an original application icon in a later signed-installer release / owner: maintainer / impact: release assets and application identifiers.
- [ ] Decide whether multi-user or cross-device conversation synchronization belongs in a later release / owner: maintainer / impact: auth, privacy, and storage schema.
- [ ] Decide the initial signed-update hosting channel for Wails releases / owner: maintainer / impact: release automation.
- [ ] Decide the production secret-encryption key source for server deployments; desktop should use the OS keychain / owner: maintainer / impact: encrypted provider credentials at rest.
