# CRV Go — current project context

Updated: 2026-09-28. This file records the current checkout and execution state. The approved design and phase plan are the authoritative implementation sources.

## Product and current state

CRV Go is a personal, local/offline desktop-browser application for recording CRV-inspired stages I–III and choosing one image among four after a blinded record. It is independent of Thiagojm/SRV. The intended application runs on Windows and Linux with Go + Chi, Svelte/TypeScript/Vite/Tailwind, Canvas 2D and SQLite. No account or synchronization is planned.

**Phase 4 is implemented in the working tree and awaits user validation.** The Go executable initializes the bundled `farsight/` catalog, serves the UI on loopback, runs blinded persisted sessions, exposes history/statistics/catalog import+repair, CSV export, printable session HTML for native Save as PDF, and ZIP backup/restore with pre-restore copy and startup recovery for interrupted swaps. Packaged Windows/Linux distribution remains Phase 5. The standalone historical prototype remains at `crv_prototipo.html`.

The source catalog is `farsight/catalog-unified.json` with `farsight/images/`. After decode validation and deduplication on this Windows checkout: **192 eligible** unique images, **4 exclusions**, inventory summary **multi=2 / empty=1**. Display assets are metadata-free PNGs under the managed revision directory. Unknown `/api/*` routes return 404. Target identity and source hashes are absent from collection, locked-choice API responses, and CSV/unconfirmed print reports; completed print views may include target image and credits. Images are served only as opaque session-position URLs after lock.

## Approved sources and authorization

The user explicitly approved `docs/specs/2026-09-28-crv-offline-design.md` and `docs/plans/2026-09-28-crv-offline-plan.md` on 2026-09-28. Phases 1–3 are committed and pushed (`fed1076` for Phase 3). The user authorized **Phase 4** (“commit and push, inicie a fase 4”) with full PDF/CSV + ZIP backup/restore. That request does **not** authorize Phase 5, signing, publication, or a Phase-4 commit/push unless asked again.

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

Phase 4 corrections (same working tree, still at user-validation gate):

- Restore coordination: `dataMu` RWMutex in auth wrappers; shared handlers hold `RLock` for the whole request; restore/catalog/backup/create take exclusive `Lock` so in-flight work drains before swap. `maintaining` still blocks new auth. Tested with an in-flight comment held across restore.
- `ApplySwap`: on move failure, only clears `OldDir`/marker after a successful undo; failed undo keeps `live_moved`+`OldDir` for `ResolveInterrupted`. Covered by failed-move+failed-undo test.
- `ValidateArchive`: required revision IDs come from the staged DB (active + session-referenced); a manifest that omits one is rejected. Image-path checks run for those required IDs.
- Printable PDF: target `data:` URI typed as `template.URL` (avoids `#ZgotmplZ`); Esboço uses `drawing-block-continued` with a forced page break so heading and drawing stay together. E2E asserts `img.naturalWidth > 0`, pypdf `--min-images 1`, and pymupdf `--heading-has-drawings` for Ideograma/Esboço. Rendered PDF pages inspected under `tmp-data/phase4-pdf-evidence/`.
- Prior hardening retained: `live_moved` before payload move; Create/Validate require referenced catalog files; backup download awaits `queue.flush()`.

Dependencies unchanged: Chi `v5.3.2` (MIT), `modernc.org/sqlite v1.59.0`, `golang.org/x/sys v0.47.0`, Playwright `1.63.0` (dev). Module: `github.com/Thiagojm/crv-go`, Go 1.26.5, Node 24.18.0.

## Limitations (still unverified or out of phase)

- Linux runtime and offline packaged distribution remain Phase 5.
- Native browser “Save as PDF” dialog remains a separate manual check; automated evidence uses Chromium headless `page.pdf()` plus `tests/inspect_pdf.py` (pypdf) on disposable artifacts under `tmp-data/phase4-pdf-evidence/` (gitignored).
- `go test -race` was not executed here because the toolchain has `CGO_ENABLED=0`; re-run on a host with CGO/gcc before claiming race coverage.
- Working tree Phase-4 changes are **uncommitted** until a separate request.

## Next authorization boundary

Stop here for user testing of Phase 4. Start Phase 5 only after explicit user approval. See `docs/PROXIMOS_PASSOS.md`.
