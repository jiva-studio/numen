/**
 * One saved search tab over a scripted service: it reads the cached results on each change of the
 * watch, keeps the last change, opens a result at its range, saves the query of the note, and
 * aborts the watch and the calls when it closes. It never asks the backend to run the search.
 */
import { describe, expect, it, vi } from 'vitest'
import {
  COMPUTED,
  FAILED_LAST_RUN,
  FUNCTIONS,
  HARBOR,
  HARBOR_REFRESHED,
  HARBOR_TWO_LOGBOOK_PASSAGES,
  MISSPELLED,
  NOTE_PATH,
} from '../fixtures/responses'
import type { ListSearchFunctionsResponse } from '../api/types'
import { createScriptedService } from '../fixtures/service'
import { createSavedSearchTab } from './tab'

const settle = (): Promise<void> => new Promise((resolve) => setTimeout(resolve, 0))

const LATER = { computedAt: '2026-11-14T10:05:00Z', isRunning: false }

let catalog: ListSearchFunctionsResponse | string = { functions: [] }

const setup = () => {
  catalog = { functions: [] }
  const service = createScriptedService(HARBOR, () => catalog)
  const openAt = vi.fn()
  const notify = vi.fn()
  const tab = createSavedSearchTab(NOTE_PATH, { service, openAt, notify, locale: 'en-GB' })
  return { service, openAt, notify, tab, state: tab.state }
}

const CHANGE = {
  query: 'kind in ["note", "pdf"]\n&& meaning("the lighthouse keeper in a gale")',
}

describe('following the results of a saved search', () => {
  it('reads nothing before the first change', async () => {
    const { service, tab, state } = setup()

    tab.follow()
    await settle()

    expect(service.calls).toStrictEqual(['watchResults'])
    expect(state.view.value).toBeNull()
  })

  it('shows the cached results after the first change', async () => {
    const { service, tab, state } = setup()

    tab.follow()
    service.watch.emit(COMPUTED)
    await settle()

    expect(state.view.value?.items.map((item) => item.title)).toStrictEqual([
      '1911',
      'Night of the gale',
      'Pier plan',
    ])
    expect(service.calls).toStrictEqual(['watchResults', 'getResults'])
    expect(service.reads).toStrictEqual([{ path: NOTE_PATH }])
  })

  it('shows the new results after the next change', async () => {
    const { service, tab, state } = setup()
    tab.follow()
    service.watch.emit(COMPUTED)
    await settle()

    service.results.set(HARBOR_REFRESHED)
    service.watch.emit(LATER)
    await settle()

    expect(state.view.value?.items).toHaveLength(4)
    expect(state.view.value?.items[3]?.title).toBe('1912')
    expect(service.calls).toStrictEqual(['watchResults', 'getResults', 'getResults'])
  })

  it('keeps the last change when two arrive during a call', async () => {
    const { service, tab, state } = setup()
    service.results.hold()
    tab.follow()
    service.watch.emit(COMPUTED)
    await settle()

    service.results.set(HARBOR_REFRESHED)
    service.watch.emit(LATER)
    await settle()
    service.results.release()
    await settle()

    expect(service.signals.reads[0]?.aborted).toBe(true)
    expect(service.signals.reads[1]?.aborted).toBe(false)
    expect(state.view.value?.items).toHaveLength(4)
  })

  it('shows the failure of getResults and keeps the last results', async () => {
    const { service, tab, state } = setup()
    tab.follow()
    service.watch.emit(COMPUTED)
    await settle()

    service.results.fail(`there is no note at ${NOTE_PATH}`)
    service.watch.emit(LATER)
    await settle()

    expect(state.failure.value).toBe(`there is no note at ${NOTE_PATH}`)
    expect(state.view.value?.items).toHaveLength(3)
  })

  it('clears the failure once getResults answers again', async () => {
    const { service, tab, state } = setup()
    tab.follow()
    service.results.fail('the window shows no vault')
    service.watch.emit(COMPUTED)
    await settle()

    service.results.set(HARBOR)
    service.watch.emit(LATER)
    await settle()

    expect(state.failure.value).toBe('')
    expect(state.view.value?.items).toHaveLength(3)
  })

  it('shows the failure of the watch', async () => {
    const { service, tab, state } = setup()

    tab.follow()
    service.watch.fail('the window shows no vault')
    await settle()

    expect(state.failure.value).toBe('the window shows no vault')
  })

  it('does not follow again while the watch runs', async () => {
    const { service, tab } = setup()
    tab.follow()
    await settle()

    tab.follow()
    await settle()

    expect(service.calls.filter((call) => call === 'watchResults')).toHaveLength(1)
    expect(service.signals.watches[0]?.aborted).toBe(false)
  })

  it('follows again after the watch ended', async () => {
    const { service, tab, state } = setup()
    tab.follow()
    service.watch.emit(COMPUTED)
    service.watch.end()
    await settle()

    service.results.set(HARBOR_REFRESHED)
    tab.follow()
    service.watch.emit(LATER)
    await settle()

    expect(service.calls.filter((call) => call === 'watchResults')).toHaveLength(2)
    expect(state.view.value?.items).toHaveLength(4)
  })

  it('does not restart a watch that ended on its own', async () => {
    const { service, tab } = setup()
    tab.follow()
    service.watch.end()
    await settle()
    await settle()

    expect(service.calls).toStrictEqual(['watchResults'])
  })

  it('aborts the watch and the call when the tab closes', async () => {
    const { service, tab } = setup()
    service.results.hold()
    tab.follow()
    service.watch.emit(COMPUTED)
    await settle()

    tab.dispose()
    await settle()

    expect(service.signals.watches[0]?.aborted).toBe(true)
    expect(service.signals.reads[0]?.aborted).toBe(true)
  })
})

describe('opening a result', () => {
  it('opens a passage at its range and a whole file at its start', async () => {
    const { service, openAt, tab, state } = setup()
    tab.follow()
    service.watch.emit(COMPUTED)
    await settle()

    state.actions.openResult(0)
    state.actions.openResult(2)

    expect(openAt).toHaveBeenNthCalledWith(1, 'Logbooks/1911.pdf', [{ start: 48210, end: 50620 }])
    expect(openAt).toHaveBeenNthCalledWith(2, 'Harbor/Pier plan.png', [{ start: 0, end: 0 }])
  })

  it('opens nothing for an index without a result', async () => {
    const { service, openAt, tab, state } = setup()
    tab.follow()
    service.watch.emit(COMPUTED)
    await settle()

    state.actions.openResult(7)

    expect(openAt).not.toHaveBeenCalled()
  })
})

describe('saving the query of the note', () => {
  it('saves the query of the note', async () => {
    const { service, tab, state } = setup()
    tab.follow()
    service.watch.emit(COMPUTED)
    await settle()

    await state.actions.saveSearch(CHANGE)

    expect(service.updates).toStrictEqual([{ path: NOTE_PATH, ...CHANGE }])
  })

  it('reads the note again after a save', async () => {
    const { service, tab, state } = setup()
    tab.follow()
    service.watch.emit(COMPUTED)
    await settle()

    await state.actions.saveSearch(CHANGE)
    await settle()

    expect(service.calls).toStrictEqual([
      'watchResults',
      'getResults',
      'updateSearch',
      'getResults',
    ])
    expect(state.view.value?.query).toBe(CHANGE.query)
  })

  it('shows the problems of a query that was not saved', async () => {
    const { service, tab, state } = setup()
    tab.follow()
    service.watch.emit(COMPUTED)
    await settle()
    service.setUpdateAnswer({
      errors: [{ line: 2, column: 4, message: "undeclared reference to 'meanin'" }],
    })

    await state.actions.saveSearch({ ...CHANGE, query: 'meanin("gale")' })
    await settle()

    expect(state.queryProblems.value).toStrictEqual([
      { line: 2, column: 4, message: "undeclared reference to 'meanin'" },
    ])
    expect(service.calls).toStrictEqual(['watchResults', 'getResults', 'updateSearch'])
    expect(state.view.value?.query).toBe(HARBOR.query)
  })

  it('clears the problems of the last save after a good save', async () => {
    const { service, tab, state } = setup()
    tab.follow()
    service.watch.emit(COMPUTED)
    await settle()
    service.setUpdateAnswer({
      errors: [{ line: 2, column: 4, message: "undeclared reference to 'meanin'" }],
    })
    await state.actions.saveSearch({ ...CHANGE, query: 'meanin("gale")' })

    service.setUpdateAnswer({ errors: [] })
    await state.actions.saveSearch(CHANGE)

    expect(state.queryProblems.value).toStrictEqual([])
  })

  it('shows the failure of a save', async () => {
    const { service, tab, state } = setup()
    tab.follow()
    service.watch.emit(COMPUTED)
    await settle()
    service.setUpdateAnswer('the note changed while it was updated; try again')

    await state.actions.saveSearch(CHANGE)

    expect(state.failure.value).toBe('the note changed while it was updated; try again')
  })
})

describe('filtering the results', () => {
  const setupWith = async (answer = HARBOR_TWO_LOGBOOK_PASSAGES) => {
    const found = setup()
    found.service.results.set(answer)
    found.tab.follow()
    found.service.watch.emit(COMPUTED)
    await settle()
    return found
  }

  it('shows every result in the order of the core before a filter', async () => {
    const { state } = await setupWith()

    expect(state.items.value.map((item) => item.index)).toStrictEqual([0, 1, 2, 3])
  })

  it('narrows the results by the filter text and starts no search', async () => {
    const { service, state } = await setupWith()

    state.actions.setFilter('quay')

    expect(state.items.value.map((item) => item.text)).toStrictEqual([
      'Ice on the quay; no boat went out for a week.',
    ])
    expect(service.calls).toStrictEqual(['watchResults', 'getResults'])
  })

  it('hides the results of an unchecked source and shows them again when checked', async () => {
    const { state } = await setupWith()

    state.actions.setSourceShown('Logbooks/1911.pdf', false)
    const hidden = state.items.value.map((item) => item.source)
    state.actions.setSourceShown('Logbooks/1911.pdf', true)

    expect(hidden).toStrictEqual(['Harbor/Night of the gale.md', 'Harbor/Pier plan.png'])
    expect(state.items.value).toHaveLength(4)
  })

  it('shows the results of every hidden source again', async () => {
    const { state } = await setupWith()

    state.actions.setSourceShown('Logbooks/1911.pdf', false)
    state.actions.setSourceShown('Harbor/Pier plan.png', false)
    state.actions.showAllSources()

    expect(state.items.value).toHaveLength(4)
    expect(state.hiddenSources.value.size).toBe(0)
  })

  it('opens the result the filter leaves by its index in the answer', async () => {
    const { openAt, state } = await setupWith()

    state.actions.setFilter('quay')
    const [item] = state.items.value
    if (!item) throw new Error('the filter left no result')
    state.actions.openResult(item.index)

    expect(openAt).toHaveBeenCalledWith('Logbooks/1911.pdf', [{ start: 190400, end: 191220 }])
  })
})

describe('the notice of a failed run', () => {
  it('shows no notice for a failure present when the tab opens', async () => {
    const { service, notify, tab } = setup()
    service.results.set(FAILED_LAST_RUN)
    tab.follow()
    service.watch.emit(COMPUTED)
    await settle()

    expect(notify).not.toHaveBeenCalled()
  })

  it('shows one notice for a new failure, with the problems of the query', async () => {
    const { service, notify, tab } = setup()
    tab.follow()
    service.watch.emit(COMPUTED)
    await settle()

    service.results.set(MISSPELLED)
    service.watch.emit(LATER)
    await settle()
    service.watch.emit(LATER)
    await settle()

    expect(notify).toHaveBeenCalledTimes(1)
    expect(notify).toHaveBeenCalledWith(
      'Harbor master: winter storms: the query has problems',
    )
  })
})

describe('editing the query of a saved search', () => {
  it('reads the catalog of the search language when the editor opens, once', async () => {
    const { service, state } = setup()
    catalog = FUNCTIONS

    state.actions.setEditing(true)
    await settle()
    state.actions.setEditing(false)
    state.actions.setEditing(true)
    await settle()

    expect(state.functions.value.map((fn) => fn.name)).toStrictEqual([
      'words',
      'meaning',
      'tags',
      'top',
      'rerank',
    ])
    expect(service.calls.filter((call) => call === 'listSearchFunctions')).toHaveLength(1)
  })

  it('reads the catalog again on the next open after it failed', async () => {
    const { state } = setup()
    catalog = 'the catalog is not ready'
    state.actions.setEditing(true)
    await settle()
    state.actions.setEditing(false)
    catalog = FUNCTIONS

    state.actions.setEditing(true)
    await settle()

    expect(state.functions.value).toHaveLength(5)
  })

  it('shows the failure of reading the catalog', async () => {
    const { state } = setup()
    catalog = 'the catalog is not ready'

    state.actions.setEditing(true)
    await settle()

    expect(state.failure.value).toBe('the catalog is not ready')
    expect(state.functions.value).toStrictEqual([])
  })

  it('gives the editor the problems of the saved query', async () => {
    const { service, tab, state } = setup()
    service.results.set(MISSPELLED)
    tab.follow()
    service.watch.emit(COMPUTED)
    await settle()

    state.actions.setEditing(true)

    expect(state.queryProblems.value).toStrictEqual(MISSPELLED.errors)
  })

  it('gives the editor no problem of a provider, which has no place in the query', async () => {
    const { service, tab, state } = setup()
    service.results.set({
      ...MISSPELLED,
      errors: [{ line: 0, column: 0, message: 'the reranking service did not answer' }],
    })
    tab.follow()
    service.watch.emit(COMPUTED)
    await settle()

    expect(state.queryProblems.value).toStrictEqual([])
  })

  it('gives the editor the problems of the last save over those of the saved query', async () => {
    const { service, tab, state } = setup()
    service.results.set(MISSPELLED)
    tab.follow()
    service.watch.emit(COMPUTED)
    await settle()
    state.actions.setEditing(true)
    service.setUpdateAnswer({
      errors: [{ line: 1, column: 1, message: 'unknown function meanin' }],
    })

    await state.actions.saveSearch({ query: 'meanin("gale")' })

    expect(state.queryProblems.value).toStrictEqual([
      { line: 1, column: 1, message: 'unknown function meanin' },
    ])
  })

  it('forgets the problems of the last save when the editor closes', async () => {
    const { service, tab, state } = setup()
    tab.follow()
    service.watch.emit(COMPUTED)
    await settle()
    state.actions.setEditing(true)
    service.setUpdateAnswer({
      errors: [{ line: 1, column: 1, message: 'unknown function meanin' }],
    })
    await state.actions.saveSearch({ query: 'meanin("gale")' })

    state.actions.setEditing(false)

    expect(state.queryProblems.value).toStrictEqual([])
    expect(state.isEditing.value).toBe(false)
  })

  it('keeps the editor open after a good save and reads the results again', async () => {
    const { service, tab, state } = setup()
    tab.follow()
    service.watch.emit(COMPUTED)
    await settle()
    state.actions.setEditing(true)

    await state.actions.saveSearch(CHANGE)

    expect(state.isEditing.value).toBe(true)
    expect(service.updates).toStrictEqual([{ path: NOTE_PATH, query: CHANGE.query }])
    expect(service.calls.filter((call) => call === 'getResults')).toHaveLength(2)
  })
})
