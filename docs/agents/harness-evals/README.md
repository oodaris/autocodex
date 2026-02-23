# Harness Eval Suite (Deterministic)

Use this suite to measure Harness-v2 quality deltas after any change to:
1. `.codex/config.toml`
2. `.codex/agents/*.toml`
3. harness preflight/lint scripts
4. autonomy gate policy and action schema

## Components
1. `golden-task-catalog.md`
2. `failure-mode-catalog.md`

## Run protocol
1. Execute golden scenarios in fixed order.
2. Execute failure scenarios in fixed order.
3. Record outcomes with stable run IDs.
4. Compare against prior baseline.
5. Reject policy changes when critical regression appears.

## Required score dimensions
1. Task success rate
2. First-pass gate pass rate
3. Retry/admission stability
4. Council/critic gate consistency (high-impact mode)
5. Evidence completeness and traceability

## CI strict defaults
1. `HARNESS_EVAL_MIN_PASS_RATE=1.0`
2. `HARNESS_EVAL_MIN_SCENARIOS=6`
3. `HARNESS_EVAL_MAX_SOFT_FAILURES=0`
