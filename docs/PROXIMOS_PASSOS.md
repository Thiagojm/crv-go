# CRV Go — next actions

Updated: 2026-10-01. The approved implementation plan is `docs/plans/2026-09-28-crv-offline-plan.md`; this file tracks authorization and live work, not a second step-by-step plan.

## Completed orientation milestones

- [x] Frontend prototype and its historical Linux Chromium review delivered; the visual direction is the baseline for targeted changes.
- [x] Three source pools consolidated into the Git-tracked `farsight/` catalog and 242 images, preserving source records and credits.
- [x] New offline design explicitly approved on 2026-09-28; phased implementation plan written.
- [x] Phase 1 implemented: Go launch, SQLite catalog install, loopback protection, readiness UI, shutdown.
- [x] Phase 2 implemented, committed and pushed: blinded persisted session, server-side transitions, autosave/recovery, drawing and choice, API and Playwright tests.
- [x] Phase 3 implemented, committed and pushed (`fed1076`): real history (pagination + comment edit), continuous statistics, optional catalog ZIP/folder replacement and in-place repair.
- [x] Phase 4 implemented, committed and pushed (`221b923`): CSV export, printable session HTML (Exportar PDF → Salvar como PDF nativo), ZIP backup/restore with pre-restore copy and interrupted-swap recovery.
- [x] Phase 4 follow-up: partial-move rollback preserves live catalog files when the SQLite move and undo fail; regression and full Windows checks pass. User validation remains pending.

## Current boundary

- [x] User-reported drawing-save failure fixed locally: clamp pointer-captured coordinates to the canvas in ideogram and sketch; navigation reports the server error. Regression and frontend checks pass (7 E2E passed, package test skipped).
- [x] Sketch drawable-area correction: removed short-window canvas constraints; the full dashed area now matches the drawing surface without stretching its logical aspect ratio. Geometry verified at three window sizes; 3 session E2E tests pass.
- [x] User accepted the canvas correction ("ótimo") and requested memory synchronization plus commit/push on 2026-10-01. This is not evidence for the remaining Linux/native platform checks.
- [ ] Regenerate Phase-5 archives after closing the previous running packaged executable; current archives predate the drawing corrections. Keep earlier browser tabs open if they contain unsaved edits.

- [ ] **User validation of Phase 4** after the latest restore/PDF corrections (export short/long PDF via browser dialog and confirm the target image + Esboço stay intact, open CSV in a spreadsheet, backup ZIP then restore into an isolated data dir, confirm paused active session returns from backup, reject bad archive).
- [x] Phase 5 explicitly authorized on 2026-10-01 using `tjm-multi-agent`; local Windows/Linux amd64 packaging implemented, with Windows extracted-binary browser evidence.
- [ ] Finish Phase-5 platform validation: real Linux execution; default-browser opening; physically disconnected/offline operation; native Save as PDF; complete keyboard/drawing checks. These remain incomplete; no Linux runtime environment is installed on this host.
- [ ] At every phase gate, report what passed, what remains unverified, how to test, and update `docs/CONTEXTO.md` and this file. Preserve unrelated work and keep commit/push separate from phase authorization.

## Delivery boundary

- [ ] User validation of the extracted distributions and remaining platform evidence before declaring Phase 5 complete.
- Commit/push of the current Phase-5 delivery and drawing corrections was separately requested on 2026-10-01. Signing/release publication and future changes remain unauthorized.

## Deferred and out of scope

Native full-session drawing on Windows, Firefox behavior, and real Linux execution remain unverified until relevant phase testing. No SRV edits, old-session migration, accounts/sync, mobile app, AI interpretation, stages IV–VI, fixed blocks, release signing or publication are included in the approved implementation phases.
