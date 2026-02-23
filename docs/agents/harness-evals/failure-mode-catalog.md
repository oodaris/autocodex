# Failure Mode Catalog

1. Missing `ACTIONS` JSON in test phase output -> gate failure.
2. Invalid `ACTIONS` schema payload -> gate failure.
3. High-impact run with `council_verdict != GREEN` -> closure blocked.
4. High-impact run with `critic_verdict != GO` -> closure blocked.
5. Missing lifecycle/admission metadata fields in run events -> verification failure.
6. Harness lint detects policy drift -> preflight fails.

Expected behaviors:
- deterministic error reason,
- bead status transitions to `blocked` when closure conditions are unmet,
- no false success claims.
