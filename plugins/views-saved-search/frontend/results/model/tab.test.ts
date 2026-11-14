/**
 * One saved search tab over a scripted service: it reads the cached results on each change of the
 * watch, keeps the last change, opens a result at its range, saves the query of the note, and
 * lets go of the watch and the calls when it closes. It never asks the backend to run the search.
 */
import { describe, expect, it, vi } from 'vitest'
import {
  COMPUTED,
  HARBOR,
  HARBOR_REFRESHED,
  NOTE_PATH,
} from '../fixtures/responses'
import { createScriptedService } from '../fixtures/service'
import { createSavedSearchTab } from './tab'

/** Lets every pending promise and stream step of the tab run. */
const settle = (): Promise<void> => new Promise((resolve) => setTimeout(resolve, 0))

const LATER = { computedAt: '2026-11-14T10:05:00Z', isRunning: false }

const setup = () => {
  const service = createScriptedService(HARBOR)
  const openAt = vi.fn()
  const tab = createSavedSearchTab(NOTE_PATH, { service, openAt, locale: 'en-GB' })
  return { service, openAt, tab, state: tab.state }
}

const CHANGE = {
  query: 'kind in ["note", "pdf"]\n&& meaning("the lighthouse keeper in a gale")',
  rerank: 'The keeper of the lighthouse on a night of gale.',
  limit: 20,
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
    service.emit(COMPUTED)
    await settle()

    expect(state.view.value?.items.map((item) => item.title)).toStrictEqual([
      '1911',
      'Night of the gale',
      'Pier talk',
    ])
    expect(service.calls).toStrictEqual(['watchResults', 'getResults'])
    expect(service.reads).toStrictEqual([{ path: NOTE_PATH }])
  })

  it('shows the new results after the next change', async () => {
    const { service, tab, state } = setup()
    tab.follow()
    service.emit(COMPUTED)
    await settle()

    service.setAnswer(HARBOR_REFRESHED)
    service.emit(LATER)
    await settle()

    expect(state.view.value?.items).toHaveLength(4)
    expect(state.view.value?.items[3]?.title).toBe('1912')
    expect(service.calls).toStrictEqual(['watchResults', 'getResults', 'getResults'])
  })

  it('keeps the last change when two arrive during a call', async () => {
    const { service, tab, state } = setup()
    service.holdReads()
    tab.follow()
    service.emit(COMPUTED)
    await settle()

    service.setAnswer(HARBOR_REFRESHED)
    service.emit(LATER)
    await settle()
    service.releaseReads()
    await settle()

    expect(service.readSignals[0]?.aborted).toBe(true)
    expect(service.readSignals[1]?.aborted).toBe(false)
    expect(state.view.value?.items).toHaveLength(4)
  })

  it('shows the failure of getResults and keeps the last results', async () => {
    const { service, tab, state } = setup()
    tab.follow()
    service.emit(COMPUTED)
    await settle()

    service.failReads(`there is no note at ${NOTE_PATH}`)
    service.emit(LATER)
    await settle()

    expect(state.failure.value).toBe(`there is no note at ${NOTE_PATH}`)
    expect(state.view.value?.items).toHaveLength(3)
  })

  it('clears the failure once getResults answers again', async () => {
    const { service, tab, state } = setup()
    tab.follow()
    service.failReads('the window shows no vault')
    service.emit(COMPUTED)
    await settle()

    service.setAnswer(HARBOR)
    service.emit(LATER)
    await settle()

    expect(state.failure.value).toBe('')
    expect(state.view.value?.items).toHaveLength(3)
  })

  it('shows the failure of the watch', async () => {
    const { service, tab, state } = setup()

    tab.follow()
    service.failWatch('the window shows no vault')
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
    expect(service.watchSignals[0]?.aborted).toBe(false)
  })

  it('follows again after the watch ended', async () => {
    const { service, tab, state } = setup()
    tab.follow()
    service.emit(COMPUTED)
    service.endWatch()
    await settle()

    service.setAnswer(HARBOR_REFRESHED)
    tab.follow()
    service.emit(LATER)
    await settle()

    expect(service.calls.filter((call) => call === 'watchResults')).toHaveLength(2)
    expect(state.view.value?.items).toHaveLength(4)
  })

  it('does not restart a watch that ended on its own', async () => {
    const { service, tab } = setup()
    tab.follow()
    service.endWatch()
    await settle()
    await settle()

    expect(service.calls).toStrictEqual(['watchResults'])
  })

  it('aborts the watch and the call when the tab closes', async () => {
    const { service, tab } = setup()
    service.holdReads()
    tab.follow()
    service.emit(COMPUTED)
    await settle()

    tab.dispose()
    await settle()

    expect(service.watchSignals[0]?.aborted).toBe(true)
    expect(service.readSignals[0]?.aborted).toBe(true)
  })
})

describe('opening a result', () => {
  it('opens the file of a result at its range', async () => {
    const { service, openAt, tab, state } = setup()
    tab.follow()
    service.emit(COMPUTED)
    await settle()

    state.actions.openResult(0)
    state.actions.openResult(2)

    expect(openAt).toHaveBeenNthCalledWith(1, 'Logbooks/1911.pdf', [{ start: 48210, end: 50620 }])
    expect(openAt).toHaveBeenNthCalledWith(2, 'Recordings/Pier talk.m4a', [
      { start: 10233, end: 11413 },
    ])
  })

  it('opens nothing for an index without a result', async () => {
    const { service, openAt, tab, state } = setup()
    tab.follow()
    service.emit(COMPUTED)
    await settle()

    state.actions.openResult(7)

    expect(openAt).not.toHaveBeenCalled()
  })
})

describe('saving the query of the note', () => {
  it('saves the query, the rerank text and the limit of the note', async () => {
    const { service, tab, state } = setup()
    tab.follow()
    service.emit(COMPUTED)
    await settle()

    await state.actions.saveSearch(CHANGE)

    expect(service.updates).toStrictEqual([{ path: NOTE_PATH, ...CHANGE }])
  })

  it('reads the note again after a save', async () => {
    const { service, tab, state } = setup()
    tab.follow()
    service.emit(COMPUTED)
    await settle()

    await state.actions.saveSearch(CHANGE)
    await settle()

    expect(service.calls).toStrictEqual(['watchResults', 'getResults', 'updateSearch', 'getResults'])
    expect(state.view.value?.query).toBe(CHANGE.query)
    expect(state.view.value?.rerank).toBe(CHANGE.rerank)
    expect(state.view.value?.limit).toBe(20)
  })

  it('shows the problems of a query that was not saved', async () => {
    const { service, tab, state } = setup()
    tab.follow()
    service.emit(COMPUTED)
    await settle()
    service.setUpdateAnswer({
      errors: [{ line: 2, column: 4, message: "undeclared reference to 'meanin'" }],
    })

    await state.actions.saveSearch({ ...CHANGE, query: 'meanin("gale")' })
    await settle()

    expect(state.saveProblems.value).toStrictEqual(["2:4 undeclared reference to 'meanin'"])
    expect(service.calls).toStrictEqual(['watchResults', 'getResults', 'updateSearch'])
    expect(state.view.value?.query).toBe(HARBOR.query)
  })

  it('clears the problems of the last save after a good save', async () => {
    const { service, tab, state } = setup()
    tab.follow()
    service.emit(COMPUTED)
    await settle()
    service.setUpdateAnswer({
      errors: [{ line: 2, column: 4, message: "undeclared reference to 'meanin'" }],
    })
    await state.actions.saveSearch({ ...CHANGE, query: 'meanin("gale")' })

    service.setUpdateAnswer({ errors: [] })
    await state.actions.saveSearch(CHANGE)

    expect(state.saveProblems.value).toStrictEqual([])
  })

  it('shows the failure of a save', async () => {
    const { service, tab, state } = setup()
    tab.follow()
    service.emit(COMPUTED)
    await settle()
    service.setUpdateAnswer('the note changed while it was updated; try again')

    await state.actions.saveSearch(CHANGE)

    expect(state.failure.value).toBe('the note changed while it was updated; try again')
    expect(state.isSaving.value).toBe(false)
  })

  it('is saving until the backend answers', async () => {
    const { service, tab, state } = setup()
    tab.follow()
    service.emit(COMPUTED)
    await settle()

    const saving = state.actions.saveSearch(CHANGE)
    const during = state.isSaving.value
    await saving

    expect(during).toBe(true)
    expect(state.isSaving.value).toBe(false)
  })
})
