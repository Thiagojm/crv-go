# CRV Go — current project context

Updated: 2026-10-01. This file records the current checkout and execution state. The approved design and phase plan are the authoritative implementation sources.

## Product and current state

CRV Go is a personal, local/offline desktop-browser application for recording CRV-inspired stages I–III and choosing one image among four after a blinded record. It is independent of Thiagojm/SRV. The intended application runs on Windows and Linux with Go + Chi, Svelte/TypeScript/Vite/Tailwind, Canvas 2D and SQLite. No account or synchronization is planned.

**Phase 5 was explicitly authorized on 2026-10-01; local packaging and drawing corrections are implemented, with platform validation still incomplete.** Phase 4 and the partial-move restore correction (`551bdd0`) are on `main`. The Go executable initializes the bundled `farsight/` catalog, serves the UI on loopback, runs blinded persisted sessions, exposes history/statistics/catalog import+repair, CSV export, printable session HTML for native Save as PDF, and ZIP backup/restore with pre-restore copy and startup recovery for interrupted swaps. The standalone historical prototype remains at `crv_prototipo.html`.

The source catalog is `farsight/catalog-unified.json` with `farsight/images/`. After decode validation and deduplication on this Windows checkout: **192 eligible** unique images, **4 exclusions**, inventory summary **multi=2 / empty=1**. Display assets are metadata-free PNGs under the managed revision directory. Unknown `/api/*` routes return 404. Target identity and source hashes are absent from collection, locked-choice API responses, and CSV/unconfirmed print reports; completed print views may include target image and credits. Images are served only as opaque session-position URLs after lock.

## Approved sources and authorization

The user explicitly approved `docs/specs/2026-09-28-crv-offline-design.md` and `docs/plans/2026-09-28-crv-offline-plan.md` on 2026-09-28. Phases 1–3 are on `main` (`fed1076` for Phase 3). Phase 4 and its restore corrections were committed and pushed after explicit requests. On 2026-10-01 the user explicitly requested Phase 5 with `tjm-multi-agent`. The user separately requested project-memory synchronization and commit/push of this delivery and its drawing corrections on the same date. Signing/publication and future changes remain outside that authorization. Earlier native Phase-4 validation remains unverified where no evidence was supplied.

## Verification evidence (Phase 4, Windows)

| Check | Result |
| --- | --- |
| `npm.cmd run check` | 0 errors / 0 warnings |
| `npm.cmd run build` | `dist/index.html` single-file build OK |
| `go vet ./...` | clean |
| `go test ./...` | pass (`app`, `backup`, `catalog`, `export`, `session`, `store`) |
| `go test -race ./...` | **blocked** on this host: `CGO_ENABLED=0` → `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1` |
| `npm.cmd run test:e2e` | 7 passed (session + history/catalog + exports/backup + headless PDF pages) |
| `go build -o bin/crv-phase4.exe .` | OK (`crv.exe` may be locked by a running instance) |
| Browser (Playwright) | CSV; print HTML; Chromium `page.pdf()` multipage artifact; pypdf + pymupdf inspection; backup ZIP |

Phase 4 APIs: `GET /api/exports/csv`, `GET /api/exports/sessions/{id}/print`, `GET /api/backup`, `POST /api/backup/restore` (multipart `archive` + `confirm=true`). Backup format `crv-backup-v1` with SHA-256 manifest; restore refuses active collecting/locked sessions and creates a verified pre-restore ZIP under `backups/`. Startup calls `backup.ResolveInterrupted` before opening SQLite.

Phase 4 corrections (`221b923` and a follow-up rollback fix, still at user-validation gate before Phase 5):

- Restore coordination: `dataMu` RWMutex in auth wrappers; shared handlers hold `RLock` for the whole request; restore/catalog/backup/create take exclusive `Lock` so in-flight work drains before swap. `maintaining` still blocks new auth. Tested with an in-flight comment held across restore.
- `ApplySwap`: on move failure, only clears `OldDir`/marker after a successful undo; failed undo keeps `live_moved`+`OldDir` for `ResolveInterrupted`. Covered by failed-move+failed-undo test.
- Partial-move rollback: the marker records which live payload entries existed before the swap. Recovery restores entries moved to `OldDir`, preserves old entries still live, and removes only newly created entries. A regression test moves SQLite but leaves `catalog/` live, then fails the undo; startup recovery retains both the session and catalog. After this fix, `go test ./... -count=1`, `go vet ./...`, `npm.cmd run check`, `npm.cmd run build`, `npm.cmd run test:e2e` (7 passed), and `go build -o bin/crv-rollback-review.exe .` passed on Windows.
- `ValidateArchive`: required revision IDs come from the staged DB (active + session-referenced); a manifest that omits one is rejected. Image-path checks run for those required IDs.
- Printable PDF: target `data:` URI typed as `template.URL` (avoids `#ZgotmplZ`); Esboço uses `drawing-block-continued` with a forced page break so heading and drawing stay together. E2E asserts `img.naturalWidth > 0`, pypdf `--min-images 1`, and pymupdf `--heading-has-drawings` for Ideograma/Esboço. Rendered PDF pages inspected under `tmp-data/phase4-pdf-evidence/`.
- Prior hardening retained: `live_moved` before payload move; Create/Validate require referenced catalog files; backup download awaits `queue.flush()`.

Dependencies unchanged: Chi `v5.3.2` (MIT), `modernc.org/sqlite v1.59.0`, `golang.org/x/sys v0.47.0`, Playwright `1.63.0` (dev). Module: `github.com/Thiagojm/crv-go`, Go 1.26.5, Node 24.18.0.

## Limitations (still unverified or out of phase)

- Linux runtime remains unverified: this Windows host has no installed Linux/WSL environment. Both platform archives are generated locally; successful cross-compilation is not Linux runtime evidence.
- Native browser “Save as PDF” dialog remains a separate manual check; automated evidence uses Chromium headless `page.pdf()` plus `tests/inspect_pdf.py` (pypdf) on disposable artifacts under `tmp-data/phase4-pdf-evidence/` (gitignored).
- `go test -race` was not executed here because the toolchain has `CGO_ENABLED=0`; re-run on a host with CGO/gcc before claiming race coverage.
- Commit/push of this delivery was separately requested on 2026-10-01; signing/publication and future changes require separate explicit requests.

## Phase 5 local packaging evidence (2026-10-01)

- `npm.cmd run package` builds the embedded frontend and Go binaries (`CGO_ENABLED=0`, Windows/Linux amd64), and writes `packages/crv-go-windows-amd64.tar.gz`, `packages/crv-go-linux-amd64.tar.gz`, and `packages/SHA256SUMS.txt`. Generated packages are gitignored.
- Each archive contains the executable, unchanged sibling `farsight/`, local instructions, dependency/source notices and license texts. Notices contain actual toolchain versions, Go module checksums, npm integrity records and the catalog inventory hash. No installer, additional dependency or runtime Node/Go requirement was introduced.
- Tar listings were inspected: no user data, backups, node_modules or test artifacts; Linux `crv` has executable mode 0755, including when produced on Windows.
- Current test host: Windows 11 Pro 10.0.26300 (build 26300); Go 1.27.1; Node 24.21.0; npm 12.2.0; Playwright Chromium 153.0.8010.12. `go test ./...`, `go vet ./...`, `npm.cmd run check` (0 errors/warnings), frontend build and packaging passed.
- Final `npm.cmd run test:e2e` with `CRV_PACKAGE_DIR` pointing to the extracted final Windows archive: **8 passed** (including the distribution test). Without the variable, the distribution test is intentionally skipped.
- `tests/package.spec.ts` is opt-in via `CRV_PACKAGE_DIR`, and launches the extracted binary from an unrelated directory with spaces, an isolated data directory and empty child PATH. It performs bootstrap, real-bank startup (192 eligible / 4 excluded), a mouse drawing, graceful shutdown/restart, persisted drawing verification, explicit editing takeover/resume, lock/choice/feedback, CSV, headless PDF with decoded target, ZIP backup/restore, missing-bank and insufficient-bank (three eligible images) startup and shutdown.
- Browser external requests are blocked and no external request was observed. This demonstrates browser-network independence under the test, not physical host-network disconnection. Home screenshots in both themes (1280-wide) and dark 1000-wide were inspected; horizontal overflow at 1000 pixels is checked. This is not a complete native keyboard/drawing/printing review.
- Native default-browser opening, native Save as PDF, full offline host-network-disabled operation and real Linux runtime remain unverified. Phase 5 is incomplete until the required platform/manual evidence is supplied. See `docs/VERIFICACAO.md` for rerun commands.

## Drawing-save follow-up (2026-10-01)

- User reported failure when continuing after an ideogram and impressions. Reproduced in Playwright by dragging outside the canvas while pointer capture is active: the frontend sent coordinates outside 1000 × 620 and backend validation rejected the record.
- The shared DrawingPad now clamps sampled points to the logical canvas bounds for both ideogram and sketch. Backend validation remains unchanged. Failed step navigation now includes the server's save error.
- Regression evidence: the extended session test verifies successful save responses, persisted boundary coordinates (0,0 and 1000,620), drawing outside both canvases and advancement through lock/feedback. `npm.cmd run check`: zero errors/warnings; `npm.cmd run test:e2e`: 7 passed, opt-in distribution test skipped.
- The earlier packaged binary remains running for the user's unsaved browser draft. The corrected executable is `bin/crv-drawing-fix.exe`, using a separate disposable test data directory when launched. Existing Phase-5 archives predate this correction and must be regenerated after the running packaged binary has been closed. Commit/push of this delivery was separately requested on 2026-10-01.

## Sketch canvas follow-up

Sketch canvas follow-up (2026-10-01): the short-window CSS constrained/centered the canvas inside a larger dashed wrapper. Removed those two overrides so the shared drawing surface fills its wrapper at the original 1000:620 aspect ratio; taller surfaces use normal page scrolling. Extended the session regression to check wrapper/canvas geometry and aspect ratio at 1000×720, 1280×720 and 1440×900. All three session E2E tests pass; frontend check has zero errors/warnings. The full source E2E suite was rerun before commit: 7 passed / 1 skipped (distribution opt-in). The user accepted the canvas correction ("ótimo"); this does not close the remaining platform-validation gate. Corrected executable: `bin/crv-canvas-fix.exe`. The existing running sessions and their tabs are preserved; launch retesting with separate disposable data. Phase-5 archives still need regeneration after closing the previous packaged executable.

## Next authorization boundary

Stop for packaged application testing and the remaining Phase-5 platform evidence. Do not call Phase 5 complete from cross-compilation or browser automation alone. See `docs/PROXIMOS_PASSOS.md`. The user requested commit/push of the current delivery on 2026-10-01; this does not authorize signing/publication or future changes.
