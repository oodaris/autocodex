import { useCallback, useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { MemoryDocDetail, MemoryDocSummary } from '../api/client'
import { useAsync } from '../hooks/useAsync'
import { formatBytes, formatTimestamp } from '../utils/format'

type LoadState = 'idle' | 'loading' | 'ready' | 'error'

export default function MemoryDocs() {
  const { workspaceId } = useParams<{ workspaceId?: string }>()
  const [docs, setDocs] = useState<MemoryDocSummary[]>([])
  const [selectedName, setSelectedName] = useState<string | null>(null)
  const [activeDoc, setActiveDoc] = useState<MemoryDocDetail | null>(null)
  const [progressDoc, setProgressDoc] = useState<MemoryDocDetail | null>(null)
  const [listState, setListState] = useState<LoadState>('idle')
  const [detailState, setDetailState] = useState<LoadState>('idle')
  const [progressState, setProgressState] = useState<LoadState>('idle')
  const [error, setError] = useState<string | null>(null)

  const refreshList = useCallback(async ({ signal }: { signal?: AbortSignal } = {}) => {
    if (signal?.aborted) return
    setListState('loading')
    setError(null)
    try {
      const workspace = workspaceId?.trim()
      const payload = workspace ? await api.hubMemoryDocs(workspace, { signal }) : await api.memoryDocs({ signal })
      if (signal?.aborted) return
      setDocs(payload)
      setListState('ready')
      if (!selectedName || !payload.find((doc) => doc.name === selectedName)) {
        setSelectedName(payload[0]?.name ?? null)
      }
    } catch (err) {
      if (signal?.aborted) return
      if (err instanceof DOMException && err.name === 'AbortError') return
      const message = err instanceof Error ? err.message : 'Unknown error'
      setError(message)
      setListState('error')
    }
  }, [selectedName, workspaceId])

  const refreshDetail = useCallback(
    async ({ signal }: { signal?: AbortSignal } = {}) => {
      if (!selectedName) {
        setActiveDoc(null)
        return
      }
      if (signal?.aborted) return
      setDetailState('loading')
      setError(null)
      try {
        const workspace = workspaceId?.trim()
        const payload = workspace
          ? await api.hubMemoryDoc(workspace, selectedName, { signal })
          : await api.memoryDoc(selectedName, { signal })
        if (signal?.aborted) return
        setActiveDoc(payload)
        setDetailState('ready')
      } catch (err) {
        if (signal?.aborted) return
        if (err instanceof DOMException && err.name === 'AbortError') return
        const message = err instanceof Error ? err.message : 'Unknown error'
        setError(message)
        setDetailState('error')
      }
    },
    [selectedName, workspaceId],
  )

  useAsync((signal) => refreshList({ signal }), [refreshList])
  useAsync((signal) => refreshDetail({ signal }), [refreshDetail])

  const progressName = useMemo(() => {
    const doc = docs.find((item) => item.name.toLowerCase() === 'progress.md')
    return doc?.name ?? null
  }, [docs])

  const refreshProgress = useCallback(
    async ({ signal }: { signal?: AbortSignal } = {}) => {
      if (!progressName) {
        setProgressDoc(null)
        return
      }
      if (signal?.aborted) return
      setProgressState('loading')
      try {
        const workspace = workspaceId?.trim()
        const payload = workspace
          ? await api.hubMemoryDoc(workspace, progressName, { signal })
          : await api.memoryDoc(progressName, { signal })
        if (signal?.aborted) return
        setProgressDoc(payload)
        setProgressState('ready')
      } catch (err) {
        if (signal?.aborted) return
        if (err instanceof DOMException && err.name === 'AbortError') return
        const message = err instanceof Error ? err.message : 'Unknown error'
        setError(message)
        setProgressState('error')
      }
    },
    [progressName, workspaceId],
  )

  useAsync((signal) => refreshProgress({ signal }), [refreshProgress])

  const sortedDocs = useMemo(() => {
    return [...docs].sort((a, b) => a.name.localeCompare(b.name))
  }, [docs])

  const latestSummary = useMemo(() => {
    if (!progressDoc?.content) {
      return null
    }
    const trimmed = progressDoc.content.trim()
    if (!trimmed) {
      return null
    }
    const sections = trimmed.split(/\n##\s+/).filter(Boolean)
    if (sections.length === 0) {
      return { title: 'Latest summary', body: trimmed }
    }
    const last = sections[sections.length - 1]
    const lines = last.split('\n')
    const title = lines[0]?.trim() ?? 'Latest summary'
    const body = lines.slice(1).join('\n').trim()
    return { title, body: body || 'Summary details will appear here.' }
  }, [progressDoc])

  return (
    <div className="page">
      <header className="detail-header">
        <div>
          <h1>Memory docs</h1>
          <p>
            Local markdown notes that autocodex reads and updates during each loop.
            {workspaceId ? ` Workspace: ${workspaceId}.` : ''}
          </p>
        </div>
        <button className="button" type="button" onClick={() => void refreshList()}>
          Refresh list
        </button>
      </header>

      {error && (
        <div className="error" role="alert">
          {error}
        </div>
      )}

      <section className="panel memory-summary">
        <div className="panel__header">
          <h2>Latest summary</h2>
          <span className="panel__meta">
            {progressState === 'loading'
              ? 'Loading…'
              : progressDoc
                ? formatTimestamp(progressDoc.updated_at)
                : 'No updates yet'}
          </span>
        </div>
        {latestSummary ? (
          <>
            <p className="memory-summary__title">{latestSummary.title}</p>
            <pre className="memory-summary__content">{latestSummary.body}</pre>
          </>
        ) : (
          <p className="empty">No run summaries yet. Run autocodex to populate PROGRESS.md.</p>
        )}
      </section>

      <section className="memory-grid">
        <div className="panel memory-list">
          <div className="panel__header">
            <h2>Documents</h2>
            <span className="panel__meta">
              {listState === 'loading' ? 'Loading…' : `${docs.length} docs`}
            </span>
          </div>
          {sortedDocs.length === 0 ? (
            <p className="empty">No memory docs found.</p>
          ) : (
            <ul className="memory-items">
              {sortedDocs.map((doc) => (
                <li key={doc.name}>
                  <button
                    className={`memory-item${doc.name === selectedName ? ' memory-item--active' : ''}`}
                    type="button"
                    onClick={() => setSelectedName(doc.name)}
                  >
                    <div>
                      <p className="memory-item__title">{doc.name}</p>
                      <p className="memory-item__meta">{formatTimestamp(doc.updated_at)}</p>
                    </div>
                    <span className="memory-item__meta">{formatBytes(doc.size_bytes)}</span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        <div className="panel memory-detail">
          <div className="panel__header">
            <h2>Preview</h2>
            <span className="panel__meta">
              {detailState === 'loading'
                ? 'Loading…'
                : activeDoc
                  ? formatTimestamp(activeDoc.updated_at)
                  : 'Select a doc'}
            </span>
          </div>
          {activeDoc ? (
            <>
              <div className="memory-detail__meta">
                <span>{activeDoc.path}</span>
                <span>{formatBytes(activeDoc.size_bytes)}</span>
              </div>
              <pre className="memory-detail__content">{activeDoc.content}</pre>
            </>
          ) : (
            <p className="empty">Choose a memory doc to preview its content.</p>
          )}
        </div>
      </section>
    </div>
  )
}
