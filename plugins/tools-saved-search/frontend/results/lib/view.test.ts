/**
 * What the tab draws for an answer of GetResults: the state of the run and its details, the
 * problems, the warnings, the results with their places, the sources, the filter, and the words for
 * no result.
 */
import { describe, expect, it } from 'vitest'
import { WORDS } from '../../words'
import {
  FAILED_LAST_RUN,
  FAILURE_REASON,
  FUSED,
  HARBOR,
  HARBOR_TWO_LOGBOOK_PASSAGES,
  MISSPELLED,
  NEVER_COMPUTED,
  NOTHING_FOUND,
  RUNNING,
  STALE,
  WARNING,
} from '../fixtures/responses'
import type { GetResultsResponse } from '../api/types'
import { filterItems, getFilterEmpty, getResultsView } from './view'

const LOCALE = 'en-GB'

/** 12:00 UTC is 14 November in every zone of the year the fixtures are of. */
const NOON = '2026-11-14T12:00:00Z'
const TIME = String.raw`14 Nov 2026, \d\d:\d\d`

const at = (response: GetResultsResponse): GetResultsResponse => ({ ...response, computedAt: NOON })

describe('the state of the run', () => {
  it('says the results are being computed before the first run', () => {
    for (const response of [NEVER_COMPUTED, { ...NEVER_COMPUTED, isRunning: true }]) {
      expect(getResultsView(response, WORDS, LOCALE).status).toStrictEqual({
        state: 'running',
        label: 'Computing the results…',
        details: ['Computing the results…'],
      })
    }
  })

  it('says the search is updating while it runs, with the time of the results', () => {
    const { status } = getResultsView(at(RUNNING), WORDS, LOCALE)

    expect(status.state).toBe('running')
    expect(status.label).toBe('Updating…')
    expect(status.details).toStrictEqual([
      'Updating…',
      expect.stringMatching(new RegExp(`^Computed ${TIME}$`)),
    ])
  })

  it('says the search will update when it is stale', () => {
    const { status } = getResultsView(at(STALE), WORDS, LOCALE)

    expect(status.state).toBe('stale')
    expect(status.label).toBe('Will update')
    expect(status.details).toStrictEqual([
      'Will update',
      expect.stringMatching(new RegExp(`^Computed ${TIME}$`)),
    ])
  })

  it('says the results are up to date and names when they were computed', () => {
    const { status } = getResultsView(at(HARBOR), WORDS, LOCALE)

    expect(status.state).toBe('fresh')
    expect(status.label).toBe('Up to date')
    expect(status.details).toStrictEqual([
      'Up to date',
      expect.stringMatching(new RegExp(`^Computed ${TIME}$`)),
    ])
  })

  it('names the time and the reason of a failed last run, and the time of the last good run', () => {
    const { status } = getResultsView(
      { ...FAILED_LAST_RUN, computedAt: '2026-11-14T09:30:00Z', failedAt: NOON },
      WORDS,
      LOCALE,
    )

    expect(status.state).toBe('failed')
    expect(status.label).toBe('Last run failed')
    expect(status.details).toStrictEqual([
      expect.stringMatching(new RegExp(`^Last run failed ${TIME}: ${FAILURE_REASON}$`)),
      expect.stringMatching(new RegExp(`^Computed ${TIME}$`)),
    ])
  })

  it('says the search will update after a failed last run that is stale', () => {
    const { status } = getResultsView({ ...FAILED_LAST_RUN, isStale: true }, WORDS, LOCALE)

    expect(status.state).toBe('failed')
    expect(status.details.at(-1)).toBe('Will update')
  })

  it('names the failed last run while the next run is in progress', () => {
    const { status } = getResultsView({ ...FAILED_LAST_RUN, isRunning: true }, WORDS, LOCALE)

    expect(status.state).toBe('running')
    expect(status.details.at(-1)).toMatch(
      new RegExp(`^Last run failed ${TIME}: ${FAILURE_REASON}$`),
    )
  })

  it('names no failed run when a good run followed it', () => {
    const { status } = getResultsView(
      { ...FAILED_LAST_RUN, computedAt: '2026-11-15T12:00:00Z', failedAt: NOON },
      WORDS,
      LOCALE,
    )

    expect(status.state).toBe('fresh')
    expect(status.details.some((line) => line.startsWith('Last run failed'))).toBe(false)
  })

  it('keeps the results of the last good run when the last run failed', () => {
    expect(getResultsView(FAILED_LAST_RUN, WORDS, LOCALE).items).toHaveLength(3)
  })
})

describe('the query of the note', () => {
  it('shows the title and the query as the note has them', () => {
    const view = getResultsView(HARBOR, WORDS, LOCALE)

    expect(view.title).toBe('Harbor master: winter storms')
    expect(view.query).toBe(HARBOR.query)
  })
})

describe('the problems and the warnings', () => {
  it('names the problems with line and column in the details of the run', () => {
    const view = getResultsView(MISSPELLED, WORDS, LOCALE)

    expect(view.status.details).toContain('2:4 unknown function meanin (did you mean meaning?)')
  })

  it('keeps the last good results while the query has problems', () => {
    const view = getResultsView(MISSPELLED, WORDS, LOCALE)

    expect(view.items).toHaveLength(3)
    expect(view.status.state).toBe('failed')
  })

  it('lists the warnings of the run', () => {
    expect(getResultsView(FUSED, WORDS, LOCALE).warnings).toStrictEqual([WARNING])
  })
})

describe('the results', () => {
  it('writes the place of each passage', () => {
    const view = getResultsView(HARBOR, WORDS, LOCALE)

    expect(view.items.map((item) => item.place)).toStrictEqual(['Page 14', '', ''])
  })

  it('keeps the title and the text of each result in the order of the core', () => {
    const view = getResultsView(HARBOR, WORDS, LOCALE)

    expect(view.items.map((item) => item.title)).toStrictEqual([
      '1911',
      'Night of the gale',
      'Pier plan',
    ])
    expect(view.items[1]?.text).toBe(
      'The gale broke two moorings before the harbor master reached the pier.',
    )
  })

  it('draws a whole file as a file row with its path and kind', () => {
    const file = getResultsView(HARBOR, WORDS, LOCALE).items[2]

    expect(file).toMatchObject({
      isWholeFile: true,
      title: 'Pier plan',
      path: 'Harbor/Pier plan.png',
      kind: 'image',
      text: '',
      place: '',
    })
  })

  it('hides the path of a passage titled as its file name', () => {
    const view = getResultsView(HARBOR, WORDS, LOCALE)

    expect(view.items.slice(0, 2).map((item) => item.path)).toStrictEqual(['', ''])
  })

  it('shows the path of a passage whose title is not its file name', () => {
    const [first] = HARBOR.results
    if (!first) throw new Error('the fixture has no result')
    const view = getResultsView(
      { ...HARBOR, results: [{ ...first, title: 'Logbook of the harbor, 1911' }] },
      WORDS,
      LOCALE,
    )

    expect(view.items[0]?.path).toBe('Logbooks/1911.pdf')
  })

  it('lists each file the results come from once, in the order of its first result', () => {
    const view = getResultsView(HARBOR_TWO_LOGBOOK_PASSAGES, WORDS, LOCALE)

    expect(view.sources.map((source) => source.path)).toStrictEqual([
      'Logbooks/1911.pdf',
      'Harbor/Night of the gale.md',
      'Harbor/Pier plan.png',
    ])
  })

  it('says nothing matches when the run found no result', () => {
    expect(getResultsView(NOTHING_FOUND, WORDS, LOCALE).emptyText).toBe('The query found nothing')
  })

  it('says nothing about matches before the first run', () => {
    expect(getResultsView(NEVER_COMPUTED, WORDS, LOCALE).emptyText).toBe('')
  })

  it('says nothing about matches when there are results', () => {
    expect(getResultsView(HARBOR, WORDS, LOCALE).emptyText).toBe('')
  })
})

describe('the filter over the results', () => {
  const { items } = getResultsView(HARBOR_TWO_LOGBOOK_PASSAGES, WORDS, LOCALE)
  const titlesOf = (filter: string, hidden: readonly string[] = []): string[] =>
    filterItems(items, filter, new Set(hidden)).map((item) => `${item.title}@${item.index}`)

  it('keeps every result for an empty filter', () => {
    expect(titlesOf('  ')).toHaveLength(4)
  })

  it('keeps the passages whose text holds the filter, without regard to case', () => {
    expect(titlesOf('MOORINGS')).toStrictEqual(['Night of the gale@1'])
  })

  it('keeps the results whose title holds the filter', () => {
    expect(titlesOf('pier')).toStrictEqual(['Night of the gale@1', 'Pier plan@2'])
  })

  it('keeps the results whose place holds the filter', () => {
    expect(titlesOf('page 52')).toStrictEqual(['1911@3'])
  })

  it('keeps the results whose file name holds the filter', () => {
    expect(titlesOf('.png')).toStrictEqual(['Pier plan@2'])
  })

  it('leaves out the results of a hidden source', () => {
    expect(titlesOf('', ['Logbooks/1911.pdf'])).toStrictEqual([
      'Night of the gale@1',
      'Pier plan@2',
    ])
  })
})

describe('the results the filter leaves out', () => {
  const sources = getResultsView(HARBOR, WORDS, LOCALE).sources

  it('says every source is hidden and offers to show them all', () => {
    const hidden = new Set(sources.map((source) => source.path))

    expect(getFilterEmpty(sources, hidden, 'gale', WORDS)).toStrictEqual({
      reason: 'sources',
      text: 'All sources are hidden',
      actionText: 'Show all sources',
    })
  })

  it('names the filter text no result holds and offers to clear it', () => {
    expect(getFilterEmpty(sources, new Set(), '  lighthouse ', WORDS)).toStrictEqual({
      reason: 'filter',
      text: 'No result matches “lighthouse”',
      actionText: 'Clear filter',
    })
  })
})
