# CRV Go — next actions

Updated: 2026-09-28. The approved implementation plan is `docs/plans/2026-09-28-crv-offline-plan.md`; this file tracks authorization and live work, not a second step-by-step plan.

## Completed orientation milestones

- [x] Frontend prototype and its historical Linux Chromium review delivered; the visual direction is the baseline for targeted changes.
- [x] Three source pools consolidated into the Git-tracked `farsight/` catalog and 242 images, preserving source records and credits.
- [x] New offline design explicitly approved on 2026-09-28; phased implementation plan written.
- [x] Phase 1 implemented: Go launch, SQLite catalog install, loopback protection, readiness UI, shutdown.
- [x] Phase 2 implemented, committed and pushed: blinded persisted session, server-side transitions, autosave/recovery, drawing and choice, API and Playwright tests.
- [x] Phase 3 implemented, committed and pushed (`fed1076`): real history (pagination + comment edit), continuous statistics, optional catalog ZIP/folder replacement and in-place repair.
- [x] Phase 4 implemented in the working tree: CSV export, printable session HTML (Exportar PDF → Salvar como PDF nativo), ZIP backup/restore with pre-restore copy and interrupted-swap recovery.

## Current boundary

- [ ] **User validation of Phase 4** after the latest restore/PDF corrections (export short/long PDF via browser dialog and confirm the target image + Esboço stay intact, open CSV in a spreadsheet, backup ZIP then restore into an isolated data dir, confirm paused active session returns from backup, reject bad archive). No commit/push unless separately requested.
- [ ] After acceptance, explicit authorization required before Phase 5 (packaging and real OS delivery evidence).
- [ ] At every phase gate, report what passed, what remains unverified, how to test, and update `docs/CONTEXTO.md` and this file. Preserve unrelated work and keep commit/push separate from phase authorization.

## Later phases (not yet authorized)

- [ ] Phase 5: Windows/Linux distribution and separate real runtime validation.

## Deferred and out of scope

Native full-session drawing on Windows, Firefox behavior, and real Linux execution remain unverified until relevant phase testing. No SRV edits, old-session migration, accounts/sync, mobile app, AI interpretation, stages IV–VI, fixed blocks, release signing or publication are included in the approved implementation phases.
