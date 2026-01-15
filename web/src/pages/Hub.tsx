import { useCallback, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { WorkspaceSummary } from '../api/client'
import { useAsync } from '../hooks/useAsync'
import { usePolling } from '../hooks/usePolling'
import { formatTimestamp, statusLabel } from '../utils/format'

type LoadState = 'idle' | 'loading' | 'ready' | 'error'

const workspaceStatus: Record<WorkspaceSummary['status'], string> = {
  ok: 'Ready',
  missing_config: 'Missing config',
  invalid_config: 'Invalid config',
  state_error: 'State error',
}

export default function Hub() {
  const [workspaces, setWorkspaces] = useState<WorkspaceSummary[]>([])
  const [state, setState] = useState<LoadState>('idle')
  const [error, setError] = useState<string | null>(null)
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null)
  const [autoRefresh, setAutoRefresh] = useState(false)

  const pollConfig = {
    intervalMs: 12000,
    maxIntervalMs: 45000,
    backoffFactor: 1.6,
  }

  const refresh = useCallback(
    async ({ signal, silent }: { signal?: AbortSignal; silent?: boolean } = {}) => {
      if (signal?.aborted) return
      if (!silent) setState('loading')
      setError(null)
      try {
        const payload = await api.hubWorkspaces({ signal })
        if (signal?.aborted) return
        setWorkspaces(payload)
        setState('ready')
        setLastUpdated(new Date())
      } catch (err) {
        if (signal?.aborted) return
        if (err instanceof DOMException && err.name === 'AbortError') return
        const message = err instanceof Error ? err.message : 'Unknown error'
        setError(message)
        setState('error')
      }
    },
    [],
  )

  useAsync((signal) => refresh({ signal }), [refresh])

  const pollDelayMs = usePolling(
    (signal) => refresh({ signal, silent: true }),
    { enabled: autoRefresh, ...pollConfig },
  )

  const sortedWorkspaces = useMemo(() => {
    return [...workspaces].sort((a, b) => a.name.localeCompare(b.name))
  }, [workspaces])

  return (
    <div className="page">
      <header className="detail-header">
        <div>
          <h1>Hub workspaces</h1>
          <p>Monitor multiple autocodex repos from a single dashboard.</p>
        </div>
        <div className="detail-actions">
          <div className="refresh-controls">
            <label className="toggle">
              <input
                type="checkbox"
                name="hub-auto-refresh"
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
      </header>

      {error && (
        <div className="error" role="alert">
          {error}
        </div>
      )}

      <section className="runs" aria-live="polite">
        <div className="runs__header">
          <h2>Configured repos</h2>
          <p>Each workspace reads run state from its local store.</p>
        </div>
        {sortedWorkspaces.length === 0 ? (
          <div className="empty">
            <h3>No workspaces configured</h3>
            <p>Add hub.workspaces entries to autocodex.yaml to start tracking repos.</p>
          </div>
        ) : (
          <div className="runs__grid">
            {sortedWorkspaces.map((workspace) => (
              <Link
                key={workspace.id}
                to={`/hub/${workspace.id}`}
                className={`card card--run card--${workspace.last_run_status ?? 'idle'}`}
              >
                <div className="card__header">
                  <div>
                    <p className="card__eyebrow">Workspace</p>
                    <h3 className="card__title">{workspace.name}</h3>
                  </div>
                  <span className={`pill pill--${workspace.status}`}>{workspaceStatus[workspace.status]}</span>
                </div>
                <dl className="card__meta">
                  <div>
                    <dt>Root</dt>
                    <dd>{workspace.root}</dd>
                  </div>
                  <div>
                    <dt>Runs</dt>
                    <dd>{workspace.runs_count}</dd>
                  </div>
                  <div>
                    <dt>Last status</dt>
                    <dd>{workspace.last_run_status ? statusLabel(workspace.last_run_status) : '—'}</dd>
                  </div>
                  <div>
                    <dt>Last start</dt>
                    <dd>{workspace.last_run_started_at ? formatTimestamp(workspace.last_run_started_at) : '—'}</dd>
                  </div>
                </dl>
                {workspace.error ? <p className="card__error">{workspace.error}</p> : null}
              </Link>
            ))}
          </div>
        )}
      </section>

      <p className="panel__foot">Last refresh {lastUpdated ? formatTimestamp(lastUpdated.toISOString()) : '—'}</p>
    </div>
  )
}
