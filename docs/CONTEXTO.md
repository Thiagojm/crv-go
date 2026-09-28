# CRV Go — current project context

Updated: 2026-09-28. This file records the current checkout and execution state. The approved design and phase plan are the authoritative implementation sources.

## Product and current state

CRV Go is a personal, local/offline desktop-browser application for recording CRV-inspired stages I–III and choosing one image among four after a blinded record. It is independent of Thiagojm/SRV. The intended application runs on Windows and Linux with Go + Chi, Svelte/TypeScript/Vite/Tailwind, Canvas 2D and SQLite. No account or synchronization is planned.

**Phase 1 is implemented in the working tree and awaits user validation.** The Go executable initializes the bundled `farsight/` catalog into an OS/user data directory (or `--data-dir`), serves the embedded UI on loopback with bootstrap/CSRF protection, reports catalog readiness, and supports orderly shutdown. Real sessions remain disabled until Phase 2. The standalone historical prototype remains at `crv_prototipo.html`.

The source catalog is `farsight/catalog-unified.json` with `farsight/images/`. After decode validation and deduplication on this Windows checkout: **192 eligible** unique images, **4 exclusions** (inventory images without a single-image source), with inventory summary counts **multi=2 / empty=1** preserved from the catalog `summary`/`reviewEntries`. Display assets are metadata-free PNGs under the managed revision directory. Installation inserts an inactive revision, finalizes files, verifies them, then activates; `EnsureInstalled` reuses a verified revision even if the distribution `farsight/` folder is later absent. Unknown `/api/*` routes return 404. The source `farsight/` tree is not modified by installation.

## Approved sources and authorization

The user explicitly approved `docs/specs/2026-09-28-crv-offline-design.md` and `docs/plans/2026-09-28-crv-offline-plan.md` on 2026-09-28. Planning artifacts were committed/pushed as `a48ebc6`. The user then authorized **Phase 1 only**, with a stop at the user-validation gate. That request does **not** authorize commits, pushes, Phase 2, signing or publication.

## Verification evidence (Phase 1, Windows)

| Check | Result |
| --- | --- |
| `npm.cmd run check` | 0 errors / 0 warnings |
| `npm.cmd run build` | `dist/index.html` single-file build OK |
| `go vet ./...` | clean |
| `go test ./...` | pass (`internal/store`, `internal/catalog`, `internal/app`) |
| `go build -o bin/crv.exe .` | OK (~17 MB) |
| Bundled catalog test | eligible=192, excluded=4 (logged; not hard-coded) |
| Isolated launch + API | ready=true, sessionsOpen=false, Host/Origin/CSRF denials OK, `/farsight/` 404 |
| Second instance lock | refused with existing URL/pid |
| Restart same data dir | readiness persisted, single `rev-1` |
| Missing catalog dir | ready=false, clear error, no empty demo bank |
| Browser (Playwright) | catalog banner, Nova sessão disabled, theme/settings/help/history, Salvar e encerrar |

Dependencies pinned and license-checked at install: Chi `v5.3.2` (MIT), `modernc.org/sqlite v1.59.0` (BSD + SQLite public domain), `golang.org/x/sys v0.47.0` (BSD). Module: `github.com/Thiagojm/crv-go`, Go 1.26.5, Node 24.18.0.

## Limitations (still unverified or out of phase)

- Linux runtime, offline network inspection of a packaged distribution, and automatic browser-open failure UX were not re-checked as final delivery evidence (A1 complete remains Phase 5).
- No session create/lock/choice, no image API, no history/statistics/export/backup (Phases 2–4).
- Interactive user mouse validation of this Phase-1 shell is the current gate.
- Working tree changes are **uncommitted**; do not commit/push without a separate request.

## Next authorization boundary

Stop here for user testing of Phase 1. Start Phase 2 only after explicit user approval. See `docs/PROXIMOS_PASSOS.md`.
