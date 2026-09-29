# CRV Go — durable decisions

Accepted on 2026-09-28 unless otherwise stated. These decisions are implemented only when the approved phase plan reaches them; Phase 4 is the active authorized work. Full behavior and acceptance criteria are in `docs/specs/2026-09-28-crv-offline-design.md`.

## Product and presentation — accepted

- Keep the new project independent of SRV; reuse its source bank and credits, not its application interface. Use Go + Chi, Svelte + TypeScript + Vite + Tailwind, Canvas 2D and SQLite for a Windows/Linux local/offline app. This preserves separate installation histories and a mouse-first workflow.
- Keep the prototype's visual direction as the starting point with targeted changes. This is not pixel-level approval. Preserve optional checkboxes, text, sketches, contextual help and collapsed fictional examples so the user can record impressions without forced interpretation.
- Use one session mode and continuous descriptive history/statistics. The user rejected separate familiarization/evaluation modes and fixed blocks. Count confirmed choices for hit rate and use 25% only as the four-choice random reference; do not claim confirmatory significance from accumulated results.

## Bank and blinding — accepted

- Include the Git-tracked `farsight/` catalog and images in the application distribution, ready for offline first use. Install managed revisions locally and allow optional later import. The bank is backend-controlled; preserve all credits and source provenance, and never bundle target images into Vite.
- Sample uniformly from eligible unique images, not pool entries. Pick one target and three distinct distractors before collection. Every eligible image is available for each new session, even if shown earlier. No reservations, exclusion cycles or exposure-based weighting. This avoids silently giving repeated pool entries more selection weight.
- Persist target/order before collection, hide the answer and images until the authorized state, lock the original before alternatives, and commit choice before feedback. Distractors never depend on session content. Keep the original record and feedback comment separate.

## Delivery and execution — accepted

- The approved design supersedes the older `docs/design_app_crv.md` where it describes mandatory first-use import, pool-A-only selection, two modes, fixed blocks, no-repeat cycles or Wilson/binomial block inference. Its stage fields and help remain the source for those exact lists/texts.
- The approved plan has five phases, each ending with user validation and separate authorization for the next. Spec/plan approval does not authorize implementation, commits, pushes or release work. Phases 1–3 were authorized, implemented, committed and pushed. Phase 4 was separately authorized (full exports + backup/restore); it does not authorize Phase 5, signing or publication.
- Catalog repair recomposes missing files in-place into existing `rev-{id}` directories from a matching `CatalogDir` by SHA-256. It must not deactivate the revision used by historical sessions or allocate a new revision id for repair.
- Preserve server-side state transitions, crash recovery, per-installation storage and narrow loopback API access. These are required before calling a session blinded or saved. The prototype's localStorage flow does not satisfy them.
