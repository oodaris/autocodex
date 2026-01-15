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

export type RunControlStatus = {
  run_id: string
  status: string
  last_action?: string | null
  last_action_at?: string | null
}

export type RunControlRequest = {
  action: 'resume' | 'stop' | 'cancel' | 'kill'
  reason?: string
  dry_run?: boolean
}

export type RunControlResponse = {
  run_id: string
  action: string
  accepted: boolean
  status: string
  message: string
}
