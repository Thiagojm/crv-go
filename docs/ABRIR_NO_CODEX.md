# Open CRV Go in Codex

This guide describes the current local checkout at `D:\Projetos\crv-go`. If the repository is cloned elsewhere, use that actual path.

1. Open the repository as a local Codex project.
2. Read `AGENTS.md`, `docs/CONTEXTO.md`, `docs/specs/2026-09-28-crv-offline-design.md` and `docs/plans/2026-09-28-crv-offline-plan.md` before editing.
3. Inspect `git status` and the current branch. Preserve unrelated changes.
4. The approved design and plan do not authorize implementation on their own. Wait for an explicit Phase-1 request; stop after the phase's user-validation gate. Later phases require separate authorization.

The checkout currently contains a Svelte prototype and the tracked `farsight/` bank. There is no Go backend or effective blinding yet. The older `docs/design_app_crv.md` supplies the exact stage fields and help referenced by the approved spec; its older mode, block and bank-distribution rules have been superseded.
