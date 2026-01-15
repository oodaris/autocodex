import { useCallback, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type {
  Artifact,
  Run,
  RunControlStatus,
  RunEvent,
  SnapshotDetail,
  SnapshotSummary,
} from '../api/client'
import { useAsync } from '../hooks/useAsync'
import { usePolling } from '../hooks/usePolling'
import { formatBytes, formatTimestamp, statusLabel } from '../utils/format'

type LoadState = 'idle' | 'loading' | 'ready' | 'error'

export default function RunDetail() {
  const { runId, workspaceId } = useParams<{ runId: string; workspaceId?: string }>()
  const [run, setRun] = useState<Run | null>(null)
  const [events, setEvents] = useState<RunEvent[]>([])
  const [artifacts, setArtifacts] = useState<Artifact[]>([])
  const [controlStatus, setControlStatus] = useState<RunControlStatus | null>(null)
  const [controlAction, setControlAction] =
    useState<RunControlStatus['last_action'] | 'resume' | 'stop' | 'cancel' | 'kill'>('stop')
  const [controlReason, setControlReason] = useState('')
  const [controlMessage, setControlMessage] = useState<string | null>(null)
  const [controlBusy, setControlBusy] = useState(false)
  const [snapshots, setSnapshots] = useState<SnapshotSummary[]>([])
  const [snapshotDetail, setSnapshotDetail] = useState<SnapshotDetail | null>(null)
  const [snapshotReason, setSnapshotReason] = useState('')
  const [snapshotBusy, setSnapshotBusy] = useState(false)
  const [snapshotMessage, setSnapshotMessage] = useState<string | null>(null)
  const [state, setState] = useState<LoadState>('idle')
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
      if (!runId) {
        setState('error')
        setError('Run ID is missing.')
        return
      }
      if (signal?.aborted) return
      if (!silent) setState('loading')
      setError(null)
      try {
        const workspace = workspaceId?.trim()
        const [runPayload, eventPayload, artifactPayload, controlPayload, snapshotPayload] = await Promise.all([
          workspace ? api.hubRun(workspace, runId, { signal }) : api.run(runId, { signal }),
          workspace ? api.hubRunEvents(workspace, runId, { signal }) : api.runEvents(runId, { signal }),
          workspace ? api.hubRunArtifacts(workspace, runId, { signal }) : api.runArtifacts(runId, { signal }),
          workspace
            ? api.hubRunControlStatus(workspace, runId, { signal })
            : api.runControlStatus(runId, { signal }),
          workspace ? api.hubRunSnapshots(workspace, runId, { signal }) : api.runSnapshots(runId, { signal }),
        ])
        if (signal?.aborted) return
        setRun(runPayload)
        setEvents(eventPayload)
        setArtifacts(artifactPayload)
        setControlStatus(controlPayload)
        setSnapshots(snapshotPayload)
        setLastUpdated(new Date())
        setState('ready')
      } catch (err) {
        if (signal?.aborted) return
        if (err instanceof DOMException && err.name === 'AbortError') return
        const message = err instanceof Error ? err.message : 'Unknown error'
        setError(message)
        setState('error')
      }
    },
    [runId, workspaceId],
  )

  useAsync((signal) => refresh({ signal }), [refresh])

  const pollDelayMs = usePolling(
    (signal) => refresh({ signal, silent: true }),
    { enabled: autoRefresh, ...pollConfig },
  )

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
          <Link className="link" to={workspaceId ? `/hub/${workspaceId}` : '/'}>
            ← Back to runs
          </Link>
          <h1>Run details</h1>
          <p>Inspect events and artifacts captured during this loop.</p>
        </div>
        <div className="detail-actions">
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
              <div>
                <dt>Last updated</dt>
                <dd>{lastUpdated ? formatTimestamp(lastUpdated.toISOString()) : '—'}</dd>
              </div>
            </dl>
          </div>

          <div className="panel panel--note">
            <div className="panel__header">
              <h2>Run control</h2>
              <span className="panel__meta">
                {controlStatus?.last_action ? `Last: ${controlStatus.last_action}` : 'No actions yet'}
              </span>
            </div>
            <div className="control-grid">
              <div>
                <p className="control-label">Action</p>
                <div className="control-row">
                  <select
                    className="control-select"
                    value={controlAction ?? 'stop'}
                    onChange={(event) => setControlAction(event.target.value as typeof controlAction)}
                    disabled={controlBusy}
                  >
                    <option value="resume">Resume</option>
                    <option value="stop">Stop</option>
                    <option value="cancel">Cancel</option>
                    <option value="kill">Kill</option>
                  </select>
                  <button
                    className="button button--danger"
                    type="button"
                    disabled={controlBusy || !runId}
                    onClick={async () => {
                      if (!runId) return
                      setControlBusy(true)
                      setControlMessage(null)
                      try {
                        const payload = {
                          action: controlAction as 'resume' | 'stop' | 'cancel' | 'kill',
                          reason: controlReason.trim() || undefined,
                        }
                      const workspace = workspaceId?.trim()
                      const response = workspace
                        ? await api.hubRunControlAction(workspace, runId, payload)
                        : await api.runControlAction(runId, payload)
                      setControlMessage(`Requested ${response.action}. ${response.message}`)
                      const latest = workspace
                        ? await api.hubRunControlStatus(workspace, runId)
                        : await api.runControlStatus(runId)
                        setControlStatus(latest)
                      } catch (err) {
                        const message = err instanceof Error ? err.message : 'Failed to send control action'
                        setControlMessage(message)
                      } finally {
                        setControlBusy(false)
                      }
                    }}
                  >
                    {controlBusy ? 'Sending…' : 'Send'}
                  </button>
                </div>
              </div>
              <div>
                <p className="control-label">Reason (optional)</p>
                <input
                  className="control-input"
                  type="text"
                  value={controlReason}
                  onChange={(event) => setControlReason(event.target.value)}
                  placeholder="Why should this run stop?"
                  disabled={controlBusy}
                />
              </div>
              <div className="control-meta">
                <p>
                  <strong>Status:</strong> {controlStatus?.status ?? run.status}
                </p>
                <p>
                  <strong>Last action:</strong> {controlStatus?.last_action ?? '—'}
                </p>
                <p>
                  <strong>Last action at:</strong>{' '}
                  {controlStatus?.last_action_at ? formatTimestamp(controlStatus.last_action_at) : '—'}
                </p>
              </div>
              {controlMessage ? <p className="control-message">{controlMessage}</p> : null}
            </div>
          </div>

          <div className="panel panel--note">
            <div className="panel__header">
              <h2>Snapshots</h2>
              <span className="panel__meta">{snapshots.length} saved</span>
            </div>
            <div className="control-grid">
              <div>
                <p className="control-label">Snapshot reason (optional)</p>
                <input
                  className="control-input"
                  type="text"
                  value={snapshotReason}
                  onChange={(event) => setSnapshotReason(event.target.value)}
                  placeholder="Why capture this snapshot?"
                  disabled={snapshotBusy}
                />
              </div>
              <div className="control-row">
                <button
                  className="button"
                  type="button"
                  disabled={snapshotBusy || !runId}
                  onClick={async () => {
                    if (!runId) return
                    setSnapshotBusy(true)
                    setSnapshotMessage(null)
                    try {
                      const payload = snapshotReason.trim() ? { reason: snapshotReason.trim() } : {}
                      const workspace = workspaceId?.trim()
                      const response = workspace
                        ? await api.hubCreateSnapshot(workspace, runId, payload)
                        : await api.createSnapshot(runId, payload)
                      setSnapshotDetail(response)
                      setSnapshotMessage(`Snapshot ${response.summary.id} created.`)
                      const latest = workspace
                        ? await api.hubRunSnapshots(workspace, runId)
                        : await api.runSnapshots(runId)
                      setSnapshots(latest)
                    } catch (err) {
                      const message = err instanceof Error ? err.message : 'Failed to create snapshot'
                      setSnapshotMessage(message)
                    } finally {
                      setSnapshotBusy(false)
                    }
                  }}
                >
                  {snapshotBusy ? 'Creating…' : 'Create snapshot'}
                </button>
              </div>
              {snapshotMessage ? <p className="control-message">{snapshotMessage}</p> : null}
              {snapshots.length === 0 ? (
                <p className="empty">No snapshots yet.</p>
              ) : (
                <ul className="list">
                  {snapshots.map((snapshot) => (
                    <li key={snapshot.id} className="list__item list__item--clickable">
                      <button
                        className="ghost-button"
                        type="button"
                        onClick={async () => {
                          if (!runId) return
                          try {
                            const workspace = workspaceId?.trim()
                            const detail = workspace
                              ? await api.hubSnapshot(workspace, runId, snapshot.id)
                              : await api.snapshot(runId, snapshot.id)
                            setSnapshotDetail(detail)
                          } catch (err) {
                            const message =
                              err instanceof Error ? err.message : 'Failed to load snapshot detail'
                            setSnapshotMessage(message)
                          }
                        }}
                      >
                        <div>
                          <p className="list__title">{snapshot.id}</p>
                          <p className="list__meta">{snapshot.reason || 'No reason provided'}</p>
                        </div>
                        <div className="list__meta">
                          <span>{formatBytes(snapshot.size_bytes)}</span>
                          <span>{formatTimestamp(snapshot.created_at)}</span>
                        </div>
                      </button>
                    </li>
                  ))}
                </ul>
              )}
              {snapshotDetail ? (
                <div className="snapshot-detail">
                  <h3>Snapshot detail</h3>
                  <div className="snapshot-meta">
                    <span>Events: {snapshotDetail.manifest.events}</span>
                    <span>Artifacts: {snapshotDetail.manifest.artifacts}</span>
                    <span>Memory docs: {snapshotDetail.manifest.memory_docs}</span>
                    <span>Bytes: {snapshotDetail.manifest.bytes}</span>
                  </div>
                  <pre className="snapshot-content">{snapshotDetail.content}</pre>
                </div>
              ) : null}
            </div>
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
