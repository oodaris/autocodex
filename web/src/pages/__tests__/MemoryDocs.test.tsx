import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

type MockApi = {
  memoryDocs: ReturnType<typeof vi.fn>
  memoryDoc: ReturnType<typeof vi.fn>
}

vi.mock('../../api/client', () => ({
  api: {
    memoryDocs: vi.fn(),
    memoryDoc: vi.fn(),
  },
}))

async function getMockedApi(): Promise<MockApi> {
  const { api } = await import('../../api/client')
  return api as unknown as MockApi
}

beforeEach(() => {
  vi.resetModules()
})

afterEach(() => {
  cleanup()
  vi.clearAllMocks()
})

describe('MemoryDocs', () => {
  it('renders memory docs and loads the first doc', async () => {
    const mockedApi = await getMockedApi()
    mockedApi.memoryDocs.mockResolvedValue([
      {
        name: 'TODO.md',
        path: 'TODO.md',
        updated_at: '2026-01-15T10:00:00Z',
        size_bytes: 10,
      },
      {
        name: 'PROGRESS.md',
        path: 'PROGRESS.md',
        updated_at: '2026-01-15T11:00:00Z',
        size_bytes: 20,
      },
    ])
    mockedApi.memoryDoc.mockImplementation(async () => ({
      name: 'TODO.md',
      path: 'TODO.md',
      updated_at: '2026-01-15T10:00:00Z',
      size_bytes: 10,
      content: 'Focus on memory docs.',
    }))

    const { default: MemoryDocs } = await import('../MemoryDocs')
    render(<MemoryDocs />)

    expect(await screen.findByText('TODO.md')).toBeInTheDocument()
    expect(await screen.findByText('Focus on memory docs.')).toBeInTheDocument()
  })

  it('loads a new doc when selected', async () => {
    const mockedApi = await getMockedApi()
    mockedApi.memoryDocs.mockResolvedValue([
      {
        name: 'TODO.md',
        path: 'TODO.md',
        updated_at: '2026-01-15T10:00:00Z',
        size_bytes: 10,
      },
      {
        name: 'SPEC.md',
        path: 'SPEC.md',
        updated_at: '2026-01-15T12:00:00Z',
        size_bytes: 30,
      },
    ])
    mockedApi.memoryDoc.mockImplementation(async (name: string) => {
      if (name === 'SPEC.md') {
        return {
          name: 'SPEC.md',
          path: 'SPEC.md',
          updated_at: '2026-01-15T12:00:00Z',
          size_bytes: 30,
          content: 'Second doc.',
        }
      }
      return {
        name: 'TODO.md',
        path: 'TODO.md',
        updated_at: '2026-01-15T10:00:00Z',
        size_bytes: 10,
        content: 'First doc.',
      }
    })

    const { default: MemoryDocs } = await import('../MemoryDocs')
    render(<MemoryDocs />)

    await screen.findByText('First doc.')
    const specButtons = await screen.findAllByText('SPEC.md')
    await userEvent.click(specButtons[0])

    await waitFor(() => {
      expect(mockedApi.memoryDoc).toHaveBeenLastCalledWith('SPEC.md', expect.any(Object))
    })
    expect(await screen.findByText('Second doc.')).toBeInTheDocument()
  })
})
