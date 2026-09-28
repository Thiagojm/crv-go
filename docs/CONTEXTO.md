# CRV Go — current project context

Updated: 2026-09-28. This file records the current checkout and execution state. The approved design and phase plan are the authoritative implementation sources.

## Product and current state

CRV Go is a personal, local/offline desktop-browser application for recording CRV-inspired stages I–III and choosing one image among four after a blinded record. It is independent of Thiagojm/SRV. The intended application runs on Windows and Linux with Go + Chi, Svelte/TypeScript/Vite/Tailwind, Canvas 2D and SQLite. No account or synchronization is planned.

**Phase 2 is implemented in the working tree and awaits user validation.** The Go executable initializes the bundled `farsight/` catalog, serves the UI on loopback, and runs a persisted blinded session: create → collect stages I–III → lock → choose A–D → feedback/comment, with autosave, pause/resume, abandonment, revision conflicts and a per-tab editing lease. Real history, statistics, catalog replacement and exports remain Phase 3–4. The standalone historical prototype remains at `crv_prototipo.html`.

The source catalog is `farsight/catalog-unified.json` with `farsight/images/`. After decode validation and deduplication on this Windows checkout: **192 eligible** unique images, **4 exclusions**, inventory summary **multi=2 / empty=1**. Display assets are metadata-free PNGs under the managed revision directory. Unknown `/api/*` routes return 404. Target identity and source hashes are absent from collection and locked-choice API responses; images are served only as opaque session-position URLs after lock.

## Approved sources and authorization

The user explicitly approved `docs/specs/2026-09-28-crv-offline-design.md` and `docs/plans/2026-09-28-crv-offline-plan.md` on 2026-09-28. Planning artifacts were committed/pushed as `a48ebc6`. Phase 1 was authorized and implemented, then the user authorized **Phase 2**. That request does **not** authorize commits, pushes, Phase 3, signing or publication.

## Verification evidence (Phase 2, Windows)

| Check | Result |
| --- | --- |
| `npm.cmd run check` | 0 errors / 0 warnings |
| `npm.cmd run build` | `dist/index.html` single-file build OK |
| `go vet ./...` | clean |
| `go test ./...` | pass (`internal/store`, `internal/catalog`, `internal/session`, `internal/app`) |
| `npm.cmd run test:e2e` | 3 passed (fluxo cego, recarga/pausa/segunda aba, abandono + encerrar) |
| `go build -o bin/crv.exe .` | OK |

Go tests cover uniform four-distinct assignment, repeated sessions on a 4-image bank, create idempotency, random-failure rollback, record validation, lock/confirm/abandon state machine, timing without revision bump, lease transfer, API blinding, guessed image 404, concurrent create, stale revision, failed save without unlocking images, and restart-pause.

Dependencies: Chi `v5.3.2` (MIT), `modernc.org/sqlite v1.59.0`, `golang.org/x/sys v0.47.0`, Playwright `1.63.0` (dev). Module: `github.com/Thiagojm/crv-go`, Go 1.26.5, Node 24.18.0.

## Limitations (still unverified or out of phase)

- Linux runtime, offline packaged distribution, and automatic browser-open failure UX remain Phase 5.
- History list, continuous statistics, catalog replacement (Phase 3); PDF/CSV/ZIP (Phase 4).
- Interactive user mouse validation of this Phase-2 session is the current gate.
- Working tree changes are **uncommitted**; do not commit/push without a separate request.

## Next authorization boundary

Stop here for user testing of Phase 2. Start Phase 3 only after explicit user approval. See `docs/PROXIMOS_PASSOS.md`.
