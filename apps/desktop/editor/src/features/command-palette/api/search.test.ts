/**
 * What a search answers, read into the window's own words: a part of a match
 * the answer left out reads as nothing, and a passage is a start and a length.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { SEARCH_MODE } from '@numen/core-client'
import { installCoreAnswers } from '@numen/core-client/testing'
import { searchCore } from './search'

const search = {
  searchNames: vi.fn(),
  searchPassages: vi.fn(),
  cancelSearch: vi.fn(),
}

let restore: () => void

beforeEach(() => {
  vi.clearAllMocks()
  restore = installCoreAnswers({ search })
})

afterEach(() => {
  restore()
})

describe('a name searched for', () => {
  it('is requested for with the words typed, as many as the window shows', async () => {
    search.searchNames.mockResolvedValue({ found: [] })

    await searchCore.names('lla', 8)

    expect(search.searchNames.mock.lastCall?.[0]).toMatchObject({ query: 'lla', limit: 8 })
  })

  it('carries the note and the heading it was found at', async () => {
    search.searchNames.mockResolvedValue({
      found: [
        {
          note: { path: 'Llama.md', title: 'Llama' },
          heading: { text: 'Diet', line: 4 },
          ranges: [],
        },
      ],
    })

    expect(await searchCore.names('lla', 8)).toMatchObject([
      { path: 'Llama.md', title: 'Llama', heading: 'Diet', line: 4 },
    ])
  })

  it('is at no note and no heading where the answer names neither', async () => {
    search.searchNames.mockResolvedValue({ found: [{ ranges: [] }] })

    expect(await searchCore.names('lla', 8)).toMatchObject([
      { path: '', title: '', heading: '', line: -1 },
    ])
  })
})

describe('a passage searched for', () => {
  it('is requested for in the mode chosen', async () => {
    search.searchPassages.mockResolvedValue({ found: [] })

    await searchCore.search('grass', 'meaning', 8)

    expect(search.searchPassages.mock.lastCall?.[0]).toMatchObject({
      query: 'grass',
      limit: 8,
      mode: SEARCH_MODE.MEANING,
    })
  })

  it('starts where its run of text starts, as long as that run', async () => {
    search.searchPassages.mockResolvedValue({
      found: [
        {
          path: 'Llama.md',
          note: { title: 'Llama' },
          text: 'eats grass',
          range: { start: 3, end: 10 },
          line: 2,
          ranges: [],
        },
      ],
    })

    expect(await searchCore.search('grass', 'words', 8)).toMatchObject([
      { title: 'Llama', start: 3, length: 7 },
    ])
  })

  it('is untitled and empty where the answer names no note and no run', async () => {
    search.searchPassages.mockResolvedValue({
      found: [
        {
          path: 'library/book.epub',
          text: 'grass',
          line: 0,
          ranges: [],
          selector: '{"chapter":2}',
        },
        { path: 'library/null.epub', text: 'grass', line: 0, ranges: [], selector: 'null' },
        { path: 'library/prim.epub', text: 'grass', line: 0, ranges: [], selector: '42' },
        { path: 'library/bad.epub', text: 'grass', line: 0, ranges: [], selector: '{bad' },
        { path: 'library/empty.epub', text: 'grass', line: 0, ranges: [], selector: '' },
      ],
    })

    expect(await searchCore.search('grass', 'words', 8)).toMatchObject([
      { title: '', start: 0, length: 0, selector: { chapter: 2 } },
      { selector: null },
      { selector: null },
      { selector: null },
      { selector: null },
    ])
  })
})

describe('the kind of a note found by name', () => {
  it('is an ordinary note where the answer carries no kind at all', async () => {
    search.searchNames.mockResolvedValue({
      found: [{ note: { path: 'Entropy.md', title: 'Entropy' }, ranges: [] }],
    })
    expect((await searchCore.names('ent', 8))[0]?.type).toBe('note')
  })

  it('is drawn as whichever of four the answer names', async () => {
    search.searchNames.mockResolvedValue({
      found: [
        { note: { path: 'Ants.md', title: 'Ants' }, ranges: [], type: '' },
        { note: { path: 'Animals.md', title: 'Animals' }, ranges: [], type: 'sample' },
        { note: { path: 'Animal.md', title: 'Animal' }, ranges: [], type: 'form' },
        { note: { path: 'Daily.md', title: 'Daily' }, ranges: [], type: 'sheet' },
      ],
    })
    expect((await searchCore.names('an', 8)).map((one) => one.type)).toEqual([
      'note',
      'sample',
      'form',
      'sheet',
    ])
  })

  it('is the type a plugin declares as the answer names it', async () => {
    search.searchNames.mockResolvedValue({
      found: [{ note: { path: 'Later.md', title: 'Later' }, ranges: [], type: 'later' }],
    })
    expect((await searchCore.names('la', 8))[0]?.type).toBe('later')
  })
})

describe('the kind of a passage', () => {
  it('is the kind of the note it was read out of', async () => {
    search.searchPassages.mockResolvedValue({
      found: [
        {
          path: 'Daily.md',
          note: { path: 'Daily.md', title: 'Daily' },
          ranges: [],
          type: 'sheet',
        },
        { path: 'Ants.md', note: { path: 'Ants.md', title: 'Ants' }, ranges: [] },
      ],
    })
    const found = await searchCore.search('an', 'words', 8)
    expect(found.map((one) => one.type)).toEqual(['sheet', 'note'])
    expect(found.map((one) => one.isNote)).toEqual([true, true])
  })

  it('is no note at all where the answer names none', async () => {
    search.searchPassages.mockResolvedValue({
      found: [{ path: 'library/mahabharata.epub', text: 'war', ranges: [] }],
    })
    const found = await searchCore.search('war', 'words', 8)
    expect(found[0]?.isNote).toBe(false)
    expect(found[0]?.type).toBe('note')
  })
})

describe('a passage search the window aborts', () => {
  it('cancels the search in the core by its search id', async () => {
    search.searchPassages.mockReturnValue(new Promise(() => undefined))
    search.cancelSearch.mockResolvedValue({})
    const asking = new AbortController()

    const answer = searchCore.search('seeds', 'meaning', 8, asking.signal)
    await vi.waitFor(() => expect(search.searchPassages).toHaveBeenCalled())
    asking.abort()
    await expect(answer).rejects.toThrow()

    const searchId: unknown = search.searchPassages.mock.lastCall?.[0]?.searchId
    expect(searchId).toEqual(expect.any(String))
    expect(searchId).not.toBe('')
    await vi.waitFor(() => expect(search.cancelSearch).toHaveBeenCalledOnce())
    expect(search.cancelSearch.mock.lastCall?.[0]).toMatchObject({ searchId })
  })

  it('is not cancelled once the core has answered it', async () => {
    search.searchPassages.mockResolvedValue({ found: [] })
    const asking = new AbortController()

    await searchCore.search('seeds', 'meaning', 8, asking.signal)
    asking.abort()

    expect(search.cancelSearch).not.toHaveBeenCalled()
  })

  it('sends a new search id with each search', async () => {
    search.searchPassages.mockResolvedValue({ found: [] })

    await searchCore.search('seeds', 'meaning', 8, new AbortController().signal)
    await searchCore.search('seeds', 'words', 8, new AbortController().signal)

    const [first, second] = search.searchPassages.mock.calls.map(
      (call) => (call[0] as { searchId: string }).searchId,
    )
    expect(first).not.toBe(second)
  })
})

describe('a name search the window aborts', () => {
  it('cancels the search in the core by its search id', async () => {
    search.searchNames.mockReturnValue(new Promise(() => undefined))
    search.cancelSearch.mockResolvedValue({})
    const asking = new AbortController()

    const answer = searchCore.names('lla', 8, asking.signal)
    await vi.waitFor(() => expect(search.searchNames).toHaveBeenCalled())
    asking.abort()
    await expect(answer).rejects.toThrow()

    const searchId: unknown = search.searchNames.mock.lastCall?.[0]?.searchId
    expect(searchId).toEqual(expect.any(String))
    expect(searchId).not.toBe('')
    await vi.waitFor(() => expect(search.cancelSearch).toHaveBeenCalledOnce())
    expect(search.cancelSearch.mock.lastCall?.[0]).toMatchObject({ searchId })
  })

  it('is not cancelled once the core has answered it', async () => {
    search.searchNames.mockResolvedValue({ found: [] })
    const asking = new AbortController()

    await searchCore.names('lla', 8, asking.signal)
    asking.abort()

    expect(search.cancelSearch).not.toHaveBeenCalled()
  })
})

describe('the three questions of a query the window aborts', () => {
  it('are each cancelled in the core by their own id', async () => {
    search.searchNames.mockReturnValue(new Promise(() => undefined))
    search.searchPassages.mockReturnValue(new Promise(() => undefined))
    search.cancelSearch.mockResolvedValue({})
    const asking = new AbortController()

    const answers = [
      searchCore.names('seeds', 8, asking.signal),
      searchCore.search('seeds', 'words', 8, asking.signal),
      searchCore.search('seeds', 'meaning', 8, asking.signal),
    ]
    await vi.waitFor(() => expect(search.searchPassages).toHaveBeenCalledTimes(2))
    asking.abort()
    await Promise.allSettled(answers)

    const asked = [...search.searchNames.mock.calls, ...search.searchPassages.mock.calls].map(
      (call) => (call[0] as { searchId: string }).searchId,
    )
    await vi.waitFor(() => expect(search.cancelSearch).toHaveBeenCalledTimes(3))
    const cancelled = search.cancelSearch.mock.calls.map(
      (call) => (call[0] as { searchId: string }).searchId,
    )
    expect(new Set(cancelled)).toEqual(new Set(asked))
    expect(new Set(asked).size).toBe(3)
  })
})
