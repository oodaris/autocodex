import {
  parseArtifact,
  parseHealth,
  parseRun,
  parseRunArtifacts,
  parseRunEvents,
  parseRuns,
} from './schema'
import type { Artifact, Health, Run, RunEvent } from './types'

export type { Artifact, Health, Run, RunEvent } from './types'

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
  runs: (options?: ApiRequestOptions) => requestJson<Run[]>('/runs', options, parseRuns),
  run: (id: string, options?: ApiRequestOptions) => requestJson<Run>(`/runs/${id}`, options, parseRun),
  runEvents: (id: string, options?: ApiRequestOptions) =>
    requestJson<RunEvent[]>(`/runs/${id}/events`, options, parseRunEvents),
  runArtifacts: (id: string, options?: ApiRequestOptions) =>
    requestJson<Artifact[]>(`/runs/${id}/artifacts`, options, parseRunArtifacts),
  artifact: (id: string, options?: ApiRequestOptions) =>
    requestJson<Artifact>(`/artifacts/${id}`, options, parseArtifact),
}
