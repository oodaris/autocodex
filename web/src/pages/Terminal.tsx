import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import { api, buildWsUrl } from '../api/client'
import type { TerminalSessionCreateRequest, TerminalSessionSummary } from '../api/client'
import { useAsync } from '../hooks/useAsync'
import { usePolling } from '../hooks/usePolling'
import { formatTimestamp } from '../utils/format'
import { normalizeTerminalOutput, stripAnsi } from '../utils/stripAnsi'

type LoadState = 'idle' | 'loading' | 'ready' | 'error'

export default function Terminal() {
  const { workspaceId } = useParams<{ workspaceId?: string }>()
  const [sessions, setSessions] = useState<TerminalSessionSummary[]>([])
  const [activeId, setActiveId] = useState<string | null>(null)
  const [sessionOutputs, setSessionOutputs] = useState<Record<string, string>>({})
  const [input, setInput] = useState('')
  const [command, setCommand] = useState('')
  const [state, setState] = useState<LoadState>('idle')
  const [error, setError] = useState<string | null>(null)
  const [autoRefresh, setAutoRefresh] = useState(true)
  const socketRef = useRef<WebSocket | null>(null)
  const autoStartRef = useRef(false)

  const pollConfig = {
    intervalMs: 5000,
    maxIntervalMs: 20000,
    backoffFactor: 1.5,
  }

  const handleAutoCreate = useCallback(async () => {
    if (autoStartRef.current) return
    autoStartRef.current = true
    const payload: TerminalSessionCreateRequest = {}
    if (workspaceId) payload.workspace_id = workspaceId
    try {
      const created = await api.createTerminalSession(payload)
      setSessions((current) => [created, ...current])
      setActiveId(created.id)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to create session'
      setError(message)
    }
  }, [workspaceId])

  const refresh = useCallback(
    async ({ signal, silent }: { signal?: AbortSignal; silent?: boolean } = {}) => {
      if (signal?.aborted) return
      if (!silent) setState('loading')
      setError(null)
      try {
        const payload = await api.terminalSessions({ signal })
        if (signal?.aborted) return
        setSessions(payload)
        setState('ready')
        if (!activeId && payload.length > 0) {
          setActiveId(payload[0].id)
        }
        if (payload.length === 0 && !autoStartRef.current) {
          void handleAutoCreate()
        }
      } catch (err) {
        if (signal?.aborted) return
        if (err instanceof DOMException && err.name === 'AbortError') return
        const message = err instanceof Error ? err.message : 'Unknown error'
        setError(message)
        setState('error')
      }
    },
    [activeId, handleAutoCreate],
  )

  useAsync((signal) => refresh({ signal }), [refresh])

  const pollDelayMs = usePolling(
    (signal) => refresh({ signal, silent: true }),
    { enabled: autoRefresh, ...pollConfig },
  )

  const activeSession = useMemo(() => sessions.find((session) => session.id === activeId) ?? null, [sessions, activeId])

  useEffect(() => {
    if (!activeSession) {
      return
    }

    const wsUrl = buildWsUrl(`/terminal/sessions/${activeSession.id}/ws`)
    const socket = new WebSocket(wsUrl)
    socket.binaryType = 'arraybuffer'
    socketRef.current = socket

    socket.onopen = () => {
      setError(null)
    }

    socket.onmessage = (event) => {
      const raw =
        typeof event.data === 'string'
          ? event.data
          : new TextDecoder().decode(event.data as ArrayBuffer)
      const text = stripAnsi(raw)
      if (!text) return
      setSessionOutputs((current) => ({
        ...current,
        [activeSession.id]: `${current[activeSession.id] ?? ''}${text}`,
      }))
    }

    socket.onerror = () => {
      setError('terminal websocket error')
    }

    return () => {
      socket.close()
      socketRef.current = null
    }
  }, [activeSession])

  useEffect(() => {
    autoStartRef.current = false
  }, [workspaceId])

  const handleCreate = useCallback(async () => {
    const payload: TerminalSessionCreateRequest = {}
    if (workspaceId) payload.workspace_id = workspaceId
    if (command.trim()) payload.command = command.trim()
    try {
      const created = await api.createTerminalSession(payload)
      setSessions((current) => [created, ...current])
      setActiveId(created.id)
      setCommand('')
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to create session'
      setError(message)
    }
  }, [command, workspaceId])


  const handleSend = useCallback(async () => {
    if (!input.trim()) return
    if (!socketRef.current || socketRef.current.readyState !== WebSocket.OPEN) {
      setError('terminal is not connected')
      return
    }
    socketRef.current.send(input)
    setInput('')
  }, [input])

  const handleClose = useCallback(async (sessionId: string) => {
    try {
      await api.closeTerminalSession(sessionId)
      setSessions((current) => current.map((s) => (s.id === sessionId ? { ...s, status: 'closed' } : s)))
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to close session'
      setError(message)
    }
  }, [])

  const output = activeId ? sessionOutputs[activeId] ?? '' : ''
  const displayOutput = normalizeTerminalOutput(output)

  return (
    <div className="page">
      <header className="detail-header">
        <div>
          <h1>Terminal sessions</h1>
          <p>
            Start a live terminal session for the current workspace.
            {workspaceId ? ` Workspace: ${workspaceId}.` : ''}
          </p>
        </div>
        <div className="detail-actions">
          <div className="refresh-controls">
            <label className="toggle">
              <input
                type="checkbox"
                name="terminal-auto-refresh"
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

      <section className="terminal-grid">
        <div className="panel terminal-sidebar">
          <div className="panel__header">
            <h2>Sessions</h2>
            <span className="panel__meta">{sessions.length} active</span>
          </div>
          <div className="terminal-create">
            <input
              className="control-input"
              type="text"
              name="terminal-command"
              placeholder="Command (optional, defaults to codex)"
              value={command}
              onChange={(event) => setCommand(event.target.value)}
            />
            <button className="button" type="button" onClick={() => void handleCreate()}>
              Start session
            </button>
          </div>
          {sessions.length === 0 ? (
            <p className="empty">No terminal sessions yet.</p>
          ) : (
            <ul className="terminal-list">
              {sessions.map((session) => (
                <li key={session.id}>
                  <button
                    type="button"
                    className={`terminal-item${session.id === activeId ? ' terminal-item--active' : ''}`}
                    onClick={() => setActiveId(session.id)}
                  >
                    <div>
                      <p className="terminal-item__title">{session.command}</p>
                      <p className="terminal-item__meta">
                        {formatTimestamp(session.created_at)} · {session.status}
                      </p>
                    </div>
                    <span className="terminal-item__meta">PID {session.pid}</span>
                  </button>
                  <div className="terminal-item__actions">
                    <button
                      className="link"
                      type="button"
                      onClick={() => void handleClose(session.id)}
                    >
                      Close
                    </button>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>

        <div className="panel terminal-output">
          <div className="panel__header">
            <h2>Output</h2>
            <span className="panel__meta">{activeSession ? activeSession.id : 'No session selected'}</span>
          </div>
          <pre className="terminal-screen">{displayOutput || 'Connect to a session to stream output.'}</pre>
          <div className="terminal-input">
            <input
              className="control-input"
              type="text"
              name="terminal-input"
              placeholder="Send input"
              value={input}
              onChange={(event) => setInput(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter') {
                  void handleSend()
                }
              }}
            />
            <button className="button" type="button" onClick={() => void handleSend()}>
              Send
            </button>
          </div>
        </div>
      </section>
    </div>
  )
}
