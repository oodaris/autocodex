import {
  parseArtifact,
  parseHealth,
  parseMemoryDoc,
  parseMemoryDocs,
  parseRun,
  parseRunArtifacts,
  parseRunControlResponse,
  parseRunControlStatus,
  parseRunEvents,
  parseRuns,
  parseSnapshotDetail,
  parseSnapshotSummaries,
  parseTerminalSessionSummaries,
  parseTerminalSessionSummary,
  parseWorkspaceSummaries,
  parseWorkspaceSummary,
} from './schema'
import type {
  Artifact,
  Health,
  MemoryDocDetail,
  MemoryDocSummary,
  Run,
  RunControlRequest,
  RunControlResponse,
  RunControlStatus,
  SnapshotCreateRequest,
  SnapshotDetail,
  SnapshotSummary,
  RunEvent,
  TerminalSessionCreateRequest,
  TerminalSessionSummary,
  WorkspaceSummary,
} from './types'

export type {
  Artifact,
  Health,
  MemoryDocDetail,
  MemoryDocSummary,
  Run,
  RunControlRequest,
  RunControlResponse,
  RunControlStatus,
  SnapshotCreateRequest,
  SnapshotDetail,
  SnapshotSummary,
  RunEvent,
  TerminalSessionCreateRequest,
  TerminalSessionSummary,
  WorkspaceSummary,
} from './types'

const DEFAULT_BASE_URL = 'http://127.0.0.1:7788'
const TOKEN_STORAGE_KEY = 'autocodex-api-token'

export const apiAuth = {
  getToken: (): string => {
    if (typeof sessionStorage === 'undefined') return ''
    return sessionStorage.getItem(TOKEN_STORAGE_KEY) ?? ''
  },
  setToken: (token: string) => {
    if (typeof sessionStorage === 'undefined') return
    const value = token.trim()
    if (value === '') {
      sessionStorage.removeItem(TOKEN_STORAGE_KEY)
    } else {
      sessionStorage.setItem(TOKEN_STORAGE_KEY, value)
    }
  },
  clear: () => {
    if (typeof sessionStorage === 'undefined') return
    sessionStorage.removeItem(TOKEN_STORAGE_KEY)
  },
}

const apiBaseUrl = (() => {
  const env = import.meta.env
  if (env && typeof env.VITE_API_BASE_URL === 'string' && env.VITE_API_BASE_URL.trim() !== '') {
    return env.VITE_API_BASE_URL.trim()
  }
  return DEFAULT_BASE_URL
})()

function buildUrl(path: string): string {
  if (path.startsWith('http://') || path.startsWith('https://')) {
    return path
  }
  const normalizedBase = apiBaseUrl.endsWith('/') ? apiBaseUrl.slice(0, -1) : apiBaseUrl
  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  return `${normalizedBase}${normalizedPath}`
}

export type ApiRequestOptions = RequestInit

async function requestJson<T>(
  path: string,
  init: ApiRequestOptions | undefined,
  parse?: (value: unknown) => T,
): Promise<T> {
  const token = apiAuth.getToken()
  const response = await fetch(buildUrl(path), {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
  })

  if (!response.ok) {
    const text = await response.text()
    throw new Error(text || `Request failed with status ${response.status}`)
  }

  const payload = (await response.json()) as unknown
  return parse ? parse(payload) : (payload as T)
}

export function buildWsUrl(path: string, token?: string): string {
  const base = buildUrl(path)
  const url = new URL(base)
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
  const authToken = token ?? apiAuth.getToken()
  if (authToken) {
    url.searchParams.set('token', authToken)
  }
  return url.toString()
}

export const api = {
  baseUrl: apiBaseUrl,
  health: (options?: ApiRequestOptions) => requestJson<Health>('/health', options, parseHealth),
  memoryDocs: (options?: ApiRequestOptions) =>
    requestJson<MemoryDocSummary[]>('/memory', options, parseMemoryDocs),
  memoryDoc: (name: string, options?: ApiRequestOptions) =>
    requestJson<MemoryDocDetail>(`/memory/${encodeURIComponent(name)}`, options, parseMemoryDoc),
  runs: (options?: ApiRequestOptions) => requestJson<Run[]>('/runs', options, parseRuns),
  run: (id: string, options?: ApiRequestOptions) => requestJson<Run>(`/runs/${id}`, options, parseRun),
  runEvents: (id: string, options?: ApiRequestOptions) =>
    requestJson<RunEvent[]>(`/runs/${id}/events`, options, parseRunEvents),
  runArtifacts: (id: string, options?: ApiRequestOptions) =>
    requestJson<Artifact[]>(`/runs/${id}/artifacts`, options, parseRunArtifacts),
  artifact: (id: string, options?: ApiRequestOptions) =>
    requestJson<Artifact>(`/artifacts/${id}`, options, parseArtifact),
  runControlStatus: (id: string, options?: ApiRequestOptions) =>
    requestJson<RunControlStatus>(`/runs/${id}/control`, options, parseRunControlStatus),
  runControlAction: (id: string, payload: RunControlRequest, options?: ApiRequestOptions) =>
    requestJson<RunControlResponse>(
      `/runs/${id}/control`,
      {
        method: 'POST',
        body: JSON.stringify(payload),
        ...options,
      },
      parseRunControlResponse,
    ),
  runSnapshots: (id: string, options?: ApiRequestOptions) =>
    requestJson<SnapshotSummary[]>(`/runs/${id}/snapshots`, options, parseSnapshotSummaries),
  createSnapshot: (id: string, payload: SnapshotCreateRequest, options?: ApiRequestOptions) =>
    requestJson<SnapshotDetail>(
      `/runs/${id}/snapshots`,
      {
        method: 'POST',
        body: JSON.stringify(payload),
        ...options,
      },
      parseSnapshotDetail,
    ),
  snapshot: (runId: string, snapshotId: string, options?: ApiRequestOptions) =>
    requestJson<SnapshotDetail>(
      `/runs/${runId}/snapshots/${snapshotId}`,
      options,
      parseSnapshotDetail,
    ),
  hubWorkspaces: (options?: ApiRequestOptions) =>
    requestJson<WorkspaceSummary[]>('/hub/workspaces', options, parseWorkspaceSummaries),
  hubWorkspace: (workspaceId: string, options?: ApiRequestOptions) =>
    requestJson<WorkspaceSummary>(
      `/hub/workspaces/${encodeURIComponent(workspaceId)}`,
      options,
      parseWorkspaceSummary,
    ),
  hubRuns: (workspaceId: string, options?: ApiRequestOptions) =>
    requestJson<Run[]>(
      `/hub/workspaces/${encodeURIComponent(workspaceId)}/runs`,
      options,
      parseRuns,
    ),
  hubRun: (workspaceId: string, runId: string, options?: ApiRequestOptions) =>
    requestJson<Run>(
      `/hub/workspaces/${encodeURIComponent(workspaceId)}/runs/${runId}`,
      options,
      parseRun,
    ),
  hubRunEvents: (workspaceId: string, runId: string, options?: ApiRequestOptions) =>
    requestJson<RunEvent[]>(
      `/hub/workspaces/${encodeURIComponent(workspaceId)}/runs/${runId}/events`,
      options,
      parseRunEvents,
    ),
  hubRunArtifacts: (workspaceId: string, runId: string, options?: ApiRequestOptions) =>
    requestJson<Artifact[]>(
      `/hub/workspaces/${encodeURIComponent(workspaceId)}/runs/${runId}/artifacts`,
      options,
      parseRunArtifacts,
    ),
  hubArtifact: (workspaceId: string, artifactId: string, options?: ApiRequestOptions) =>
    requestJson<Artifact>(
      `/hub/workspaces/${encodeURIComponent(workspaceId)}/artifacts/${artifactId}`,
      options,
      parseArtifact,
    ),
  hubRunControlStatus: (workspaceId: string, runId: string, options?: ApiRequestOptions) =>
    requestJson<RunControlStatus>(
      `/hub/workspaces/${encodeURIComponent(workspaceId)}/runs/${runId}/control`,
      options,
      parseRunControlStatus,
    ),
  hubRunControlAction: (
    workspaceId: string,
    runId: string,
    payload: RunControlRequest,
    options?: ApiRequestOptions,
  ) =>
    requestJson<RunControlResponse>(
      `/hub/workspaces/${encodeURIComponent(workspaceId)}/runs/${runId}/control`,
      {
        method: 'POST',
        body: JSON.stringify(payload),
        ...options,
      },
      parseRunControlResponse,
    ),
  hubRunSnapshots: (workspaceId: string, runId: string, options?: ApiRequestOptions) =>
    requestJson<SnapshotSummary[]>(
      `/hub/workspaces/${encodeURIComponent(workspaceId)}/runs/${runId}/snapshots`,
      options,
      parseSnapshotSummaries,
    ),
  hubCreateSnapshot: (
    workspaceId: string,
    runId: string,
    payload: SnapshotCreateRequest,
    options?: ApiRequestOptions,
  ) =>
    requestJson<SnapshotDetail>(
      `/hub/workspaces/${encodeURIComponent(workspaceId)}/runs/${runId}/snapshots`,
      {
        method: 'POST',
        body: JSON.stringify(payload),
        ...options,
      },
      parseSnapshotDetail,
    ),
  hubSnapshot: (workspaceId: string, runId: string, snapshotId: string, options?: ApiRequestOptions) =>
    requestJson<SnapshotDetail>(
      `/hub/workspaces/${encodeURIComponent(workspaceId)}/runs/${runId}/snapshots/${snapshotId}`,
      options,
      parseSnapshotDetail,
    ),
  hubMemoryDocs: (workspaceId: string, options?: ApiRequestOptions) =>
    requestJson<MemoryDocSummary[]>(
      `/hub/workspaces/${encodeURIComponent(workspaceId)}/memory`,
      options,
      parseMemoryDocs,
    ),
  hubMemoryDoc: (workspaceId: string, name: string, options?: ApiRequestOptions) =>
    requestJson<MemoryDocDetail>(
      `/hub/workspaces/${encodeURIComponent(workspaceId)}/memory/${encodeURIComponent(name)}`,
      options,
      parseMemoryDoc,
    ),
  terminalSessions: (options?: ApiRequestOptions) =>
    requestJson<TerminalSessionSummary[]>(
      '/terminal/sessions',
      options,
      parseTerminalSessionSummaries,
    ),
  terminalSession: (sessionId: string, options?: ApiRequestOptions) =>
    requestJson<TerminalSessionSummary>(
      `/terminal/sessions/${sessionId}`,
      options,
      parseTerminalSessionSummary,
    ),
  createTerminalSession: (payload: TerminalSessionCreateRequest, options?: ApiRequestOptions) =>
    requestJson<TerminalSessionSummary>(
      '/terminal/sessions',
      {
        method: 'POST',
        body: JSON.stringify(payload),
        ...options,
      },
      parseTerminalSessionSummary,
    ),
  closeTerminalSession: (sessionId: string, options?: ApiRequestOptions) =>
    requestJson<TerminalSessionSummary>(
      `/terminal/sessions/${sessionId}`,
      {
        method: 'DELETE',
        ...options,
      },
      parseTerminalSessionSummary,
    ),
}
