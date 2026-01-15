import { useCallback, useMemo, useState } from 'react'
import { api } from '../api/client'
import type { MemoryDocDetail, MemoryDocSummary } from '../api/client'
import { useAsync } from '../hooks/useAsync'
import { formatBytes, formatTimestamp } from '../utils/format'

type LoadState = 'idle' | 'loading' | 'ready' | 'error'

export default function MemoryDocs() {
  const [docs, setDocs] = useState<MemoryDocSummary[]>([])
  const [selectedName, setSelectedName] = useState<string | null>(null)
  const [activeDoc, setActiveDoc] = useState<MemoryDocDetail | null>(null)
  const [listState, setListState] = useState<LoadState>('idle')
  const [detailState, setDetailState] = useState<LoadState>('idle')
  const [error, setError] = useState<string | null>(null)

  const refreshList = useCallback(async ({ signal }: { signal?: AbortSignal } = {}) => {
    if (signal?.aborted) return
    setListState('loading')
    setError(null)
    try {
      const payload = await api.memoryDocs({ signal })
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
  }, [selectedName])

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
        const payload = await api.memoryDoc(selectedName, { signal })
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
    [selectedName],
  )

  useAsync((signal) => refreshList({ signal }), [refreshList])
  useAsync((signal) => refreshDetail({ signal }), [refreshDetail])

  const sortedDocs = useMemo(() => {
    return [...docs].sort((a, b) => a.name.localeCompare(b.name))
  }, [docs])

  return (
    <div className="page">
      <header className="detail-header">
        <div>
          <h1>Memory docs</h1>
          <p>Local markdown notes that autocodex reads and updates during each loop.</p>
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
