/**
 * The tab kind of saved search notes: a tab per note path, titled by the note, following its
 * results from the moment it opens and aborting the watch when it closes.
 */
import { describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { COMPUTED, HARBOR, HARBOR_REFRESHED, NOTE_PATH } from '../fixtures/responses'
import { createScriptedService } from '../fixtures/service'
import { createSavedSearchKind } from './kind'
import { SAVED_SEARCH } from './tabKind'

const settle = (): Promise<void> => new Promise((resolve) => setTimeout(resolve, 0))

const TAB_ID = 'tab-1'

const setup = () => {
  const service = createScriptedService(HARBOR)
  const openAt = vi.fn()
  const kind = createSavedSearchKind({
    service,
    openAt,
    notify: vi.fn(),
    locale: 'en-GB',
    getNow: () => 0,
    component: defineComponent({ render: () => null }),
  })
  return { service, openAt, kind }
}

const openTab = (kind: ReturnType<typeof setup>['kind']) => {
  if (!kind.open) throw new Error('the kind opens no tab')
  return kind.open(NOTE_PATH, TAB_ID)
}

describe('the saved search tab kind', () => {
  it('declares an icon, so that its tab shows one in the tab strip', () => {
    const { kind } = setup()

    expect(kind.icon).toBeDefined()
  })

  it('registers the saved-search kind', () => {
    const { kind } = setup()

    expect(SAVED_SEARCH).toBe('saved-search')
    expect(kind.kind).toBe('saved-search')
  })

  it('opens one tab per note path', () => {
    const { kind } = setup()

    expect(kind.identity?.(NOTE_PATH)).toBe(NOTE_PATH)
  })

  it('keeps no state for a restart', () => {
    const { kind } = setup()

    expect(kind.getState).toBeUndefined()
  })

  it('follows the results when the tab opens', async () => {
    const { service, kind } = setup()

    const state = openTab(kind)
    service.watch.emit(COMPUTED)
    await settle()

    expect(state.path.value).toBe(NOTE_PATH)
    expect(state.view.value?.items).toHaveLength(3)
  })

  it('titles the tab with the file name before the results are known', () => {
    const { kind } = setup()

    const state = openTab(kind)

    expect(kind.getTitle?.(state)).toBe('Harbor master - winter storms')
  })

  it('titles the tab with the note title once known', async () => {
    const { service, kind } = setup()
    const state = openTab(kind)

    service.watch.emit(COMPUTED)
    await settle()

    expect(kind.getTitle?.(state)).toBe('Harbor master: winter storms')
  })

  it('disposes the tab on close', async () => {
    const { service, kind } = setup()
    const state = openTab(kind)
    await settle()

    const isClosing = kind.onClose?.(state, TAB_ID)

    expect(isClosing).toBe(true)
    expect(service.signals.watches[0]?.aborted).toBe(true)
  })

  it('disposes the tab when the window is destroyed', async () => {
    const { service, kind } = setup()
    const state = openTab(kind)
    await settle()

    kind.onDestroy?.(state, TAB_ID)

    expect(service.signals.watches[0]?.aborted).toBe(true)
  })

  it('follows again when the tab is shown after the watch ended', async () => {
    const { service, kind } = setup()
    const state = openTab(kind)
    service.watch.end()
    await settle()

    service.results.set(HARBOR_REFRESHED)
    kind.onShow?.(state, TAB_ID)
    service.watch.emit(COMPUTED)
    await settle()

    expect(service.calls.filter((call) => call === 'watchResults')).toHaveLength(2)
    expect(state.view.value?.items).toHaveLength(4)
  })

  it('does not follow again when the tab is shown while the watch runs', async () => {
    const { service, kind } = setup()
    const state = openTab(kind)
    await settle()

    kind.onShow?.(state, TAB_ID)
    await settle()

    expect(service.calls.filter((call) => call === 'watchResults')).toHaveLength(1)
  })
})
