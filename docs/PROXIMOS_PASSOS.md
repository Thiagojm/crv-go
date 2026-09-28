# CRV Go — next actions

Updated: 2026-09-28. The approved implementation plan is `docs/plans/2026-09-28-crv-offline-plan.md`; this file tracks authorization and live work, not a second step-by-step plan.

## Completed orientation milestones

- [x] Frontend prototype and its historical Linux Chromium review delivered; the visual direction is the baseline for targeted changes.
- [x] Three source pools consolidated into the Git-tracked `farsight/` catalog and 242 images, preserving source records and credits. Final decoding/eligibility is not yet verified.
- [x] New offline design explicitly approved on 2026-09-28; phased implementation plan written.

## Current boundary

- [ ] Wait for a separate user request to start Phase 1. No product implementation is authorized by the planning approval or this documentation handoff.
- [ ] On Phase-1 authorization, follow only the approved plan's offline startup and bundled catalog scope. Validate catalog decoding, safe paths, hashes, actual eligibility, loopback access, startup/shutdown, and the partial Windows evidence specified there. Stop for user testing and explicit Phase-2 approval.
- [ ] At every phase gate, report what passed, what remains unverified, how to test, and update `docs/CONTEXTO.md` and this file. Preserve unrelated work and keep commit/push separate from phase authorization.

## Later phases (not yet authorized)

- [ ] Phase 2: persistent blinded session, server-side transitions, autosave/recovery, drawing and choice, direct API/browser tests.
- [ ] Phase 3: real history, continuous descriptive statistics, optional catalog replacement.
- [ ] Phase 4: PDF/CSV exports and consistent ZIP backup/restore.
- [ ] Phase 5: Windows/Linux distribution and separate real runtime validation.

## Deferred and out of scope

Native Windows drawing/navigation checks of the historical prototype, Firefox behavior, and real Linux execution remain unverified until relevant phase testing. No SRV edits, old-session migration, accounts/sync, mobile app, AI interpretation, stages IV–VI, fixed blocks, release signing or publication are included in the approved implementation phases.
