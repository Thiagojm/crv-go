# CRV Go — current project context

Updated: 2026-09-28. This file records the current checkout and execution state. The approved design and phase plan are the authoritative implementation sources.

## Product and current state

CRV Go is a personal, local/offline desktop-browser application for recording CRV-inspired stages I–III and choosing one image among four after a blinded record. It is independent of Thiagojm/SRV. The intended application runs on Windows and Linux with Go + Chi, Svelte/TypeScript/Vite/Tailwind, Canvas 2D and SQLite. No account or synchronization is planned.

The repository currently has a navigable Svelte prototype in `src/`, `crv_prototipo.html` and `dist/index.html`. It includes mouse drawing, optional fields, help, review, choice, feedback and demo localStorage history. The answer and four repeated Unsplash photos are frontend-visible. There is no Go module/server, SQLite database, effective blinding, real bank integration, final-app PDF or ZIP backup. The prototype must not be used as an experimental application.

The source catalog is `farsight/catalog-unified.json` with `farsight/images/`. Git tracks the catalog and 242 image files on `main`/`origin/main`; this replaces the former private-package-only distribution rule. The catalog retains provenance and credits from three pools: 714 source records, 715 references and 242 distinct SHA-256 file hashes. There are 238 distinct images referenced by a single-image source record, two multi-image source entries and one empty entry. Decoding and final eligibility have not been validated. Historical manifest names are provenance, not files to recreate. Do not treat source-entry count as unique-image count.

## Approved sources and authorization

The user explicitly approved `docs/specs/2026-09-28-crv-offline-design.md` on 2026-09-28. The companion `docs/plans/2026-09-28-crv-offline-plan.md` defines five independently validated phases. The older `docs/design_app_crv.md` remains authoritative only for stage fields/help referenced by the new spec; its modes, fixed blocks, reservation cycles and first-use import rule are superseded. `docs/DECISIONS.md` contains compact durable choices.

The user requested project-memory synchronization, a commit and push of the approved planning/memory artifacts, and an implementation handoff for another agent. The user explicitly said not to start implementation in this task. No phase of product implementation has been requested. A later implementation request starts Phase 1 only, followed by a stop for user testing; each later phase requires separate approval. The current request authorizes this planning/documentation commit and push, not product code or release actions.

## Verification and remaining work

On this Windows checkout, `npm.cmd ci`, `npm.cmd run check` (0 errors/warnings) and `npm.cmd run build` passed on 2026-09-28 with Node 24.18.0. Go 1.26.5 is available, but no Go code/tests exist. The built prototype HTML was restored to its tracked version after verification. Interactive drawing/navigation on Windows and real application execution on Linux were not checked in this review. `docs/VERIFICACAO.md` records older Linux Chromium tests of the prototype only.

Next action is an explicit Phase-1 implementation request from the user, then the first validation gate in the approved plan. See `docs/PROXIMOS_PASSOS.md` for the live task list. No final-app tests, offline distribution or experimental safeguards have been implemented yet.
