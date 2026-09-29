# CRV Go — current project context

Updated: 2026-09-28. This file records the current checkout and execution state. The approved design and phase plan are the authoritative implementation sources.

## Product and current state

CRV Go is a personal, local/offline desktop-browser application for recording CRV-inspired stages I–III and choosing one image among four after a blinded record. It is independent of Thiagojm/SRV. The intended application runs on Windows and Linux with Go + Chi, Svelte/TypeScript/Vite/Tailwind, Canvas 2D and SQLite. No account or synchronization is planned.

**Phase 3 is committed and pushed.** The Go executable initializes the bundled `farsight/` catalog, serves the UI on loopback, runs blinded persisted sessions, exposes real history (newest-first, date/state filters, pagination), continuous descriptive statistics (SVG cumulative chart + table), optional catalog replacement (ZIP or absolute folder path), and in-place catalog repair by SHA from `CatalogDir`. History detail allows editing the post-feedback comment. Exports/PDF/CSV/ZIP backup remain Phase 4. The standalone historical prototype remains at `crv_prototipo.html`.

The source catalog is `farsight/catalog-unified.json` with `farsight/images/`. After decode validation and deduplication on this Windows checkout: **192 eligible** unique images, **4 exclusions**, inventory summary **multi=2 / empty=1**. Display assets are metadata-free PNGs under the managed revision directory. Unknown `/api/*` routes return 404. Target identity and source hashes are absent from collection and locked-choice API responses; images are served only as opaque session-position URLs after lock. Historical abandoned views keep the target hidden.

## Approved sources and authorization

The user explicitly approved `docs/specs/2026-09-28-crv-offline-design.md` and `docs/plans/2026-09-28-crv-offline-plan.md` on 2026-09-28. Phases 1–3 were authorized, implemented, committed and pushed. The user then authorized **Phase 4** (“commit and push, inicie a fase 4”) with full PDF/CSV + ZIP backup/restore in one pass. That request does **not** authorize Phase 5, signing or publication.

## Verification evidence (Phase 3, Windows)

| Check | Result |
| --- | --- |
| `npm.cmd run check` | 0 errors / 0 warnings |
| `npm.cmd run build` | `dist/index.html` single-file build OK; no demo/bank leaks in assets |
| `go vet ./...` | clean |
| `go test ./...` | pass (`internal/store`, `internal/catalog`, `internal/session`, `internal/app`) |
| `npm.cmd run test:e2e` | 5 passed (3 session + 2 history/catalog) |
| `go build -o bin/crv.exe .` | OK |
| Browser (Playwright MCP) | Histórico, paginação, edição de comentário, reparo in-place, estatísticas, Configurações import/reparo; viewport móvel 390×844 |

Go/store tests cover empty stats, A9 example, chart ordering, date/state filters, abandoned blinding, ZIP/folder import, traversal rejection, active-session 409, and in-place repair (same revision id; mismatched source rejected).

Dependencies unchanged: Chi `v5.3.2` (MIT), `modernc.org/sqlite v1.59.0`, `golang.org/x/sys v0.47.0`, Playwright `1.63.0` (dev). Module: `github.com/Thiagojm/crv-go`, Go 1.26.5, Node 24.18.0.

## Limitations (still unverified or out of phase)

- Linux runtime, offline packaged distribution, and automatic browser-open failure UX remain Phase 5.
- PDF/CSV exports and ZIP backup/restore are the active Phase 4 work.
- Interactive user mouse validation of Phase 4 artifacts is the next gate after implementation.

## Next authorization boundary

Implement Phase 4, then stop for user testing. Start Phase 5 only after explicit user approval. See `docs/PROXIMOS_PASSOS.md`.
