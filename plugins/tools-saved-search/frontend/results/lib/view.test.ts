/**
 * What the tab draws for an answer of GetResults: the status line, the failed last run, the
 * problems, the warnings, the results with their places, and the words for no result.
 */
import { describe, expect, it } from 'vitest'
import { WORDS } from '../../words'
import {
  FAILED_LAST_RUN,
  FAILURE_REASON,
  FUSED,
  FUSED_WARNING,
  HARBOR,
  MISSPELLED,
  NEVER_COMPUTED,
  NOTHING_FOUND,
  RUNNING,
  STALE,
} from '../fixtures/responses'
import type { GetResultsResponse } from '../api/types'
import { getResultsView } from './view'

const LOCALE = 'en-GB'

/** 12:00 UTC is 14 November in every zone of the year the fixtures are of. */
const NOON = '2026-11-14T12:00:00Z'
const TIME = String.raw`14 Nov 2026, \d\d:\d\d`

const at = (response: GetResultsResponse): GetResultsResponse => ({ ...response, computedAt: NOON })

describe('the status line', () => {
  it('says the results are being computed before the first run', () => {
    expect(getResultsView(NEVER_COMPUTED, WORDS, LOCALE).status).toBe('Computing the results…')
    expect(getResultsView({ ...NEVER_COMPUTED, isRunning: true }, WORDS, LOCALE).status).toBe(
      'Computing the results…',
    )
  })

  it('says the search is updating while it runs', () => {
    expect(getResultsView(at(RUNNING), WORDS, LOCALE).status).toMatch(
      new RegExp(`^Updating… · computed ${TIME}$`),
    )
  })

  it('says the search will update when it is stale', () => {
    expect(getResultsView(at(STALE), WORDS, LOCALE).status).toMatch(
      new RegExp(`^Will update · computed ${TIME}$`),
    )
  })

  it('names when the results were computed', () => {
    expect(getResultsView(at(HARBOR), WORDS, LOCALE).status).toMatch(
      new RegExp(`^Computed ${TIME}$`),
    )
  })

  it('names the time and the reason of a failed last run', () => {
    const view = getResultsView({ ...FAILED_LAST_RUN, failedAt: NOON }, WORDS, LOCALE)

    expect(view.failedRun).toMatch(new RegExp(`^Last run failed ${TIME}: ${FAILURE_REASON}$`))
  })

  it('keeps the time of the last good run when the last run failed', () => {
    const view = getResultsView(
      { ...FAILED_LAST_RUN, computedAt: NOON, failedAt: '2026-11-15T12:00:00Z' },
      WORDS,
      LOCALE,
    )

    expect(view.status).toMatch(new RegExp(`^Computed ${TIME}$`))
    expect(view.items).toHaveLength(3)
  })

  it('names no failed run when the last run succeeded', () => {
    expect(getResultsView(HARBOR, WORDS, LOCALE).failedRun).toBe('')
  })
})

describe('the query of the note', () => {
  it('shows the query, the rerank text and the limit as the note has them', () => {
    const view = getResultsView(HARBOR, WORDS, LOCALE)

    expect(view.title).toBe('Harbor master: winter storms')
    expect(view.query).toBe(HARBOR.query)
    expect(view.rerank).toBe(HARBOR.rerank)
    expect(view.limit).toBe(30)
  })
})

describe('the problems and the warnings', () => {
  it('lists the problems with line and column', () => {
    const view = getResultsView(MISSPELLED, WORDS, LOCALE)

    expect(view.problems).toStrictEqual(["9:4 undeclared reference to 'meanin'"])
  })

  it('shows no result while the note has problems', () => {
    const view = getResultsView({ ...MISSPELLED, results: HARBOR.results }, WORDS, LOCALE)

    expect(view.items).toStrictEqual([])
    expect(view.emptyText).toBe('')
  })

  it('lists the warnings of the run', () => {
    expect(getResultsView(FUSED, WORDS, LOCALE).warnings).toStrictEqual([FUSED_WARNING])
  })
})

describe('the results', () => {
  it('writes the place of each result', () => {
    const view = getResultsView(HARBOR, WORDS, LOCALE)

    expect(view.items.map((item) => item.place)).toStrictEqual([
      'Page 14 of 212',
      '',
      '12:34–13:32',
    ])
  })

  it('keeps the title and the text of each result in order', () => {
    const view = getResultsView(HARBOR, WORDS, LOCALE)

    expect(view.items.map((item) => item.title)).toStrictEqual([
      '1911',
      'Night of the gale',
      'Pier talk',
    ])
    expect(view.items[1]?.text).toBe(
      'The gale broke two moorings before the harbor master reached the pier.',
    )
  })

  it('hides the path of a result titled as its file name', () => {
    const view = getResultsView(HARBOR, WORDS, LOCALE)

    expect(view.items.map((item) => item.path)).toStrictEqual(['', '', ''])
  })

  it('shows the path of a result whose title is not its file name', () => {
    const [first] = HARBOR.results
    if (!first) throw new Error('the fixture has no result')
    const view = getResultsView(
      { ...HARBOR, results: [{ ...first, title: 'Logbook of the harbor, 1911' }] },
      WORDS,
      LOCALE,
    )

    expect(view.items[0]?.path).toBe('Logbooks/1911.pdf')
  })

  it('says nothing matches when the run found no result', () => {
    expect(getResultsView(NOTHING_FOUND, WORDS, LOCALE).emptyText).toBe(
      'No passage matches the query.',
    )
  })

  it('says nothing about matches before the first run', () => {
    expect(getResultsView(NEVER_COMPUTED, WORDS, LOCALE).emptyText).toBe('')
  })

  it('says nothing about matches when there are results', () => {
    expect(getResultsView(HARBOR, WORDS, LOCALE).emptyText).toBe('')
  })
})
