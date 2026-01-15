import type { Artifact, Health, Run, RunEvent } from './types'

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

export function parseHealth(value: unknown): Health {
  if (!isRecord(value)) {
    throw new Error('Invalid health payload: expected object')
  }
  assertString(value.status, 'health.status')
  assertString(value.time, 'health.time')
  return value as Health
}

export function parseRuns(value: unknown): Run[] {
  if (!Array.isArray(value)) {
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
