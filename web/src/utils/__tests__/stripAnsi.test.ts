import { describe, expect, it } from 'vitest'
import { normalizeTerminalOutput, stripAnsi } from '../stripAnsi'

describe('stripAnsi', () => {
  it('removes CSI escape sequences', () => {
    expect(stripAnsi('\x1b[31mred\x1b[0m')).toBe('red')
  })

  it('removes OSC sequences', () => {
    expect(stripAnsi('\x1b]10;?\x1b\\text')).toBe('text')
  })

  it('removes single escape codes', () => {
    expect(stripAnsi('\x1bcok')).toBe('ok')
  })

  it('removes CSI sequences with greater-than params', () => {
    expect(stripAnsi('\x1b[>7uready')).toBe('ready')
  })
})

describe('normalizeTerminalOutput', () => {
  it('collapses excessive blank lines', () => {
    expect(normalizeTerminalOutput('line1\n\n\nline2')).toBe('line1\n\nline2')
  })

  it('joins mostly single-character lines', () => {
    expect(normalizeTerminalOutput('a\nb\nc\n')).toBe('abc')
  })

  it('keeps multiline output when not mostly single-character', () => {
    expect(normalizeTerminalOutput('hello\nw\norld')).toBe('hello\nw\norld')
  })
})
