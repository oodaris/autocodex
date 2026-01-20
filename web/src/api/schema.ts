import type {
  Artifact,
  Health,
  MemoryDocDetail,
  MemoryDocSummary,
  Run,
  RunControlResponse,
  RunControlStatus,
  SnapshotDetail,
  SnapshotSummary,
  RunEvent,
  WorkspaceSummary,
  TerminalSessionSummary,
} from './types'

type UnknownRecord = Record<string, unknown>

function isRecord(value: unknown): value is UnknownRecord {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function assertString(value: unknown, path: string): asserts value is string {
  if (typeof value !== 'string') {
    throw new Error(`Invalid ${path}: expected string`)
  }
}

function assertNumber(value: unknown, path: string): asserts value is number {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    throw new Error(`Invalid ${path}: expected number`)
  }
}

function assertNullableNumber(value: unknown, path: string): asserts value is number | null | undefined {
  if (value == null) return
  if (typeof value !== 'number' || Number.isNaN(value)) {
    throw new Error(`Invalid ${path}: expected number or null`)
  }
}

function assertNullableString(value: unknown, path: string): asserts value is string | null | undefined {
  if (value == null) return
  if (typeof value !== 'string') {
    throw new Error(`Invalid ${path}: expected string or null`)
  }
}

function assertRecordOfStrings(value: unknown, path: string): asserts value is Record<string, string> {
  if (!isRecord(value)) {
    throw new Error(`Invalid ${path}: expected object`)
  }
  for (const [key, entry] of Object.entries(value)) {
    if (typeof entry !== 'string') {
      throw new Error(`Invalid ${path}.${key}: expected string`)
    }
  }
}

function assertRun(value: unknown, path = 'run'): asserts value is Run {
  if (!isRecord(value)) {
    throw new Error(`Invalid ${path}: expected object`)
  }
  assertString(value.id, `${path}.id`)
  assertString(value.status, `${path}.status`)
  assertString(value.current_phase, `${path}.current_phase`)
  assertString(value.started_at, `${path}.started_at`)
  assertNullableString(value.finished_at, `${path}.finished_at`)
  assertNumber(value.iterations, `${path}.iterations`)
}

function assertRunEvent(value: unknown, path = 'event'): asserts value is RunEvent {
  if (!isRecord(value)) {
    throw new Error(`Invalid ${path}: expected object`)
  }
  assertString(value.id, `${path}.id`)
  assertString(value.run_id, `${path}.run_id`)
  assertString(value.ts, `${path}.ts`)
  assertString(value.type, `${path}.type`)
  assertString(value.phase, `${path}.phase`)
  assertString(value.message, `${path}.message`)
  assertRecordOfStrings(value.meta, `${path}.meta`)
}

function assertArtifact(value: unknown, path = 'artifact'): asserts value is Artifact {
  if (!isRecord(value)) {
    throw new Error(`Invalid ${path}: expected object`)
  }
  assertString(value.id, `${path}.id`)
  assertString(value.run_id, `${path}.run_id`)
  assertString(value.name, `${path}.name`)
  assertString(value.type, `${path}.type`)
  assertString(value.path, `${path}.path`)
  assertString(value.created_at, `${path}.created_at`)
  assertNumber(value.size_bytes, `${path}.size_bytes`)
  assertString(value.checksum, `${path}.checksum`)
}

function assertMemoryDocSummary(value: unknown, path = 'memory_doc'): asserts value is MemoryDocSummary {
  if (!isRecord(value)) {
    throw new Error(`Invalid ${path}: expected object`)
  }
  assertString(value.name, `${path}.name`)
  assertString(value.path, `${path}.path`)
  assertString(value.updated_at, `${path}.updated_at`)
  assertNumber(value.size_bytes, `${path}.size_bytes`)
}

function assertMemoryDocDetail(value: unknown, path = 'memory_doc'): asserts value is MemoryDocDetail {
  assertMemoryDocSummary(value, path)
  const record = value as UnknownRecord
  assertString(record.content, `${path}.content`)
}

function assertRunControlStatus(value: unknown, path = 'run_control'): asserts value is RunControlStatus {
  if (!isRecord(value)) {
    throw new Error(`Invalid ${path}: expected object`)
  }
  assertString(value.run_id, `${path}.run_id`)
  assertString(value.status, `${path}.status`)
  assertNullableString(value.last_action, `${path}.last_action`)
  assertNullableString(value.last_action_at, `${path}.last_action_at`)
}

function assertRunControlResponse(value: unknown, path = 'run_control_response'): asserts value is RunControlResponse {
  if (!isRecord(value)) {
    throw new Error(`Invalid ${path}: expected object`)
  }
  assertString(value.run_id, `${path}.run_id`)
  assertString(value.action, `${path}.action`)
  if (typeof value.accepted !== 'boolean') {
    throw new Error(`Invalid ${path}.accepted: expected boolean`)
  }
  assertString(value.status, `${path}.status`)
  assertString(value.message, `${path}.message`)
}

function assertSnapshotSummary(value: unknown, path = 'snapshot_summary'): asserts value is SnapshotSummary {
  if (!isRecord(value)) {
    throw new Error(`Invalid ${path}: expected object`)
  }
  assertString(value.id, `${path}.id`)
  assertString(value.run_id, `${path}.run_id`)
  assertString(value.created_at, `${path}.created_at`)
  assertString(value.reason, `${path}.reason`)
  assertNumber(value.size_bytes, `${path}.size_bytes`)
  assertString(value.content_path, `${path}.content_path`)
}

function assertSnapshotDetail(value: unknown, path = 'snapshot_detail'): asserts value is SnapshotDetail {
  if (!isRecord(value)) {
    throw new Error(`Invalid ${path}: expected object`)
  }
  assertSnapshotSummary(value.summary, `${path}.summary`)
  if (!isRecord(value.manifest)) {
    throw new Error(`Invalid ${path}.manifest: expected object`)
  }
  assertNumber(value.manifest.events, `${path}.manifest.events`)
  assertNumber(value.manifest.artifacts, `${path}.manifest.artifacts`)
  assertNumber(value.manifest.memory_docs, `${path}.manifest.memory_docs`)
  assertNumber(value.manifest.bytes, `${path}.manifest.bytes`)
  assertString(value.content, `${path}.content`)
}

function assertWorkspaceSummary(value: unknown, path = 'workspace'): asserts value is WorkspaceSummary {
  if (!isRecord(value)) {
    throw new Error(`Invalid ${path}: expected object`)
  }
  assertString(value.id, `${path}.id`)
  assertString(value.name, `${path}.name`)
  assertString(value.root, `${path}.root`)
  assertString(value.config_path, `${path}.config_path`)
  assertString(value.status, `${path}.status`)
  assertNullableString(value.error, `${path}.error`)
  assertNumber(value.runs_count, `${path}.runs_count`)
  assertNullableString(value.last_run_id, `${path}.last_run_id`)
  assertNullableString(value.last_run_status, `${path}.last_run_status`)
  assertNullableString(value.last_run_phase, `${path}.last_run_phase`)
  assertNullableString(value.last_run_started_at, `${path}.last_run_started_at`)
  assertNullableString(value.last_run_finished_at, `${path}.last_run_finished_at`)
}

function assertTerminalSessionSummary(value: unknown, path = 'terminal_session'): asserts value is TerminalSessionSummary {
  if (!isRecord(value)) {
    throw new Error(`Invalid ${path}: expected object`)
  }
  assertString(value.id, `${path}.id`)
  assertString(value.status, `${path}.status`)
  assertString(value.command, `${path}.command`)
  if (!Array.isArray(value.args)) {
    throw new Error(`Invalid ${path}.args: expected array`)
  }
  value.args.forEach((entry: unknown, index: number) => assertString(entry, `${path}.args[${index}]`))
  assertString(value.cwd, `${path}.cwd`)
  assertNullableString(value.workspace_id, `${path}.workspace_id`)
  assertNumber(value.pid, `${path}.pid`)
  assertString(value.created_at, `${path}.created_at`)
  assertString(value.updated_at, `${path}.updated_at`)
  assertNullableNumber(value.exit_code, `${path}.exit_code`)
}

export function parseHealth(value: unknown): Health {
  if (!isRecord(value)) {
    throw new Error('Invalid health payload: expected object')
  }
  assertString(value.status, 'health.status')
  assertString(value.time, 'health.time')
  return value as Health
}

export function parseMemoryDocs(value: unknown): MemoryDocSummary[] {
  if (!Array.isArray(value)) {
    throw new Error('Invalid memory docs payload: expected array')
  }
  value.forEach((entry, index) => assertMemoryDocSummary(entry, `memory_docs[${index}]`))
  return value as MemoryDocSummary[]
}

export function parseMemoryDoc(value: unknown): MemoryDocDetail {
  assertMemoryDocDetail(value, 'memory_doc')
  return value as MemoryDocDetail
}

export function parseRuns(value: unknown): Run[] {
  if (value == null) {
    return []
  }
  if (!Array.isArray(value)) {
    if (isRecord(value) && Array.isArray(value.runs)) {
      const runsValue = value.runs as unknown[]
      runsValue.forEach((entry, index) => assertRun(entry, `runs[${index}]`))
      return runsValue as Run[]
    }
    throw new Error('Invalid runs payload: expected array')
  }
  value.forEach((entry, index) => assertRun(entry, `runs[${index}]`))
  return value as Run[]
}

export function parseRun(value: unknown): Run {
  assertRun(value, 'run')
  return value as Run
}

export function parseRunEvents(value: unknown): RunEvent[] {
  if (!Array.isArray(value)) {
    throw new Error('Invalid events payload: expected array')
  }
  value.forEach((entry, index) => assertRunEvent(entry, `events[${index}]`))
  return value as RunEvent[]
}

export function parseRunArtifacts(value: unknown): Artifact[] {
  if (!Array.isArray(value)) {
    throw new Error('Invalid artifacts payload: expected array')
  }
  value.forEach((entry, index) => assertArtifact(entry, `artifacts[${index}]`))
  return value as Artifact[]
}

export function parseArtifact(value: unknown): Artifact {
  assertArtifact(value, 'artifact')
  return value as Artifact
}

export function parseRunControlStatus(value: unknown): RunControlStatus {
  assertRunControlStatus(value, 'run_control')
  return value as RunControlStatus
}

export function parseRunControlResponse(value: unknown): RunControlResponse {
  assertRunControlResponse(value, 'run_control_response')
  return value as RunControlResponse
}

export function parseSnapshotSummaries(value: unknown): SnapshotSummary[] {
  if (!Array.isArray(value)) {
    throw new Error('Invalid snapshots payload: expected array')
  }
  value.forEach((entry, index) => assertSnapshotSummary(entry, `snapshots[${index}]`))
  return value as SnapshotSummary[]
}

export function parseSnapshotDetail(value: unknown): SnapshotDetail {
  assertSnapshotDetail(value, 'snapshot_detail')
  return value as SnapshotDetail
}

export function parseWorkspaceSummaries(value: unknown): WorkspaceSummary[] {
  if (!Array.isArray(value)) {
    throw new Error('Invalid workspaces payload: expected array')
  }
  value.forEach((entry, index) => assertWorkspaceSummary(entry, `workspaces[${index}]`))
  return value as WorkspaceSummary[]
}

export function parseWorkspaceSummary(value: unknown): WorkspaceSummary {
  assertWorkspaceSummary(value, 'workspace')
  return value as WorkspaceSummary
}

export function parseTerminalSessionSummaries(value: unknown): TerminalSessionSummary[] {
  if (!Array.isArray(value)) {
    throw new Error('Invalid terminal sessions payload: expected array')
  }
  value.forEach((entry, index) => assertTerminalSessionSummary(entry, `terminal_sessions[${index}]`))
  return value as TerminalSessionSummary[]
}

export function parseTerminalSessionSummary(value: unknown): TerminalSessionSummary {
  assertTerminalSessionSummary(value, 'terminal_session')
  return value as TerminalSessionSummary
}
