/**
 * What a search answers, read into the window's own words: a part of a match
 * the answer left out reads as nothing, and a passage is a start and a length.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { SearchMode } from '@numen/protocol'
import { searchCore } from './search'

const search = vi.hoisted(() => ({
  searchNames: vi.fn(),
  searchPassages: vi.fn(),
}))

vi.mock('@/shared/clients', () => ({ search }))

beforeEach(() => {
  vi.clearAllMocks()
})

describe('a name searched for', () => {
  it('is requested for with the words typed, as many as the window shows', async () => {
    search.searchNames.mockResolvedValue({ found: [] })

    await searchCore.names('lla', 8)

    expect(search.searchNames).toHaveBeenCalledWith({ query: 'lla', limit: 8 })
  })

  it('carries the note and the heading it was found at', async () => {
    search.searchNames.mockResolvedValue({
      found: [
        {
          note: { path: 'Llama.md', title: 'Llama' },
          heading: { text: 'Diet', line: 4 },
          spans: [],
        },
      ],
    })

    expect(await searchCore.names('lla', 8)).toMatchObject([
      { path: 'Llama.md', title: 'Llama', heading: 'Diet', line: 4 },
    ])
  })

  it('is at no note and no heading where the answer names neither', async () => {
    search.searchNames.mockResolvedValue({ found: [{ spans: [] }] })

    expect(await searchCore.names('lla', 8)).toMatchObject([
      { path: '', title: '', heading: '', line: -1 },
    ])
  })
})

describe('a passage searched for', () => {
  it('is requested for in the mode chosen', async () => {
    search.searchPassages.mockResolvedValue({ found: [] })

    await searchCore.search('grass', 'meaning', 8)

    expect(search.searchPassages).toHaveBeenCalledWith({
      query: 'grass',
      limit: 8,
      mode: SearchMode.MEANING,
    })
  })

  it('starts where its run of text starts, as long as that run', async () => {
    search.searchPassages.mockResolvedValue({
      found: [
        {
          path: 'Llama.md',
          note: { title: 'Llama' },
          text: 'eats grass',
          span: { from: 3, to: 10 },
          line: 2,
          spans: [],
        },
      ],
    })

    expect(await searchCore.search('grass', 'words', 8)).toMatchObject([
      { title: 'Llama', start: 3, length: 7 },
    ])
  })

  it('is untitled and empty where the answer names no note and no run', async () => {
    search.searchPassages.mockResolvedValue({
      found: [{ path: 'library/book.epub', text: 'grass', line: 0, spans: [] }],
    })

    expect(await searchCore.search('grass', 'words', 8)).toMatchObject([
      { title: '', start: 0, length: 0 },
    ])
  })
})
