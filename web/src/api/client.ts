export type Run = {
  id: string
  status: string
  current_phase: string
  started_at: string
  finished_at?: string | null
  iterations: number
}

export type RunEvent = {
  id: string
  run_id: string
  ts: string
  type: string
  phase: string
  message: string
  meta: Record<string, string>
}

export type Artifact = {
  id: string
  run_id: string
  name: string
  type: string
  path: string
  created_at: string
  size_bytes: number
  checksum: string
}

export type Health = {
  status: string
  time: string
}

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

async function requestJson<T>(path: string, init?: RequestInit): Promise<T> {
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

  return (await response.json()) as T
}

export const api = {
  baseUrl: apiBaseUrl,
  health: () => requestJson<Health>('/health'),
  runs: () => requestJson<Run[]>('/runs'),
  run: (id: string) => requestJson<Run>(`/runs/${id}`),
  runEvents: (id: string) => requestJson<RunEvent[]>(`/runs/${id}/events`),
  runArtifacts: (id: string) => requestJson<Artifact[]>(`/runs/${id}/artifacts`),
  artifact: (id: string) => requestJson<Artifact>(`/artifacts/${id}`),
}
