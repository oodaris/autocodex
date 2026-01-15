import { useCallback, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { Run, WorkspaceSummary } from '../api/client'
import { useAsync } from '../hooks/useAsync'
import { usePolling } from '../hooks/usePolling'
import { formatTimestamp, statusLabel } from '../utils/format'

type LoadState = 'idle' | 'loading' | 'ready' | 'error'

type HealthState = {
  status: 'unknown' | 'ok' | 'error'
  message?: string
  time?: string
}

type RunCardProps = {
  run: Run
  href: string
}

function RunCard({ run, href }: RunCardProps) {
  return (
    <Link to={href} className={`card card--run card--${run.status}`}>
      <div className="card__header">
        <div>
          <p className="card__eyebrow">Run ID</p>
          <h3 className="card__title">{run.id}</h3>
        </div>
        <span className={`pill pill--${run.status}`}>{statusLabel(run.status)}</span>
      </div>
      <dl className="card__meta">
        <div>
          <dt>Phase</dt>
          <dd>{run.current_phase || '—'}</dd>
        </div>
        <div>
          <dt>Started</dt>
          <dd>{formatTimestamp(run.started_at)}</dd>
        </div>
        <div>
          <dt>Finished</dt>
          <dd>{formatTimestamp(run.finished_at)}</dd>
        </div>
        <div>
          <dt>Iterations</dt>
          <dd>{run.iterations}</dd>
        </div>
      </dl>
    </Link>
  )
}

export default function Dashboard() {
  const { workspaceId } = useParams<{ workspaceId?: string }>()
  const [runs, setRuns] = useState<Run[]>([])
  const [workspace, setWorkspace] = useState<WorkspaceSummary | null>(null)
  const [state, setState] = useState<LoadState>('idle')
  const [health, setHealth] = useState<HealthState>({ status: 'unknown' })
  const [error, setError] = useState<string | null>(null)
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null)
  const [autoRefresh, setAutoRefresh] = useState(false)

  const pollConfig = {
    intervalMs: 8000,
    maxIntervalMs: 30000,
    backoffFactor: 1.6,
  }

  const refresh = useCallback(
    async ({ signal, silent }: { signal?: AbortSignal; silent?: boolean } = {}) => {
      if (signal?.aborted) return
      if (!silent) setState('loading')
      setError(null)

      try {
        const workspace = workspaceId?.trim()
        const [healthPayload, runsPayload, workspacePayload] = await Promise.all([
          api.health({ signal }),
          workspace ? api.hubRuns(workspace, { signal }) : api.runs({ signal }),
          workspace ? api.hubWorkspace(workspace, { signal }) : Promise.resolve(null),
        ])
        if (signal?.aborted) return
        setHealth({ status: 'ok', time: healthPayload.time })
        setRuns(runsPayload)
        setWorkspace(workspacePayload)
        setLastUpdated(new Date())
        setState('ready')
      } catch (err) {
        if (signal?.aborted) return
        if (err instanceof DOMException && err.name === 'AbortError') return
        const message = err instanceof Error ? err.message : 'Unknown error'
        setHealth({ status: 'error', message })
        setError(message)
        setState('error')
      }
    },
    [workspaceId],
  )

  useAsync((signal) => refresh({ signal }), [refresh])

  const pollDelayMs = usePolling(
    (signal) => refresh({ signal, silent: true }),
    { enabled: autoRefresh, ...pollConfig },
  )

  const sortedRuns = useMemo(() => {
    return [...runs].sort((a, b) => b.started_at.localeCompare(a.started_at))
  }, [runs])

  const summary = useMemo(() => {
    const stats = {
      running: 0,
      failed: 0,
      completed: 0,
    }
    for (const run of runs) {
      if (run.status === 'running') stats.running += 1
      else if (run.status === 'failed') stats.failed += 1
      else if (run.status === 'completed') stats.completed += 1
    }
    return stats
  }, [runs])

  return (
    <div className="page">
      <header className="hero hero--banner">
        <div className="hero__banner" aria-hidden="true" />
        <div className="hero__copy">
          <span className="hero__badge">autocodex Control Deck</span>
          {workspaceId ? (
            <>
              <h1>Workspace {workspace?.name ?? workspaceId}</h1>
              <p>Review recent runs and loop activity for this workspace.</p>
            </>
          ) : (
            <>
              <h1>Keep the loop moving. See every run in one place.</h1>
              <p>
                This dashboard reads from your local autocodex API and surfaces run history, phases, and
                artifacts. Keep it open while autocodex iterates.
              </p>
            </>
          )}
        </div>
        <div className="hero__status">
          <div className={`status-card status-card--${health.status}`} role="status" aria-live="polite">
            <p className="status-card__label">API status</p>
            <h2>
              {health.status === 'ok' ? 'Connected' : health.status === 'error' ? 'Disconnected' : 'Checking'}
            </h2>
            <p className="status-card__meta">
              {health.status === 'ok' && health.time ? `Last ping ${formatTimestamp(health.time)}` : api.baseUrl}
            </p>
            {health.status === 'error' && health.message && (
              <p className="status-card__error">{health.message}</p>
            )}
            <div className="refresh-controls">
              <label className="toggle">
                <input
                  type="checkbox"
                  checked={autoRefresh}
                  onChange={(event) => setAutoRefresh(event.target.checked)}
                />
                <span>Auto-refresh</span>
              </label>
              <span className="toggle__meta">
                {autoRefresh ? `Every ${Math.round(pollDelayMs / 1000)}s` : 'Off'}
              </span>
            </div>
            <button className="button" type="button" onClick={() => void refresh()} disabled={state === 'loading'}>
              {state === 'loading' ? 'Refreshing…' : 'Refresh'}
            </button>
          </div>
        </div>
      </header>

      <section className="grid">
        <div className="grid__item">
          <div className="panel">
            <div className="panel__header">
              <h2>Run summary</h2>
              <span className="panel__meta">{runs.length} total</span>
            </div>
            <div className="summary">
              <div>
                <p className="summary__label">Running</p>
                <p className="summary__value">{summary.running}</p>
              </div>
              <div>
                <p className="summary__label">Completed</p>
                <p className="summary__value">{summary.completed}</p>
              </div>
              <div>
                <p className="summary__label">Failed</p>
                <p className="summary__value">{summary.failed}</p>
              </div>
            </div>
            <div className="panel__foot">
              <span>API base</span>
              <code>{api.baseUrl}</code>
            </div>
          </div>
        </div>

        <div className="grid__item">
          <div className="panel panel--note">
            <h2>Loop phases</h2>
            <ul>
              <li>Ideate → Plan → Implement → Review → Test</li>
              <li>Artifacts saved per phase in JSONL</li>
              <li>Beads are tracked via the CLI and state store</li>
            </ul>
            <p className="panel__foot">
              Last refresh {lastUpdated ? formatTimestamp(lastUpdated.toISOString()) : '—'}
            </p>
          </div>
        </div>
      </section>

      <section className="runs" aria-live="polite">
        <div className="runs__header">
          <h2>Recent runs</h2>
          <p>Newest first. Click a run to open details.</p>
        </div>
        {error && (
          <div className="error" role="alert">
            {error}
          </div>
        )}
        <div className="runs__grid">
          {sortedRuns.length === 0 ? (
            <div className="empty">
              <h3>No runs yet</h3>
              <p>
                Start with <code>autocodex run</code> to populate the timeline.
              </p>
            </div>
          ) : (
            sortedRuns
              .slice(0, 6)
              .map((run) => (
                <RunCard
                  key={run.id}
                  run={run}
                  href={workspaceId ? `/hub/${workspaceId}/runs/${run.id}` : `/runs/${run.id}`}
                />
              ))
          )}
        </div>
      </section>
    </div>
  )
}
