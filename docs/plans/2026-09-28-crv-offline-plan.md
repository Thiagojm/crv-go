# CRV Go — phased implementation plan

Date: 2026-09-28
Status: Planning complete; implementation has not been requested.
Approved specification: `docs/specs/2026-09-28-crv-offline-design.md`, explicitly approved by the user on 2026-09-28.

## Goal and boundaries

Deliver the approved local/offline application with a bundled image bank, one session mode, four distinct alternatives per session, unrestricted reuse between sessions, server-authoritative records, continuous descriptive statistics, exports, and recoverable backups. Preserve the existing visual baseline and all optional stage fields/help.

No implementation phase is authorized by approval of the spec or this plan. A subsequent implementation request authorizes Phase 1 only. At the end of every phase, report changes, evidence, limitations and user test instructions, then stop. Start each later phase only after the user has tested and explicitly authorized it. Do not prebuild later-phase schema, routes or UI placeholders. Material design changes require a revised spec and renewed approval.

Do not modify SRV, add modes/blocks/reservations, build inferential statistics, migrate prototype histories, synchronize installations, or publish anything. No commits, pushes, signing, release publication, or deployment are included.

## Starting point and prerequisites

- Work in `D:\Projetos\crv-go`; check `git status`, repository instructions and relevant diffs before each phase. Preserve unrelated changes, including user commits made between phases.
- The source bank is `farsight/catalog-unified.json` and `farsight/images/`. Do not reconstruct deleted source pool files or restore the old asset location. Current inventory: 714 source records, 242 distinct files, 238 single-image candidates before decoding validation.
- The prototype is `src/App.svelte`, `src/data.ts`, `src/style.css` and the existing components. `dist/index.html` and `crv_prototipo.html` are prototype artifacts. Do not treat localStorage data as real session history.
- Node 24.18.0 and Go 1.26.5 were observed locally. Frontend check/build passed earlier on this Windows checkout; no final-app tests exist yet.
- Preserve existing frontend dependency versions and lockfile unless a phase requires a specific addition. No framework, CSS or Svelte upgrade is planned.
- Go module: `github.com/Thiagojm/crv-go`; Go language/toolchain baseline 1.26. Pin Chi `github.com/go-chi/chi/v5 v5.3.2` and SQLite `modernc.org/sqlite v1.59.0`. Use `golang.org/x/sys v0.47.0`, already required by that SQLite release, for Windows/Linux OS file locks. Module metadata was checked on 2026-09-28; this is not yet compilation/runtime verification.
- Phase 2 adds development-only `@playwright/test 1.63.0` for the critical browser workflow. Its registry version was checked during planning; install it only within that phase. Runtime browser automation is not part of the application.
- At the start of the phase introducing a dependency, verify its license and compatibility, install the pinned version, record the exact resolved graph, and keep `go.sum`/`package-lock.json` reproducible. A conflicting version is a reported blocker or justified plan revision, not an excuse for a blanket dependency upgrade.
- Use the Go standard library for HTTP helpers, JSON, cryptographic sampling, image JPEG/GIF/PNG handling, hashing, ZIP, CSV and templates. No image-processing service, charting library or generic persistence framework is needed.

## File organization

Use one executable entrypoint in root `main.go` so `go:embed` can embed the existing root `dist/`. Keep narrowly scoped production packages under `internal/app`, `internal/catalog`, `internal/store`, `internal/session`, and later `internal/export` and `internal/backup`. Use concrete types/functions, not interfaces created solely for hypothetical implementations. Add test seams for clock/randomness/failures only where invariant tests need them.

All paths below are repository-relative. New-file targets are exact; add a new file only when the corresponding phase needs it. Frontend page/component extraction is limited to the new surfaces named below, rather than a broad rewrite of `App.svelte`.

## Phase 1 — Launch offline with the bundled catalog

**Goal:** A user can start the Go executable, initialize the real bank, see catalog readiness in the existing shell, and exit safely. Real sessions remain disabled until Phase 2.

**Create:** `go.mod`, `go.sum`, `main.go`; `internal/app/server.go`, `internal/app/security.go`, `internal/app/instance.go`, `internal/app/instance_windows.go`, `internal/app/instance_linux.go`; `internal/catalog/catalog.go`, `internal/catalog/catalog_test.go`; `internal/store/store.go`, `internal/store/migrations/001_catalog.sql`, `internal/store/store_test.go`; `internal/app/server_test.go`; `src/api.ts`.

**Modify:** `src/App.svelte`, `src/data.ts`, `src/style.css`, `vite.config.ts`, `.gitignore`, `README.md`, `docs/VERIFICACAO.md`, `docs/CONTEXTO.md`, `docs/PROXIMOS_PASSOS.md`.

1. Add the pinned Go module dependencies and a root entrypoint. Resolve bundled `farsight/` relative to the executable; provide explicit developer flags `--data-dir`, `--catalog-dir` and `--no-browser` for isolated tests. Development overrides must not change release defaults.
2. Implement OS user-data directory selection and an OS-held exclusive file lock using Windows LockFileEx/Linux flock. Retain instance port metadata for a second launch, but never treat a stale metadata file as a live lock. Release the lock on normal/crashed process exit through OS semantics.
3. Initialize SQLite, foreign keys, transactional migration versioning, bounded busy handling and catalog/preferences tables only. Keep the data directory separate from the repository and executable. A failed initialization must not mark a catalog active.
4. Parse the unified catalog, validate safe paths and links, hashes, record/source integrity, image size/dimensions and JPEG/GIF decoding. Decode/re-encode display assets as metadata-free PNG, using the first GIF frame. Reuse original SHA-256 identities; preserve all source metadata. Apply the approved limits before allocations and reads, not after decompression/decoding.
5. Produce a catalog report: structural failures block activation; per-image decode/eligibility failures are exclusions with reasons. Preserve multi-image/empty source entries in provenance/report. Activate only after staging contains at least four eligible distinct images. Copy into managed immutable revision storage without altering `farsight/`.
6. Add loopback-only serving, exact Host/Origin policy, same-origin session cookie plus mutation CSRF checks, and a one-use bootstrap token. Prefer passing the bootstrap token in a URL fragment and exchanging it for the credential without access-log leakage; clear the fragment immediately. Reject unknown hosts and all external mutation origins. Authenticated readiness/status responses contain counts and errors, never target metadata or file paths.
7. Serve the embedded UI and expose readiness/catalog summary/settings/shutdown endpoints. Browser-launch failure prints a usable local URL. No static bank mount or image API exists yet. Keep the UI explicitly labeled as under construction and disable Nova sessão with a clear Phase-2 explanation; retain Conhecer o fluxo and theme controls. Do not make fake demo records appear to be real saved sessions.
8. Implement orderly shutdown for this phase, reusing the same protection for its endpoint. Build the frontend before `go build`; preserve the existing standalone prototype artifact as a historical reference rather than overwriting it as a final product.

**Automated verification:** `npm.cmd run check`, `npm.cmd run build`, `go test ./...`, `go vet ./...`, `go build -o bin/crv.exe .`. Catalog tests use tiny generated JPEG/GIF fixtures plus malformed/traversal/link/hash cases and the installed-bank inventory. HTTP tests deny untrusted Host/Origin, missing credentials/CSRF and guessed target/static routes. Run a bundled-catalog test that records the actual decoded eligible count without hard-coding a promised count of 238. Restart initialization and prove no duplicated revision or modified source bank.

**User test:** Launch with an isolated empty data directory and bundled bank path. Observe progress/report and ready state without previews; close/reopen and verify readiness persists. Launch twice and verify the second instance cannot write. Test missing-bank behavior and Salvar e encerrar. No real-session testing is claimed yet.

**Completion/gate:** Startup, catalog validation, loopback protection and shutdown pass; document exclusions and exact local evidence for A1/A2/A12 as partial Windows validation. Stop and request user validation and explicit Phase-2 authorization.

## Phase 2 — Complete a persisted, blinded session

**Depends on:** Accepted Phase 1. **Goal:** Perform the full collection → lock → choice → feedback flow, with reliable saves/resume/abandonment and no hidden answer in collection/choice responses.

**Create:** `internal/session/session.go`, `internal/session/session_test.go`; `internal/store/sessions.go`, `internal/store/migrations/002_sessions.sql`, `internal/store/sessions_test.go`; `internal/app/sessions.go`, `internal/app/sessions_test.go`; `src/session.ts`, `src/components/ConfirmDialog.svelte`; `playwright.config.ts`, `tests/session.spec.ts`.

**Modify:** `internal/app/server.go`, `internal/app/security.go`, `src/api.ts`, `src/App.svelte`, `src/data.ts`, `src/style.css`, `src/components/DrawingPad.svelte`, `src/components/FieldGroup.svelte`, `src/components/Record.svelte`, `src/components/Help.svelte`, `package.json`, `package-lock.json`, `README.md`, `docs/VERIFICACAO.md`, `docs/CONTEXTO.md`, `docs/PROXIMOS_PASSOS.md`.

1. Define concrete records with stable field/option IDs, nullable integer confidence, bounded text/strokes, immutable protocol/help snapshot and independent stage I/III drawings. Preserve the legacy document's stage options and help text, replacing prototype-only warnings with accurate final-flow guidance. Add corresponding client types without a code-generation framework.
2. Add the sessions migration: `collecting`, `locked`, `completed`, `abandoned`; a partial unique constraint allowing at most one collecting/locked session; revision counter; creation operation ID; assignment/order/catalog reference; timestamps/timers; record and locked snapshot; tentative/confirmed choice; comment and audit events. Use a direct JSON record column for structured fields where appropriate. Keep relational checks for state/choice integrity. No modes/blocks/reservations tables.
3. Implement uniform target selection, three remaining distinct distractors and shuffled positions using `crypto/rand`. Persist everything with neutral code before returning a created session. Test injected random/read failure and transaction rollback. Draw from the full eligible set each time; a four-image bank must permit arbitrarily many successive sessions.
4. Implement create/get, save record, finalize, tentative choice, confirm, abandon and post-feedback comment operations. Creation retries use operation IDs. Check idempotent replay before rejecting an already-completed matching transition; conflicting new choices fail. Commit choice/result/state before constructing feedback. The server never accepts a client-supplied correct target/result.
5. Serialize state-specific DTOs and authorized image/feedback routes. Opaque URLs identify session positions, not source hashes or paths. Block image access before lock, including guessed URLs. After abandonment, allow only the alternatives previously authorized, without correct-answer metadata. No private model may be serialized directly by a handler.
6. Add revision checking and the approved five-second heartbeat/fifteen-second editing lease. Keep lease generation/token checks server-side; revoke old ownership on confirmed transfer. Other tabs are read-only, and stale writes preserve their local draft with an explicit reload action. Timing increments must not advance the record revision and create artificial autosave conflicts. Persist timing deltas once with monotonic sequence/acknowledgment handling; clamp each interval to five seconds and never backfill downtime.
7. Replace real-flow localStorage/demo assignments with API calls. `src/session.ts` owns a single serialized save queue and 500 ms text debounce. Finalize/navigation/pause/choice/shutdown await queued saves. Keep unsaved work on error and block irreversible actions. Do not migrate existing prototype storage or delete it automatically.
8. Wire existing stage components, mouse drawing, review and 2×2 choice layout to server state. Preserve optional blanks, summaries, confidence and separate AOL. Use ConfirmDialog for lock/choice/abandon/takeover and image enlargement focus behavior. Add keyboard focus/status handling; retain current design tokens.
9. Restore active sessions paused on reload/restart. Count time only in the visible, owning, unpaused session; stop intervals on visibility changes/navigation/pause and flush the final partial interval. Help does not pause. Disable editing while paused. Preserve target/order across all interruptions and image failures.
10. Make feedback immediately mark a session completed; comments remain separately editable. Allow starting a new session after completion/abandonment. Keep history/statistics/export actions unavailable with honest explanations until later phases, rather than showing prototype totals/exports against real data.
11. Add `test:e2e` for Playwright and an isolated Go-backed server process per test worker (initially one worker). Generate synthetic catalogs in temporary directories; do not put real bank photographs or answer IDs in test reports. Shut down child servers and clean only test-owned temporary directories. Never run browser tests against the user's normal data directory.

**Automated verification:** Standard frontend checks, `go test ./...`, `go vet ./...`, and `npm.cmd run test:e2e`. Cover transitions via direct API, immutable snapshot, concurrent creates, stale revision/lease, same/different choice replay, save failure, response loss after commit, abandonment and guessed image paths. Use deterministic clock/random seams rather than flaky frequency assertions or long sleeps. Browser cases cover drawing/checkbox/text/help, blank completion, reload before and after lock, tentative choice, response loss/reload, two tabs, pause and shutdown. Inspect network responses for source metadata/hidden answers at each state. A race-enabled Go run may be performed on a host with a supported race toolchain; do not claim it from the ordinary test run.

**User test:** In an isolated data directory, draw and record all three stages, reload/resume, pause, review and lock. Verify fields become immutable, choose once, inspect feedback and add a comment. Abandon another session before and after alternatives. Open a second tab and exercise takeover. Try narrow desktop height and both themes.

**Completion/gate:** A3–A8/A12/A13 and relevant A5/A14 browser paths have evidence; original records and blinding are enforced through the API. Known UI/OS limitations are documented. Stop for user validation and explicit Phase-3 authorization.

## Phase 3 — Continuous history, statistics and catalog settings

**Depends on:** Accepted Phase 2. **Goal:** Review real session history and accumulated results, and manage local catalog revisions without breaking existing sessions.

**Create:** `internal/store/history.go`, `internal/store/history_test.go`; `internal/app/history.go`, `internal/app/history_test.go`; `internal/catalog/import.go`, `internal/catalog/import_test.go`; `internal/app/catalog.go`; `src/components/History.svelte`, `src/components/Statistics.svelte`; `tests/history.spec.ts`.

**Modify:** `internal/catalog/catalog.go`, `internal/app/server.go`, `src/App.svelte`, `src/api.ts`, `src/data.ts`, `src/style.css`, `README.md`, `docs/VERIFICACAO.md`, `docs/CONTEXTO.md`, `docs/PROXIMOS_PASSOS.md`.

1. Add backend history queries, newest-first pagination and date/state filters. Convert local date boundaries explicitly to UTC; retain timestamps as UTC. Return only fields allowed for the session state. Historical views reuse Record and the feedback rendering from Phase 2.
2. Implement initiated/active/completed/abandoned-before/abandoned-after counts, confirmed choices, hits and hit rate. Keep cumulative statistics over the complete installed history, independent of history-page filters. Return zero-denominator values as absent, not 0%. Draw the cumulative hit/reference chart with native SVG; include a textual/table equivalent. No chart library, p-values, Wilson intervals or significance messages.
3. Add catalog summary/report and authenticated ZIP upload/local-folder import actions. Reuse Phase-1 staging/validation; enforce archive limits and path/link checks while streaming. Do not duplicate an importer for the bundled bank. Serialize activation against session creation so no session can be created using an inconsistent revision.
4. Disallow replacement while collecting/locked. After completion/abandonment, activate a validated revision atomically and leave referenced old revisions untouched. Catalog failure must preserve the active revision. Session creation snapshots the active revision without rewriting historical metadata. Provide repair by revalidating/reimporting a matching revision; repair never changes the saved assignment.
5. Replace mode/block filters, demo counts and import placeholders with the approved history/settings flow. Remove unused demo image imports and `newSession`/frontend randomization from `src/data.ts` once no runtime callers remain. Preserve the standalone historical prototype artifact; real application bundles must contain no answer catalog or demo fallback.

**Automated verification:** Frontend checks, Go tests/vet, and Playwright suite. Verify the spec's 1 hit + 1 miss + 1 abandonment example; empty history; ordering and timezone boundaries; hidden fields in abandoned history; active-session replacement rejection; corrupt ZIP and folder-link cases; successful replacement preserving historical feedback. Inspect built assets for accidental bank imports. Test no-update catalog initialization and versioned replacements separately.

**User test:** Review completed and abandoned records, filter dates/states, verify totals and graph against a few known sessions. Import a copied test package, observe its report and verify old records remain unchanged. Confirm active-session replacement is disabled. No archive manipulation of the real source bank is required for the user test.

**Completion/gate:** A2/A4/A8/A9/A12 and relevant A14 pass for the implemented surfaces. Stop for user validation and explicit Phase-4 authorization.

## Phase 4 — Exports and recoverable backup/restore

**Depends on:** Accepted Phase 3. **Goal:** Export records and recover a complete installation without data loss or leaking hidden answers into ordinary reports.

**Create:** `internal/export/export.go`, `internal/export/export_test.go`, `internal/export/session.html`; `internal/backup/backup.go`, `internal/backup/restore.go`, `internal/backup/backup_test.go`; `internal/app/exports.go`, `internal/app/exports_test.go`; `tests/exports.spec.ts`.

**Modify:** `internal/app/server.go`, `internal/app/instance.go`, `internal/store/store.go`, `src/App.svelte`, `src/api.ts`, `src/components/History.svelte`, `README.md`, `docs/VERIFICACAO.md`, `docs/CONTEXTO.md`, `docs/PROXIMOS_PASSOS.md`.

1. Build one state-filtered report model, reused by CSV and printable session HTML. Use `encoding/csv` and normalize formula-like user strings. Test separators, quotes/newlines and leading formula characters. No unconfirmed target identity or path may enter this model.
2. Implement PDF export through a local print-ready HTML view and the browser's native print/Save as PDF. Use `html/template`, embedded local font, inline SVG strokes generated from validated coordinates, page-break CSS and authorized completed-target images only. Keep original drawings uncropped; split long text naturally across pages. The user action is labeled Exportar PDF and explains the native Save as PDF dialog. This avoids a PDF library or a browser runtime bundled with Go. Do not claim silent filesystem PDF creation.
3. Flush pending session saves and use SQLite's supported consistent snapshot facility to produce a backup database under exclusive application maintenance coordination. Package that snapshot, all referenced/current managed revisions and originals/display files, preferences, format version and SHA-256 manifest. Write to a temporary artifact and expose it only after successful finalization. Refuse an archive that would exceed the approved restore limits so the app cannot knowingly generate an unrestorable backup; leave existing data intact and explain the size limit.
4. Implement restore staging with schema/version/integrity/path/hash/bounds checks and the same archive safety rules as catalog import. Refuse restore while there is a current collecting/locked session. Create and verify the pre-restore backup before replacement. Acquire exclusive maintenance access against all mutating routes, stop ordinary requests, close database handles, then swap complete managed data directories using a durable recovery marker. Keep the process lock and recovery control files outside the directory being swapped.
5. Make startup resolve an interrupted swap to an intact old or verified new dataset before opening SQLite. Test interruption after each durable step. On success restart application state, invalidate stale browser revisions/credentials, and resume any restored active session paused. A failed restore must not leave a partly replaced catalog/database pair.
6. Wire CSV/PDF/session export and settings backup/restore. Explain that a full ZIP contains hidden assignments. Require explicit replacement confirmation and show the pre-restore backup location without exposing target information. Do not provide a history-merge option.

**Automated verification:** Standard checks plus export/backup API tests and browser suite. Parse CSV with malicious/quoted content; assert no hidden fields. Generate PDFs from the print route in headless Chromium, render representative pages and inspect drawings, long text, credits and page boundaries. Round-trip completed/abandoned/active-backed-up records and compare semantic data/assignments/hashes. Test corrupt/truncated/oversized/traversal/link/new-version archives and restore interruption at each swap stage. Keep all tests in disposable data directories. Native Save as PDF is checked separately from headless generation.

**User test:** Save a short and long session as PDF, open the documents, inspect drawings/credits, and import the CSV into a spreadsheet. Create a ZIP, restore it into an isolated test installation, and verify history/comments/drawings. Restore an active-session backup into an installation with no active session and confirm the same paused session returns. Verify a rejected bad archive leaves existing history intact.

**Completion/gate:** A4/A10/A11/A12 have automated and generated-artifact evidence; native dialog checks are labeled separately. Stop for user validation and explicit Phase-5 authorization.

## Phase 5 — Package and validate Windows/Linux delivery

**Depends on:** Accepted Phase 4. **Goal:** Produce reproducible offline distribution archives and document actual platform evidence without publishing a release.

**Create:** `scripts/package.mjs`, `docs/THIRD_PARTY_NOTICES.md`.

**Modify:** `package.json`, `.gitignore`, `README.md`, `docs/VERIFICACAO.md`, `docs/DESIGN-SYSTEM.md`, `docs/CONTEXTO.md`, `docs/PROXIMOS_PASSOS.md`; `main.go`, `internal/app/server.go`, existing tests/components only if packaging or final verification exposes a scoped defect.

1. Add a Node stdlib packaging script for the developer build only: build frontend; compile the Go executable with `CGO_ENABLED=0` for Windows amd64 and Linux amd64; stage the binary, sibling `farsight/`, instructions and notices into platform directories; create archives with available OS archive tools. Restore any temporary process environment values. Exclude user data, backups, development dependencies and test outputs. Do not create installer/signing/release infrastructure.
2. Record Go/npm versions, dependency checksums/licenses, catalog revision and source credits in distribution notices. Keep the original source credits/usage statements intact; Git/distribution approval does not require editing those source statements.
3. Test fresh archives from unrelated working directories and from paths containing spaces. Run with network disabled, without relying on a development server, Node or installed Go. Confirm first-use catalog installation, browser bootstrap, drawing, a complete session, restart, exports, restore and graceful shutdown on Windows and Linux separately.
4. Check both themes and 1280×720 plus a narrower desktop window, keyboard dialogs, image proportions and restored drawings. Check that packaged frontend/assets do not contain the bank or secret session state. Check the missing-bank/insufficient-bank repair states from a copied distribution.
5. Update README to distinguish final-app instructions, developer commands and the historical prototype. Update verification with exact OS/browser versions, commands, artifact names and limitations. Never present successful cross-compilation as Linux runtime validation.

**Automated verification:** `npm.cmd run check`, `npm.cmd run build`, `go test ./...`, `go vet ./...`, `npm.cmd run test:e2e`, and the new `npm.cmd run package` (use `npm` on Linux). Test staged archive contents and launch on both target systems. Run only additional focused checks when a newly discovered issue or change warrants them.

**User test:** Extract each available platform archive and use it offline through startup, one full session, restart and a PDF/backup operation. If the other OS is unavailable, record that requirement as unverified and keep the platform-validation portion incomplete until evidence is supplied.

**Completion/gate:** All A1–A14 have passing evidence, or explicit unresolved items keep the phase incomplete. Present artifacts and user-facing limitations, then stop. No push, signing or publication occurs.

## Validation matrix and phase reporting

| Spec acceptance | Primary phase(s) |
| --- | --- |
| A1 offline distribution | 1 startup; 5 complete real OS delivery |
| A2 catalog identity/validation | 1 bundled bank; 3 replacement |
| A3 sampling/recovery | 2 |
| A4 state-dependent disclosure | 2 sessions; 3 history; 4 reports |
| A5 optional fields/help | 2 |
| A6 save/drawing recovery | 2 |
| A7 concurrency/idempotency | 2 |
| A8 abandonment | 2 transitions; 3 history |
| A9 continuous statistics | 3 |
| A10 backup/restore | 4 |
| A11 PDF/CSV | 4 |
| A12 local security/validation | 1 foundation; 2–4 each new operation |
| A13 timing/shutdown | 2 behavior; 5 real OS verification |
| A14 visual/accessibility | 2–4 browser checks; 5 final OS review |

At each gate, update `docs/CONTEXTO.md` and `docs/PROXIMOS_PASSOS.md` with the current phase, completed scope, exact validation and remaining work. Distinguish compilation, automated tests, browser tests, generated PDF inspection, real OS execution and user-reported checks. A check not run stays unverified. Record substantive findings without copying secret assignments or user session text into logs/reports.

## Risks and safeguards

- Catalog eligibility can be lower than the audited candidate count: validate actual decoding before promising a count, preserve excluded provenance, and never silently replace a saved assignment.
- Serving a included bank must not bypass the API boundary: no static catalog route, no Vite asset imports, metadata-free session-scoped image responses.
- Cross-tab saves and response-loss retries can corrupt state if implemented only in the UI: use database constraints/revisions/idempotency and test direct requests.
- Replacing SQLite plus image directories on Windows can fail because handles remain open: exclusive maintenance, verified backup, closed handles and startup recovery are mandatory before directory replacement.
- Continuous statistics are descriptive and allow repeated images; do not reintroduce fixed-block conclusions or inferential badges.
- OS compatibility and PDF print layout require real evidence; build success does not satisfy those acceptance criteria.
