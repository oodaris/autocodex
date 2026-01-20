import { describe, expect, it } from 'vitest'
import { parseRuns } from '../api/schema'

describe('parseRuns', () => {
  it('returns empty array for null payload', () => {
    expect(parseRuns(null)).toEqual([])
  })

  it('returns empty array for undefined payload', () => {
    expect(parseRuns(undefined)).toEqual([])
  })

  it('accepts payloads wrapped in a runs field', () => {
    expect(parseRuns({ runs: [] })).toEqual([])
  })
})
