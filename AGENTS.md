# CRV Go — instructions for future agents

## Read before editing

1. `docs/CONTEXTO.md` for the verified current state and approval boundary.
2. `docs/specs/2026-09-28-crv-offline-design.md` for the approved product contract.
3. `docs/plans/2026-09-28-crv-offline-plan.md` for the five implementation phases and their validation gates.
4. `docs/DECISIONS.md` for durable decisions; `docs/PROXIMOS_PASSOS.md` for active work.
5. `docs/design_app_crv.md` only for the stage fields/help explicitly retained by the approved spec. It is otherwise historical. `README.md` and `docs/VERIFICACAO.md` describe the prototype and earlier checks.

The spec and plan were approved on 2026-09-28. Phases 1–3 are on `main` (`fed1076` for Phase 3). Phase 4 is implemented in the working tree and stopped at the user-validation gate. Later phases need separate authorization. Never infer permission to commit, push, sign or publish from phase approval.

## Project rules

- Reply to the user in Portuguese. Preserve approved decisions and resolve routine details without reopening the stack.
- This is a separate application from Thiagojm/SRV. Do not edit that repository or recreate its interface.
- Target stack: Go + Chi, Svelte + TypeScript + Vite + Tailwind, Canvas 2D and SQLite; local/offline on Windows and Linux, with independent histories and mouse drawing.
- Phase 4 adds CSV export, printable session HTML for native Save as PDF, and ZIP backup/restore on top of Phase-3 history/stats/catalog tools. Packaged release remains Phase 5. Do not present prototype localStorage results as experimental session data.
- Keep all perceptual fields optional, with multiple checkboxes and free text; preserve contextual help, collapsed fictional examples, drawings, AOL, protocol versions, abandonment and separate post-feedback comments.
- Assign the target and three distinct distractors before collection; keep the answer in the backend. Lock the original record before alternatives; persist the choice before feedback. Never select distractors based on session content.
- The bank is tracked at `farsight/` and will ship with the app. Preserve credits. Do not place target images or the answer catalog in the frontend bundle. There is one session mode, continuous descriptive statistics, no blocks/reservations, and every eligible image remains available for each new session.
- The existing visual design is the baseline for targeted changes, not pixel-level approval.
- At each material phase boundary, update `docs/CONTEXTO.md` and `docs/PROXIMOS_PASSOS.md` with actual evidence and the next authorization boundary.

## Verified commands and limits

On Windows, use `npm.cmd run check`, `npm.cmd run build`, `go vet ./...`, `go test ./...`, `npm.cmd run test:e2e` and `go build -o bin/crv.exe .` (Node 24.18.0, Go 1.26.5); use `npm.cmd` if PowerShell blocks `npm.ps1`. Do not change global execution policy. Bundled-catalog decode reported 192 eligible / 4 excluded. Prior prototype browser checks in `docs/VERIFICACAO.md` remain separate from final-app delivery evidence. See `docs/CONTEXTO.md` for the latest Phase-4 evidence table.
