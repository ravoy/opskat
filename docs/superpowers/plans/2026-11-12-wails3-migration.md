# Wails v3 migration plan

Status: implemented in this working tree; macOS native behavior verified. This is
a dated migration snapshot, not a description of a published release.

## Scope and acceptance

Port the desktop shell from Wails v2.16.0 to v3.0.0, keeping the existing
bindings → service → repository boundaries and the standalone opsctl transport.
Preserve data-directory selection, portable mode, single-instance identity,
window preferences, extension assets, native dialogs, file drops, events,
quit confirmation and AI flush acknowledgements. Do not change stored schemas.

Acceptance requires generated v3 bindings, a production frontend and desktop
build, Go/frontend checks, and observable GUI behavior with isolated data.
Platform-specific packaging that cannot run on this host must be called out.

## Implementation sequence

1. **Establish the API and baseline.** Download the pinned module, inspect its
   implementation/examples and templates, run relevant existing checks, and
   inventory runtime operations and generated-model consumers.
2. **Port the desktop shell.** Register existing domain binders as v3 services;
   implement startup/shutdown and asynchronous close handling without blocking
   native dispatch. Preserve single-instance and portable WebView settings.
3. **Port runtime boundaries.** Replace v2 runtime imports with explicit v3
   application/window dependencies. Exercise event payloads, dialog cancellation,
   clipboard, file drops and shutdown through behavior tests where applicable.
4. **Regenerate frontend bindings.** Use the v3 generator; migrate imports,
   model construction, events and test mocks. Reuse existing hooks and channels.
   Generated files remain generated; do not hand-edit their contents.
5. **Update tooling.** Replace v2 CLI commands in Makefile, CI, packaging and
   the isolated E2E/sandbox launchers with the pinned v3 build workflow. Preserve
   embedded opsctl, version ldflags, frontend tooling and platform resources.
6. **Verify and document.** Run Go tests, frontend tests/lint/type checking and
   production builds. Drive boot, a binding, bidirectional events and quit in
   the isolated GUI; inspect logs/data independently. Update maintained docs
   only for implemented behavior. Record failures and unobserved platforms.

## Risk controls

- Keep changes limited to desktop integration; use existing protocol tests.
- Verify v3 event envelopes and cancellation rather than assuming v2 semantics.
- Preserve the production-data refusal for `OPSKAT_E2E=1`.
- Keep Go module, CLI and frontend runtime on the same exact release.
- Review generated binding exposure, custom extension assets and platform
  WebView storage paths before treating a successful build as acceptance.

## Execution record

- Started from a clean v2.16.0 checkout; pinned Go module, CLI and frontend
  runtime to Wails v3.0.0. Existing `AGENTS.md` was preserved.
- Ported service registration, native events/dialogs/clipboard/file drops,
  single-instance identity and asynchronous quit preparation. Retained the v2
  WebView origin and macOS bundle identity to preserve workspace preferences.
- Added native workspace creation and asset context-menu actions. Each window
  owns its tab/session state; opening an asset starts an independent connection.
  Existing live sessions are not transferred between windows.
- Regenerated bindings and migrated domain consumers through generated clients.
  Wiring-only methods are excluded from binding generation. Regression tests
  cover event envelopes/routing, nullable IPC results and generated model parsing.
- Moved development/build/test runners to Bun, pinned in `.bun-version`.
  Retained `pnpm-lock.yaml`, with pnpm executed through Bun for dependency installs.
- Fixed reproduced white screens: Vite now binds IPv4 explicitly and allows the
  persisted `wails` host while continuing to reject unrelated hosts. The real HTTP
  regression fails with 403 without the allowlist and passes with it.
- Full Go tests passed; final focused desktop/system/AI/entrypoint and test-build
  checks passed. Frontend lint/type checking and 390 files / 4049 tests passed.
  Isolated boot/native-window specs passed (2 tests); real native WebViews also
  rendered production static assets in both windows and exited after saving.
- Windows amd64 executable and nightly NSIS installer compiled on this host.
  Corrected numeric resource versions, bootstrapper staging and temporary-resource
  cleanup. macOS plist validation exposed leftover v2 templates, now removed.
- Windows/Linux runtime behavior, Linux packaging, remote-protocol workflows and
  macOS notarization were not exercised. Native checks used only isolated data.
  Wails MCP and the browser verification transport are absent from release builds.

Final artifact: `build/bin/opskat.app` passed plist validation and signature
verification; binary metadata confirms only `production,embed_opsctl` tags.
The local Go 1.27 linker warns that its Go object targets macOS 13 while the
bundle declares 12; compatibility with older macOS releases remains unverified.

## Continuation record (2026-11-12, later session)

Completed the suite-green acceptance the first pass left open, plus follow-up
fixes surfaced by full verification:

- The Playwright suite had never been green on v3 (25/48). Root causes fixed:
  specs/fixtures still spoke the v2 bridge (`window.go.*`, `window.runtime`) —
  replaced with in-page dynamic imports of the generated clients and
  `src/lib/desktop.ts` (`e2e/fixtures/bindings.ts`); the settings workspace
  navigation (`?page=settings`) has no asset tree, so `createGenericAssetViaUI`
  now returns to the main workspace first.
- SSH terminal connect raced its own events: progress events can beat the
  binding call's response, and the tab/subscription existed only after the
  response. Connect requests now carry a client-generated connection id
  (honored or rejected at the IPC boundary, `*_ops.go`), the store pre-creates
  the tab/entry and subscribes before calling, and the tab is opened before the
  call so `attachSession`'s rekey can never miss it. Suite now 48/48.
- `wails3 dev` teardown orphans `scripts/desktop.mjs run` + the app, poisoning
  the next local run. `run-e2e.mjs` now reaps them by the run's
  `OPSKAT_DATA_DIR` (env-matched, never by command line), regression-tested in
  `e2e/harness/process-cleanup.test.cjs`.
- golangci-lint upgraded to v2.15.0 (CI pin bumped); the six findings it added
  (errcheck on error-channel receives, one misspell, SA4000/SA4023) fixed.
- e2e gained the missing `typescript` devDependency so `typecheck` can run.
- Module renamed `github.com/opskat/opskat` → `github.com/octays/opskat`
  (applied outside this session); bindings regenerated, `go.mod` replace for
  gopher-rdp pointed back at the existing public fork after `go mod tidy`.
- Success toasts (top-center) now clear the new top tab bar instead of sitting
  on it and swallowing clicks (`--offset-top: 84px`, DESIGN.md §6.4 notes the
  coupling).
- `useWailsEvent` unsubscribes with the listener's own cancel; the previous
  `EventsOff(name)` cleanup could silently detach another component listening
  to the same event name.
