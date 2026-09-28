# CRV Go — offline application design

Date: 2026-09-28
Status: Explicitly approved by the user on 2026-09-28. Implementation requires a separate request.

## 1. Context and authority

CRV Go is a standalone personal application for recording impressions inspired by CRV stages I–III, then choosing between four images. It records subjective impressions and an objective final choice; it does not interpret drawings or establish a paranormal mechanism.

The current application is a Svelte frontend prototype. `src/App.svelte` coordinates navigation and localStorage; `src/data.ts` contains four demonstration images and frontend-visible answers. There is no Go server, SQLite persistence, production catalog integration, or effective blinding. Existing screenshots and the prior verification report describe the prototype, not the finished application.

On this Windows checkout, `npm.cmd ci`, `npm.cmd run check` and `npm.cmd run build` passed on 2026-09-28. Interactive Windows testing and final Linux execution have not been verified. The source catalog is now at `farsight/catalog-unified.json`, with files under `farsight/images/`; the earlier `src/assets/farsight/` location is historical.

This approved specification consolidates the user's subsequent decisions and supersedes conflicting provisions in `docs/design_app_crv.md`: separate modes, fixed blocks, reservations, exclusion cycles, pool-A-only startup, and mandatory first-use import. The original stage fields and contextual guidance remain requirements as specified below. `docs/DESIGN-SYSTEM.md` remains the visual baseline, with the targeted changes in this document.

## 2. Confirmed product decisions

- Go + Chi; Svelte + TypeScript + Vite + Tailwind; Canvas 2D; SQLite.
- Windows and Linux desktop, mouse input, local browser, offline operation, one user, independent installation histories.
- Keep the existing visual direction and layout as a baseline, with targeted adjustments. This is not pixel-level approval.
- One session mode. No Familiarização/Avaliação selector, blocks, predetermined series lengths, or exclusion cycles.
- Exactly four alternatives: one correct target and three distractors, all distinct within the session.
- Every eligible unique image is available for every new session. Prior use, success, abandonment, and exposure do not change selection probabilities. Repetition across sessions is allowed.
- Continuous history and cumulative results.
- Include the bank in the application distribution so first use does not require importing it. Images and catalog are allowed in Git. Preserve source credits.
- Preserve optional fields, help, original drawings, hypotheses/AOL, protocol versions, abandonment, and feedback separate from the original record.
- Preserve PDF, CSV, ZIP backup/restore, and optional local catalog replacement as product capabilities.

## 3. Scope and architecture

Distribute a platform-specific archive containing the executable and a sibling `farsight/` catalog directory. Embed only the compiled frontend, fonts, and ordinary UI resources in the Go executable. Do not import catalog assets into Vite or expose the image directory as static web content. Resolve bundled resources relative to the executable, not the shell's working directory. No Node, Go, internet connection, CDN, or account is required on the user's machine.

On first launch, validate and install the bundled catalog into the application's managed data directory. Use an OS user-data directory, separate from the executable and the SRV checkout. Show initialization progress and any exclusions; do not open image previews automatically. Missing distribution files produce a repair/import action, not an empty demonstration mode. Future launches use the installed catalog and existing SQLite history.

The Go server binds to `127.0.0.1` on an available port and opens the browser. If automatic browser opening fails, show the local URL. Only one process may own an installation's data directory; a second launch reports the existing instance instead of starting a competing writer.

Use SQLite transactions and a small HTTP/domain/storage separation, without a separate service, job system, repository framework, or extensible plugin architecture. A driver without CGO is preferred; exact dependency versions and license checks belong to the implementation plan.

Explicitly out of scope: changes to SRV; hosted service; accounts; synchronization; mobile-first design; AI interpretation; semantic scoring; stages IV–VI; courses; importing prototype or SRV session histories; merging histories. Release publication, signing, deployment, commits, and pushes require their own authorization.

## 4. Catalog and selection

### Inventory and canonical identity

The local audit found 714 source entries, 715 image references, and 242 distinct file hashes. Of these, 238 images have at least one source entry containing exactly one image. Two source entries contain multiple images; one contains none. These are inventory counts, not proof of decoding or final eligibility.

Use the unified catalog as the source format. Preserve `sourceMetadata`, `sourceManifest`, pool provenance, full source records, and credits. Historical manifest filenames are provenance, not required files to reconstruct. Do not require the deleted pool JSONs or temporary audit script.

An eligible image must have a valid recorded hash, a safe local path, successful decoding, and at least one single-image source record. Multi-image and empty entries remain in the import report and provenance; do not silently choose an image from them. Group by original image SHA-256 so repeated pool references do not increase an image's selection probability.

For feedback, choose the primary single-image source deterministically by pool A/B/C and then source ID; retain every associated source in an expandable provenance section. Do not overwrite conflicting descriptions or discard credits. This selection affects presentation only, never sampling weight.

Support the bundled JPEG/GIF formats; decode GIF to its first displayed frame for static presentation. Generate backend-managed display images without embedded EXIF/comments or original filenames, applying the same processing regardless of target/distractor role. Preserve original bytes for source fidelity and backups. Exact-file hashing does not claim to detect all visually similar, cropped, or re-encoded images.

### Import and replacement

Settings offers a local ZIP package or an advanced local folder using the same catalog-plus-images format. Validate into staging before activation. Reject absolute/traversing paths, links, missing references, hash mismatches, malformed records, and duplicate/conflicting identities. Bound extraction to 1 GiB uncompressed, 10,000 files, 20 MiB per image and 40 megapixels per decoded image; reject larger inputs with a clear error. Never modify the supplied source folder/archive.

Invalid individual image candidates appear in the report and are excluded; structural/security failures reject the package. At least four eligible unique images are required to activate a catalog or create a session. Catalog replacement is unavailable while a session is collecting or awaiting a choice. Retain immutable catalog revisions and image bytes referenced by historical sessions. Automatic application updates must not silently replace the active catalog.

### Randomization

Before collection, sample one target uniformly from eligible unique images, then three distractors uniformly without replacement from the remaining images, then uniformly shuffle the four positions. Use Go's system-backed cryptographic randomness with unbiased bounded sampling. Failure to obtain randomness or commit the transaction fails session creation without showing a usable session code.

Persist the target, alternatives, order, independent neutral code, catalog revision, and protocol/help snapshots atomically. The browser receives no target identity or catalog image during collection. Reload, resume, retry, missing images, and errors never trigger a replacement draw. Use all eligible images again for the next session; there are no reservations, depletion counters, or exposure-based weights.

## 5. Screens and visual behavior

Retain the Inter font, neutral background, forest-green actions, light/dark themes, white drawing paper, and desktop layout from `docs/DESIGN-SYSTEM.md`. Retain a 2×2 alternative grid with `object-fit: contain` and image enlargement. Target desktop viewport is 1280×720 or larger; narrower windows stack panels without hiding actions or losing drawings.

| Screen | Required content and actions |
| --- | --- |
| Home | Nova sessão or Continuar sessão, Conhecer o fluxo, accumulated completed-session summary; no mode/block selector. |
| Preparation | Optional disposition/concentration, neutral guidance, ten-minute default reference duration, eligible-image count, Iniciar. No image previews. |
| Stage I | Mouse ideogram, Registrar impressões, optional movement/shape, consistency, gestalt, text per group, separate AOL. |
| Stage II | Optional color, luminosity, texture, temperature, moisture, sound, smell and taste groups; text per group, other impressions and separate AOL. Sound/smell/taste initially collapsed with a filled-content indicator. |
| Stage III | Large sketch canvas; forms, dimensions/proportions, spatial relations, other notes and separate AOL. |
| Review | All fields/drawings including blanks, up to five optional highlights, optional confidence 0–100 with no preset; return to edit or Finalizar registro. |
| Choice | Neutral alternatives A–D, read-only original record, optional choice confidence 0–100, explicit confirmation. |
| Feedback | Correct/incorrect, choice and correct target, description and credits, expandable SAM/provenance, collection/choice times, separate editable post-feedback comment. |
| History | Date, neutral code, state, times and result when confirmed; date-range and state filters, newest first. No modes or blocks. |
| Statistics | Continuous descriptive totals and cumulative hit chart as defined in section 8. |
| Settings | Theme, reference duration, catalog version/count/import report, optional replacement, backup/restore. No default image gallery. |

Preserve the exact initial checkbox options and stage-specific texts in sections 5.2–5.4 and 6 of `docs/design_app_crv.md`; use stable option IDs with versioned Portuguese labels. No perceptual field is mandatory, no option is preselected, and unchecked means unrecorded, not absent. Allow a fully blank record to be finalized after explicit confirmation. Confidence accepts null or an integer from 0 to 100.

Canvas tools are pen, thickness, whole-stroke eraser, undo, redo and confirmed clear. Use the existing 1000×620 logical coordinate space; resizing changes display scale only. Stage I and III keep independent strokes. Derive read-only previews from strokes, never reconstruct originals from comments.

Every stage has a short visible instruction and Como preencher. Help provides objective, how-to, collapsed fictional example and cautions. Known-flow tour is available before creation. Opening help does not pause automatically; opening examples during a session is recorded against that session's help version. Do not use catalog images or target-dependent hints in help.

Keyboard focus must be visible; checkboxes have clickable labels; dialogs trap and restore focus, close safely with Escape, and announce their headings. Saving/error feedback uses accessible status messages. Empty, loading and disabled states explain the action needed without exposing secret data.

## 6. Session, persistence and recovery

There is at most one nonterminal session per installation, enforced transactionally. Preparation does not create a session until Iniciar commits. Double-clicks/retries return the existing created session using an operation ID; concurrent creation cannot produce two active sessions.

| State | Allowed behavior and transitions |
| --- | --- |
| `collecting` | Edit stages I–III/review, pause, save, resume, abandon, or finalize the saved record. No alternatives or feedback. |
| `locked` | Original snapshot is immutable. View the four alternatives, save a tentative selection/confidence, confirm choice, pause/resume, or abandon. No correct-answer metadata. |
| `completed` | Choice/result persisted atomically before feedback is returned. Original and choice are immutable; post-feedback comment can be edited separately. |
| `abandoned` | Terminal, read-only original; no confirmed choice, no feedback, no resuming or new draw under this session ID. |

The final-choice transaction directly enters `completed`; no separate user action or persisted intermediate state is needed to conclude the session. Losing the response after commit recovers completed feedback on reload. Reconfirming the same choice is idempotent; a different choice conflicts. Repeating a successful lock or abandon does not create duplicate events.

Persist edits on checkbox changes and completed strokes, and after 500 ms of text inactivity. Serialize pending saves in the client; navigation, finalization, choice confirmation and shutdown await them. Show Salvando, Salvo or Falha ao salvar only according to actual responses. A failed save retains unsaved work in memory and blocks irreversible progression. A crash can lose unacknowledged edits; never claim otherwise.

Use a revision counter for mutable session data. Updates and transitions require the expected revision. Stale-tab writes receive a conflict without overwriting newer state; keep the local draft visible and offer reloading confirmed data. One browser tab owns an active editing/timing lease; other tabs show read-only state. A five-second heartbeat renews a fifteen-second lease; explicit transfer requires confirmation and server-side revocation. This avoids double timers and silent cross-tab overwrites without a collaboration system.

Store timestamps in UTC and show local time. Count collection and choice time separately. Only the owning tab, visibly displaying the active unpaused session, advances time; time in a hidden tab, outside the session, with the application closed, or explicitly paused is excluded. Bound each acknowledged timing interval to five seconds and do not backfill disconnected time. Help remains part of active session time. Pause saves pending work and disables record/choice editing until resumed.

After interruption, recover the last committed state, original target/order and saved content. Resume paused rather than silently running the clock. Missing/corrupted images cause a repair error, never a replacement draw. Record when the alternative set was first authorized for display and which images the frontend reported loaded; these events are audit data, not claims of human attention and not selection exclusions.

Abandon requires confirmation, preserves the last saved original and whether alternatives were available, and remains in history. It never reveals the target. Previously available alternatives may remain viewable as alternatives only. Post-feedback sessions cannot become abandoned.

Salvar e encerrar flushes pending edits, stops new mutations, completes outstanding transactions, closes storage and exits the local server. Only a successfully acknowledged shutdown request shows the closing screen; errors retain an actionable retry message. Closing the browser alone need not stop the server. If a save fails, do not shut down as if it succeeded.

## 7. Data and API boundaries

Persist catalog revisions/images/source metadata, sessions, original record snapshots, choice/result, post-feedback comments, protocol/help snapshots, preferences, and a small session-event log. No mode, block, cycle or reservation entities are required. Comment edits update only the comment and its modified timestamp. Structured original records may use versioned JSON in SQLite; relational constraints enforce state/choice integrity.

Session DTOs are state-specific. During collection, return neutral code, allowed record fields, state/revision, timing and help/protocol only. After lock, return opaque session-scoped image URLs and A–D positions, never source hashes/paths, descriptions, credits or the correct answer. Confirmed sessions may return feedback metadata. Historical/CSV/PDF endpoints apply the same restrictions. Errors and logs must not contain target identities or user text by default.

Provide explicit operations for create/resume, save record, lock, update tentative choice, confirm choice, abandon, update comment, fetch state/alternatives/feedback, history/statistics, catalog management, export, backup/restore and shutdown. Validate ownership, revision, state and payload server-side on every operation; hiding buttons is insufficient. Image reads also check session state. Serve no target directory or generic catalog-image route.

Use loopback Host/Origin checks, a per-process session credential and mutation CSRF protection; deny cross-origin browser access. Apply protection to shutdown/import/restore as well as session edits. Require an explicit browser bootstrap flow to obtain the credential, and remove its one-time token from the visible URL. Return sensitive API/image responses with `Cache-Control: no-store`. Validate payload types, known field IDs, string/body limits and finite in-bounds stroke coordinates; bound each record to 10 MiB and each text field to 20,000 characters, with client feedback before a rejected save.

The blinding boundary is the normal application/API flow. A user controlling their own executable, source image folder, process or database can inspect files deliberately. Including images in the distribution does not authorize sending future alternatives or the hidden assignment to the browser.

## 8. Continuous statistics

Use one accumulated dataset, with no required session count or end date. Display initiated, active, completed, abandoned before alternatives, abandoned after alternatives, hits, confirmed choices and hit rate. Abandoned/active sessions are not counted as failures or silently removed from initiated totals.

For `n` completed sessions and `h` hits, display `h/n` and `100*h/n`; when `n=0`, show an em dash instead of zero percent. Chart completed sessions in confirmation order, plotting cumulative hits against the reference `0.25*n`. Label 25% as the random-choice reference for four options, not a performance guarantee. No inferred scores from attributes, drawings or comments.

The approved first release provides descriptive tracking only: no fixed-block p-value, significance badge, stopping recommendation, or Wilson interval inherited from the superseded block workflow. Any later inferential analysis is a separate design decision. Retain raw outcomes and protocol/catalog versions so later analysis does not require rewriting history.

## 9. Exports, backup and restore

- PDF: session code/date/state, protocol, all recorded fields, drawings, highlights, confidence, timings and post-feedback comment. Include target identity/image only for completed sessions. Use local fonts and assets with readable multipage layout.
- CSV: one row per session with code, timestamps, state, abandonment stage, times, confirmed choice/result and confidence where permitted. No hidden target IDs or source paths for unconfirmed sessions. Quote delimiters/newlines correctly and neutralize spreadsheet formulas in user-controlled strings.
- ZIP backup: consistent SQLite snapshot plus managed catalog originals/display assets needed by all saved sessions, preferences and a versioned manifest with hashes. Use a database-supported snapshot procedure, not an unchecked live copy of a WAL database. Backup may capture a resumable active session after flushing; explain that it contains hidden assignments.
- Restore: only when no session is collecting/locked and no save/import is pending. Validate format, bounds, paths, hashes and database integrity in staging. Reject newer unsupported formats. Create and verify a pre-restore backup, then replace the complete dataset under exclusive access; retain a rollback marker/copy so interruption resolves to either the old or new complete dataset. Never merge histories or partially replace live data. Failure preserves the prior usable installation.

Catalog import and restore use the same archive/path safety controls. A successful restore restarts the application state and resumes any active session present in the restored backup as paused, preserving its assignment and order.

## 10. Alternatives and rationale

- Separate practice/evaluation modes and reserved fixed blocks were explicitly rejected by the user. One mode and continuous results fit the intended personal workflow.
- Sampling pool entries would favor repeated images. Sampling unique eligible hashes gives each eligible image equal weight.
- A mandatory first-use import was rejected in favor of a ready-to-use bundled bank. A sibling data directory avoids embedding target assets in the frontend and keeps optional replacement straightforward.
- Preserve the current design instead of redesigning the interface. Remove obsolete controls and add real saving/error/recovery states.
- Server-authoritative storage replaces localStorage for real sessions. The prototype's frontend answer assignment cannot provide the required blinding.

## 11. Acceptance and verification

| ID | Observable acceptance criterion | Required evidence |
| --- | --- | --- |
| A1 | Fresh distribution starts offline with its included bank, without Node/Go or manual import. | Real Windows and Linux launch/initialization checks, plus offline browser network inspection. |
| A2 | One selectable candidate per eligible image hash; invalid/multi-image source entries reported; source credits preserved. | Catalog fixtures covering overlap, metadata conflicts, empty/multi-image records, bad decoding/hash/path, and bundled-catalog validation. |
| A3 | Four distinct alternatives per session; all eligible images available again next session; retries/resume preserve assignment. | Deterministic random-boundary tests, no exclusion state, four-image-bank repeated-session regression, persistence/restart checks. |
| A4 | No target identity, images or answer accessible during collection; lock permits only the four alternatives; confirmation permits feedback. | Direct API/image/history/export tests in every state, including abandoned sessions and guessed IDs. |
| A5 | Optional stage fields, blank records, confidence bounds, help and drawings work without target-dependent hints. | Focused validation tests and browser workflow with mouse/keyboard, help and blanks. |
| A6 | Saved fields and drawings survive resize, theme/stage changes and restart; failed save cannot unlock images. | Browser regression plus injected database/save failure and restart tests. |
| A7 | Concurrent create/lock/choice requests cannot duplicate or overwrite committed results. | Transaction/revision/idempotency tests, stale-tab/lease tests and response-loss retries. |
| A8 | Abandonment retains original data, exposes no hidden answer, remains counted, and cannot be resumed. | State-machine/API tests before and after alternatives. |
| A9 | Continuous totals and chart use only confirmed choices for the hit denominator. | Known examples: empty dataset; one hit/one miss plus one abandonment gives 2 choices, 1 hit, 50%; ordering and filter tests. |
| A10 | Backup/restore preserves drawings, comments, immutable originals and assignments without partial replacement. | Round trip, corrupt/traversal/oversized archives, missing image, unsupported format and interrupted replacement checks. |
| A11 | PDF/CSV are readable and reveal only permitted fields. | Generated PDF rendering on representative long/drawn records; parsed CSV escaping/formula tests; unconfirmed-session leakage checks. |
| A12 | Local service resists unintended external browser actions and validates payload bounds. | Host/Origin/CSRF/image-route tests and malformed payload/path cases. |
| A13 | Pause/visibility/resume count time once; shutdown flushes or reports failure. | Controlled-clock tests, two-tab browser check, saved-session restart and real shutdown on each OS. |
| A14 | Current visual baseline remains usable at 1280×720 in both themes; dialogs and labels support keyboard use. | Browser screenshots/interactions and user mouse/keyboard validation; separate Windows/Linux evidence. |

Existing commands remain frontend checks only until backend tests exist. The later implementation plan must introduce meaningful Go tests for the actual invariants and browser coverage for the critical flow. Passing compilation or tests on one OS is not evidence of full runtime support on the other.

## 12. Approval and delivery gates

The user approved this document, including its operational defaults (catalog packaging/validation, metadata precedence, GIF treatment, concurrency, timing, descriptive statistics and recovery), on 2026-09-28. There are no implementation tasks authorized yet.

After explicit spec approval, write a separate English implementation plan with independently testable phases, exact file targets, automated verification and user test instructions. Every phase ends with user validation and a stop. Design/plan approval does not authorize implementation, later phases, dependency installation, commits, pushes or release publication. Start implementation only after a separate user request.
