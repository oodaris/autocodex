# Parallelism, Collaboration, and Subagents

This guide explains how autocodex achieves parallelism and collaboration, how it
relates to Codex collaboration modes, and how to structure work with beads,
roles, and tasks.

## TL;DR
- **Guaranteed parallelism** comes from the **autocodex coordinator** (multiple
  Codex CLI processes in parallel).
- **Codex collaboration_mode/preset** controls **in‑process collaboration**
  (role‑style behaviors), not OS‑level parallelism.
- Codex `multi_agent` is an optional in-process capability, not the scheduler of record.
- Use `--swarm` or `autonomy.coordinator.enabled: true` for parallel bead runs.
- Logs will print **"parallel agents launched"** when true parallelism starts.

## Two layers of “multi‑agent” behavior

### 1) autocodex coordinator (real parallelism)
autocodex can run **multiple Codex CLI processes concurrently**. Each bead is
handled by a separate process, scheduled by the OS across CPU cores.

Enable with config:
```yaml
autonomy:
  enabled: true
  coordinator:
    enabled: true
    strategy: bead   # or phase
    max_parallel: 4  # 0 = unlimited
    fail_fast: false
```

Or force it per run:
```bash
autocodex run --swarm --task "Run all unblocked beads in parallel"
```
Note: `--swarm` sets `autonomy.coordinator.max_parallel` to `0` (unlimited) for the run.

**What you get**
- Multiple Codex processes running at once.
- Per‑bead memory isolation: `memory/beads/<id>/`.
- Clear log signal: `parallel agents launched`.

**When to use**
- Multiple independent beads (no dependencies).
- You want guaranteed parallelism now.

### 2) Codex collaboration_mode/preset (in‑process collaboration)
These settings influence how **one Codex process** organizes its internal work.
Think **roles, critique, planning style**—not separate OS processes.

Defaults (unless overridden):
```yaml
codex:
  collaboration_enabled: true
  collaboration_mode: auto
  preset: default
```

CLI overrides:
```bash
autocodex run --collaboration-mode auto --preset default --task "..."
```

Disable collaboration for a single run:
```bash
autocodex run --no-collaboration --task "..."
```

**What you get**
- Role‑like behaviors inside a single Codex run.
- Potentially “subagent” style reasoning, but **not guaranteed parallelism**.
- Recent Codex releases add an explorer role and max-depth guardrails; if you need deeper decomposition, split work into beads or use the coordinator.

**When to use**
- You want richer reasoning or structured collaboration inside a single run.
- You do **not** need true OS‑level parallelism.

### Plan/Pair Modes (Interactive Codex UX)
Codex also supports interactive modes like Plan and Pair. autocodex does **not**
invoke these modes: it runs `codex exec` (non-interactive) under the hood.

Practical implications:
- Tools like `request_user_input` are only available in interactive Plan/Pair.
- If your prompt is underspecified, Codex may request follow-up; in non-interactive
  runs this can fail. Prefer self-contained tasks, or run Codex interactively to
  clarify first and then paste the refined task into autocodex.

## Swarm vs Collaboration: choosing the right tool

| Goal | Use |
| --- | --- |
| Guaranteed parallel execution | autocodex coordinator / `--swarm` |
| Role‑based collaboration inside one run | Codex `collaboration_mode`/`preset` |
| Optional in-process decomposition (Codex features) | Codex `multi_agent` (non-scheduling) |
| Both | Enable coordinator **and** keep collaboration defaults |

## Harness policy and high-impact work

Harness mode does not replace scheduling; it adds closure policy and evidence gates.

- Enable with `autonomy.harness.enabled: true`
- Preflight command: `autocodex harness preflight --strict`
- Lint command: `python3 scripts/harness_config_lint.py`
- In `impact_mode: high`, ACTIONS closure must include:
  - `council_verdict: GREEN`
  - `critic_verdict: GO`
  - `quality_gate_passed: true`

## Beads, roles, and tasks

autocodex uses **beads** as the unit of work in autonomy mode. Each bead
represents a task with dependencies. This aligns well with parallel execution:

- **Multiple ready beads** ⇒ run in parallel (coordinator)
- **Single ready bead** ⇒ run sequentially

If you want explicit role behavior, encode it in the task or plan:
```
Role: Reviewer
Goal: Validate edge cases and tests for the feature
```

Then let Codex collaboration modes **or** skill prompts enforce the role.

## Coordinator strategies

### `strategy: bead` (recommended)
Runs beads in parallel while sharing spec/plan artifacts.

### `strategy: phase`
Runs phases (ideate/plan/implement/...) in parallel and **does not share**
artifacts between phases. Use only when phases are truly independent.

## Resource tuning

Parallel runs can saturate CPU and memory.

- Start with `max_parallel: 3` or `4`
- Increase only if the host has spare CPU/memory
- Use `fail_fast: true` to stop all on first error if you prefer strict gating

## Logging: how to verify parallelism

When the coordinator launches parallel workers, you will see:
```
parallel agents launched
```
This message only appears when more than one worker is started (e.g. multiple
ready beads). If only one bead is ready, the coordinator still runs but the
parallel log line will not appear.

Additionally, you can confirm multiple Codex processes:
```bash
pgrep -af codex
```

## Recommended DX for AI Engineers

Use a separate guide (this file) plus a short pointer in README:
- **README**: Short “Parallelism & collaboration” section with a link.
- **This guide**: Deep explanation, tradeoffs, and examples.

Suggested quick recipe:
```bash
autocodex bootstrap
autocodex run --swarm --task "Implement all unblocked beads"
```

## FAQ

**Q: If Codex “spawns subagents,” is that parallel?**  
Not necessarily. Codex may simulate roles inside one process. If you need
guaranteed parallelism, use the autocodex coordinator.

**Q: Can I run parallel beads and still use collaboration presets?**  
Yes. Each bead run can still use Codex collaboration_mode/preset.

**Q: Should we add a separate guide?**  
Yes. Parallelism, collaboration, and subagents are easy to confuse. A dedicated
guide prevents misaligned expectations for AI engineers.
