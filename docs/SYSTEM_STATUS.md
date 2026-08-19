# Confirmed system status

## Implemented and tested

- Go standard-library sovereign runtime compiles and the full test suite passes.
- `go vet ./...` passes.
- COSL event hashes and previous-hash chain verify.
- Unique sources are gzip-compressed and SHA-256 addressed.
- Duplicate source hashes do not create duplicate EmergION candidates.
- Dropzone intake is governed by the existing capture workflow.
- Candidate EmergIONs enter GOV only after COVERAGE, PROTECTOR, RECOIL, and WVC admission.
- HUMAN_FINAL decisions are enforced.
- HUMAN_FINAL RETURN supports controlled rework with exact predecessor lineage.
- Approval produces a separate REG receipt linked to the approving decision.
- REG-accepted truth and non-authoritative FIELD projections remain distinct.
- FIELD rebuild and static JSON/HTML projections work.
- Accepted Kin lineage traverses REG-accepted ancestry and preserves a HUMAN_FINAL RETURNED predecessor as a governed historical boundary.
- Runtime-owned source identity and execution lineage relationships cannot be supplied by the model.
- Exact execution lineage binds EmergION ID, source hash, authorization event when present, authority, adapter, and action.
- Execution results re-enter the governed EmergION pipeline as bounded execution signals.
- Governed execution-result RECAPTURE is proven.
- Archonym semantic identity is represented in governed metadata.
- PRMs crystallize from REG-accepted state.
- Explicit governed COMPOSITION_KIN relationships derive SAAB structures.
- CPSL is compiled deterministically from governed SAAB composition.
- SAWs are extracted from governed composition.
- LIB indexes derived SAWs without becoming authority.
- Commercial/monetization metadata propagates through the governed PRM → SAAB → SAW → LIB projection path.
- Deterministic SAWSource representation is implemented and remains non-authoritative.
- SAWSource → Capture → GOV/HUMAN_FINAL → REG → PrepareExecution → ExecutionResult → CaptureGovernedExecutionResult → G circulation is proven.
- CPU and FIELD metrics work.
- Orphan evidence cleanup works.
- Real local Gemma inference is operational on this device.
- `llama-cli` is available at `/data/data/com.termux/files/usr/bin/llama-cli`.
- The active Gemma model is `/data/data/com.termux/files/home/models/gemma-2-2b-it.Q4_K_M.gguf`.
- `field doctor` reports the local runtime, COSL chain, evidence store, Gemma binary, and Gemma model ready.
- Native release binaries build for Linux AMD64, Linux ARM64, and Windows AMD64.
- `pkg/fieldapi` provides the existing native embedding seam.
- The canonical Git bundle records complete repository history.

## Operationally proven

- A real repository source was processed through the live local Gemma path.
- The source evidence was preserved and verified.
- Weak semantic extraction was delivered to GOV rather than silently promoted.
- HUMAN_FINAL returned that candidate for controlled rework instead of admitting weak meaning into REG.
- A live RETURNED-predecessor projection conflict was detected during the operational run.
- The existing Accepted Kin projection was corrected to preserve a governed RETURNED predecessor as historical lineage while keeping the accepted successor as the accepted Kin root.
- The corrected live ledger now renders successfully and verifies without changing prior governed events.

## Not yet verified here

- Native Android operation without Termux.
- A native Android shell using `pkg/fieldapi`.
- A platform-packaged local inference engine for Termux-free Android deployment.
- Live external email, payments, CRM, store, grant, patent, or M&A adapters unless separately proven through governed execution.

The repository does not claim unverified external capabilities are active.
