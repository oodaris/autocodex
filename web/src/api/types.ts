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

export type MemoryDocSummary = {
  name: string
  path: string
  updated_at: string
  size_bytes: number
}

export type MemoryDocDetail = MemoryDocSummary & {
  content: string
}
