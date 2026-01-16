# Review Phase Stability

## Metadata
```yaml
id: review-phase-stability
owner: autocodex
status: draft
created: 2026-01-16
updated: 2026-01-16
```

## Problem statement
Autocodex runs are stalling in the review phase and being auto-finalized as stale without producing review artifacts. This obscures root cause analysis and undermines confidence in the autonomy loop. We need deterministic evidence capture and phase-level guardrails so review runs either complete or fail with actionable diagnostics.

## Goals
- Guarantee review phase output and error capture even if the codex child exits early.
- Add prompt size and context size diagnostics to identify oversized review prompts.
- Provide phase-specific timeout and guardrail behavior so review has a longer window but still terminates cleanly.
- Produce deterministic artifacts for postmortem debugging.

## Non-goals
- Changing the Codex model or reasoning policy defaults.
- UI/UX changes outside of surfacing existing artifacts.
- Adding new external dependencies.

## Requirements
### Functional
- Stream `stdout` and `stderr` to phase artifacts while the codex process is running.
- Record review prompt size (bytes) and snapshot size in artifacts/logs.
- Support per-phase idle timeout overrides (review > implement > plan).
- If the review prompt exceeds a configured threshold, emit a `review-skipped.txt` artifact and continue.

### Non-functional
- No regressions in existing phases (ideate/plan/implement/test).
- Zero additional external services or secrets.
- All new outputs are deterministic and local to the run directory.

## Interfaces / data contracts
- Update `autocodex.yaml` schema/expectations with new phase timeout and prompt guardrail settings.
- Preserve existing autonomy tasks/action schemas.

## Acceptance criteria
- Review phase always writes at least one artifact (`review-stdout.txt`, `review-stderr.txt`, or `review-skipped.txt`).
- Prompt size and snapshot size are captured per review execution.
- Review no longer ends as `stale_after_180s` when the child process is alive.
- Unit tests cover streaming output and prompt guardrail behavior.

## Open questions
- Should prompt-size guardrail be enforced for phases beyond review?
- What is the default max review prompt size (bytes)?

## Risks
- Overly strict guardrails could skip legitimate reviews.
- Longer timeouts may extend loop duration in failure cases.

## References
- `docs/contracts/autonomy-tasks.schema.json`
