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

export type SnapshotSummary = {
  id: string
  run_id: string
  created_at: string
  reason: string
  size_bytes: number
  content_path: string
}

export type SnapshotManifest = {
  events: number
  artifacts: number
  memory_docs: number
  bytes: number
}

export type SnapshotDetail = {
  summary: SnapshotSummary
  manifest: SnapshotManifest
  content: string
}

export type SnapshotCreateRequest = {
  reason?: string
  include_events?: boolean
  include_artifacts?: boolean
  include_memory?: boolean
  max_bytes?: number
}

export type WorkspaceSummary = {
  id: string
  name: string
  root: string
  config_path: string
  status: 'ok' | 'missing_config' | 'invalid_config' | 'state_error'
  error?: string | null
  runs_count: number
  last_run_id?: string | null
  last_run_status?: string | null
  last_run_phase?: string | null
  last_run_started_at?: string | null
  last_run_finished_at?: string | null
}

export type TerminalSessionCreateRequest = {
  workspace_id?: string
  command?: string
  args?: string[]
  env?: string[]
}

export type TerminalSessionSummary = {
  id: string
  status: 'running' | 'closed'
  command: string
  args: string[]
  cwd: string
  workspace_id?: string | null
  pid: number
  created_at: string
  updated_at: string
  exit_code?: number | null
}
