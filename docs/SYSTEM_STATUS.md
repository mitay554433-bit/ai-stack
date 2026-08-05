# Confirmed system status

## Implemented and tested

- Go standard-library runtime compiles.
- Unit tests pass.
- COSL event hashes and previous-hash chain verify.
- Unique sources are gzip-compressed and SHA-256 addressed.
- Dropzone inputs are deleted after successful capture.
- Duplicate sources do not create duplicate EmergION candidates.
- One candidate event is persisted at GOV.
- HUMAN_FINAL decisions are enforced.
- Approval produces a separate REG receipt.
- FIELD rebuild and static projections work.
- CPU and FIELD metrics work.
- Orphan evidence cleanup works.
- The Gemma CLI integration completed an end-to-end test using a controlled fake CLI response.

## Not yet verified here

- The dodecahedral metadata hardening and exact REG-to-approval lineage checks
  compile and pass tests in a Go 1.23 build environment.
- Real Gemma inference against the user's actual model.
- The model path and `llama-cli` path on the user's device.
- Native Android operation without Termux.
- Live email, payments, CRM, store, grant, patent, or M&A adapters.

The repository does not claim those capabilities are active.
