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
} from './types'

const DEFAULT_BASE_URL = 'http://127.0.0.1:7788'

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
  const response = await fetch(buildUrl(path), {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
  })

  if (!response.ok) {
    const text = await response.text()
    throw new Error(text || `Request failed with status ${response.status}`)
  }

  const payload = (await response.json()) as unknown
  return parse ? parse(payload) : (payload as T)
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
}
