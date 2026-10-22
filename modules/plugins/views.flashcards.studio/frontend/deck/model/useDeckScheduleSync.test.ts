/**
 * Stencil listing and the schedule kept in step with the files a window holds.
 */
import { describe, expect, it, vi } from 'vitest'
import { asFailure, asValue } from '@numen/wire'
import { StopReason } from '@numen/protocol'
import { DEFAULTS, NO_BOUNDS, type Cards, type Presets } from '../../cards'
import { openNotes } from '@numen/plugin-sdk'
import { useDeckScheduleSync } from './useDeckScheduleSync'

const OFFER = { path: 'Animal.md', title: 'Animal', fields: ['Name'] }

const open = () => {
  const state = { isDown: false }
  const stencils = vi.fn<Cards['stencils']>(async () => {
    if (state.isDown) throw new Error('down')
    return { stencils: [OFFER], held: 1 }
  })
  const cards: Cards = {
    stencils,
    createDeck: async (title) => ({ ok: true, value: { path: `${title}.md` } }),
    createStencil: async (title) => ({ ok: true, value: { path: `${title}.md` } }),
    renameField: async () => asValue({ decks: [], cards: 0, notWritten: [], at: '' }),
    readDeck: async () => asFailure({ code: 'missing' as const, bound: 0 }),
    writeDeck: async () => asFailure({ code: 'missing' as const, bound: 0 }),
    readStencil: async () => asFailure('missing' as const),
    writeStencil: async () => asValue({ at: '' }),
  }
  const presets: Presets = {
    read: async () => asValue({ preset: null, at: '', bounds: NO_BOUNDS }),
    list: async () => [],
    createPreset: async () => asValue({ path: '' }),
    getDeckPreset: async () =>
      asValue({
        preset: {
          path: 'Slow.md',
          title: 'Slow',
          settings: DEFAULTS,
          problems: [],
          stops: StopReason.NOTHING,
          stopsOn: StopReason.NOTHING,
        },
        at: '',
        bounds: NO_BOUNDS,
      }),
    scheduleDeck: async () => asValue({ at: '' }),
    write: async () => asValue({ at: '' }),
    curve: async () => {
      throw new Error('unused')
    },
  }
  const store = openNotes({
    read: async () => asValue({ body: '', at: '' }),
    write: async () => asValue({ body: '', at: '' }),
  })
  return { state, stencils, store, sync: useDeckScheduleSync(cards, presets, store) }
}

describe('the stencils a window offers', () => {
  it('are none before a listing answers', () => {
    const { sync } = open()

    expect(sync.offers.value).toStrictEqual([])
    expect(sync.stencils.value).toStrictEqual([])
  })

  it('are what the last listing answered', async () => {
    const { sync } = open()

    await sync.listStencils()

    expect(sync.stencils.value).toStrictEqual([{ name: 'Animal', fields: ['Name'] }])
  })

  it('are not requested for again before a listing failed', () => {
    const { sync, stencils } = open()

    sync.listStencilsAgain()

    expect(stencils).not.toHaveBeenCalled()
  })

  it('are requested for again once a listing failed', async () => {
    const { sync, stencils, state } = open()
    state.isDown = true
    await sync.listStencils()

    sync.listStencilsAgain()

    expect(stencils).toHaveBeenCalledTimes(2)
  })

  it('are not requested for again once a listing answered after the failure', async () => {
    const { sync, stencils, state } = open()
    state.isDown = true
    await sync.listStencils()
    state.isDown = false
    await sync.listStencils()

    sync.listStencilsAgain()

    expect(stencils).toHaveBeenCalledTimes(2)
  })
})

describe('the files changing under the window', () => {
  it('carry the schedule of a renamed file to where it now remains', async () => {
    const { sync, store } = open()
    store.open('a', 'Animals.md')
    await sync.scheduled.refreshDeckPreset('Animals.md')

    sync.applyPathChanges([], [{ from: 'Animals.md', to: 'Beasts.md' }])
    store.open('b', 'Beasts.md')

    expect(sync.scheduled.getDeckPreset('b').path).toBe('Slow.md')
    expect(sync.scheduled.getDeckPreset('a').path).toBe('')
  })

  it('carry the schedule even where the window holds no deck', async () => {
    const { sync, store } = open()
    await sync.scheduled.refreshDeckPreset('Animals.md')

    sync.applyPathChanges([], [{ from: 'Animals.md', to: 'Beasts.md' }])
    store.open('b', 'Beasts.md')

    expect(sync.scheduled.getDeckPreset('b').path).toBe('Slow.md')
  })

  it('ask for nothing while the window holds no deck', () => {
    const { sync, stencils } = open()

    sync.applyPathChanges(['Animals.md'])

    expect(stencils).not.toHaveBeenCalled()
  })

  it('ask for the stencils once a deck is open', () => {
    const { sync, stencils, store } = open()
    store.open('a', 'Animals.md')

    sync.applyPathChanges(['Animals.md'])

    expect(stencils).toHaveBeenCalledTimes(1)
  })

  it('read the preset of each deck that is open', async () => {
    const { sync, store } = open()
    store.open('a', 'Animals.md')

    sync.applyPathChanges(['Animals.md'])
    await vi.waitFor(() => expect(sync.scheduled.getDeckPreset('a').name).toBe('Slow'))
  })
})
