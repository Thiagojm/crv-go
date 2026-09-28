# CRV Go — next actions

Updated: 2026-09-28. The approved implementation plan is `docs/plans/2026-09-28-crv-offline-plan.md`; this file tracks authorization and live work, not a second step-by-step plan.

## Completed orientation milestones

- [x] Frontend prototype and its historical Linux Chromium review delivered; the visual direction is the baseline for targeted changes.
- [x] Three source pools consolidated into the Git-tracked `farsight/` catalog and 242 images, preserving source records and credits. Final decoding/eligibility is not yet verified.
- [x] New offline design explicitly approved on 2026-09-28; phased implementation plan written.
- [x] Phase 1 implemented in the working tree: Go launch, SQLite catalog install, loopback protection, readiness UI, shutdown. Automated Windows checks passed; see `docs/CONTEXTO.md`.

## Current boundary

- [ ] **User validation of Phase 1** (isolated data dir, readiness without image previews, second-instance lock, missing-bank message, Salvar e encerrar). No commit/push unless separately requested.
- [ ] After acceptance, explicit authorization required before Phase 2 (blinded persisted sessions).
- [ ] At every phase gate, report what passed, what remains unverified, how to test, and update `docs/CONTEXTO.md` and this file. Preserve unrelated work and keep commit/push separate from phase authorization.

## Later phases (not yet authorized)

- [ ] Phase 2: persistent blinded session, server-side transitions, autosave/recovery, drawing and choice, direct API/browser tests.
- [ ] Phase 3: real history, continuous descriptive statistics, optional catalog replacement.
- [ ] Phase 4: PDF/CSV exports and consistent ZIP backup/restore.
- [ ] Phase 5: Windows/Linux distribution and separate real runtime validation.

## Deferred and out of scope

Native full-session drawing on Windows, Firefox behavior, and real Linux execution remain unverified until relevant phase testing. No SRV edits, old-session migration, accounts/sync, mobile app, AI interpretation, stages IV–VI, fixed blocks, release signing or publication are included in the approved implementation phases.
