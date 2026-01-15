import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { Artifact, Run, RunEvent } from '../api/client'
import { formatBytes, formatTimestamp, statusLabel } from '../utils/format'

type LoadState = 'idle' | 'loading' | 'ready' | 'error'

export default function RunDetail() {
  const { runId } = useParams<{ runId: string }>()
  const [run, setRun] = useState<Run | null>(null)
  const [events, setEvents] = useState<RunEvent[]>([])
  const [artifacts, setArtifacts] = useState<Artifact[]>([])
  const [state, setState] = useState<LoadState>('idle')
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async (signal?: AbortSignal) => {
    if (!runId) {
      setState('error')
      setError('Run ID is missing.')
      return
    }
    if (signal?.aborted) return
    setState('loading')
    setError(null)
    try {
      const [runPayload, eventPayload, artifactPayload] = await Promise.all([
        api.run(runId, { signal }),
        api.runEvents(runId, { signal }),
        api.runArtifacts(runId, { signal }),
      ])
      if (signal?.aborted) return
      setRun(runPayload)
      setEvents(eventPayload)
      setArtifacts(artifactPayload)
      setState('ready')
    } catch (err) {
      if (signal?.aborted) return
      if (err instanceof DOMException && err.name === 'AbortError') return
      const message = err instanceof Error ? err.message : 'Unknown error'
      setError(message)
      setState('error')
    }
  }, [runId])

  useEffect(() => {
    const controller = new AbortController()
    // eslint-disable-next-line react-hooks/set-state-in-effect -- initial data fetch
    void refresh(controller.signal)
    return () => {
      controller.abort()
    }
  }, [refresh])

  const sortedEvents = useMemo(() => {
    return [...events].sort((a, b) => b.ts.localeCompare(a.ts))
  }, [events])

  const sortedArtifacts = useMemo(() => {
    return [...artifacts].sort((a, b) => b.created_at.localeCompare(a.created_at))
  }, [artifacts])

  return (
    <div className="page">
      <header className="detail-header">
        <div>
          <Link className="link" to="/">
            ← Back to runs
          </Link>
          <h1>Run details</h1>
          <p>Inspect events and artifacts captured during this loop.</p>
        </div>
        <button className="button" type="button" onClick={() => void refresh()} disabled={state === 'loading'}>
          {state === 'loading' ? 'Refreshing…' : 'Refresh'}
        </button>
      </header>

      {error && (
        <div className="error" role="alert">
          {error}
        </div>
      )}

      {run ? (
        <section className="detail-grid">
          <div className="panel">
            <div className="panel__header">
              <h2>Run metadata</h2>
              <span className={`pill pill--${run.status}`}>{statusLabel(run.status)}</span>
            </div>
            <dl className="detail-meta">
              <div>
                <dt>Run ID</dt>
                <dd>{run.id}</dd>
              </div>
              <div>
                <dt>Status</dt>
                <dd>{statusLabel(run.status)}</dd>
              </div>
              <div>
                <dt>Phase</dt>
                <dd>{run.current_phase || '—'}</dd>
              </div>
              <div>
                <dt>Iterations</dt>
                <dd>{run.iterations}</dd>
              </div>
              <div>
                <dt>Started</dt>
                <dd>{formatTimestamp(run.started_at)}</dd>
              </div>
              <div>
                <dt>Finished</dt>
                <dd>{formatTimestamp(run.finished_at)}</dd>
              </div>
            </dl>
          </div>

          <div className="panel panel--note">
            <h2>Artifacts</h2>
            {sortedArtifacts.length === 0 ? (
              <p className="empty">No artifacts captured yet.</p>
            ) : (
              <ul className="list">
                {sortedArtifacts.map((artifact) => (
                  <li key={artifact.id} className="list__item">
                    <div>
                      <p className="list__title">{artifact.name}</p>
                      <p className="list__meta">{artifact.type}</p>
                    </div>
                    <div className="list__meta">
                      <span>{formatBytes(artifact.size_bytes)}</span>
                      <span>{formatTimestamp(artifact.created_at)}</span>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </section>
      ) : (
        <div className="empty">No run details loaded.</div>
      )}

      <section className="panel">
        <div className="panel__header">
          <h2>Event stream</h2>
          <span className="panel__meta">{sortedEvents.length} events</span>
        </div>
        {sortedEvents.length === 0 ? (
          <p className="empty">No events recorded.</p>
        ) : (
          <div className="event-list">
            {sortedEvents.map((event) => (
              <div key={event.id} className="event">
                <div>
                  <p className="event__type">{event.type.replaceAll('_', ' ')}</p>
                  <p className="event__meta">
                    {event.phase ? `${event.phase} · ` : ''}
                    {formatTimestamp(event.ts)}
                  </p>
                  <p className="event__message">{event.message}</p>
                </div>
                <span className="event__id">{event.id}</span>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  )
}
