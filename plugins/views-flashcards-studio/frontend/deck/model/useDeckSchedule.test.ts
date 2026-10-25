/**
 * Which preset schedules a deck, the presets it may be put on, and what a
 * choice came to.
 */
import { describe, expect, it, vi } from 'vitest'
import { StopReason } from '@numen/protocol'
import { asFailure, asValue } from '@numen/wire'
import {
  DEFAULTS,
  NO_BOUNDS,
  WORDS,
  type PresetChoice,
  type PresetReadResult,
  type PresetWriteResult,
  type Presets,
} from '../../cards'
import { useDeckSchedule, type ScheduledStore } from './useDeckSchedule'

const SCHEDULING = { stops: StopReason.NOTHING, stopsOn: StopReason.NOTHING }

const named = (path: string, title: string, problems: string[] = []): PresetReadResult =>
  asValue({
    preset: { path, title, settings: DEFAULTS, problems, ...SCHEDULING },
    at: '',
    bounds: NO_BOUNDS,
  })

/** Presets answering what a test sets, and counting every listing. */
const fakePresets = () => {
  const state = {
    listings: 0,
    isListingDown: false,
    list: [{ path: 'Slow.md', title: 'Slow' }] as readonly PresetChoice[],
    read: named('Slow.md', 'Slow') as PresetReadResult,
    isReadDown: false,
    write: asValue({ at: '' }) as PresetWriteResult,
    isWriteDown: false,
  }
  const scheduleDeck = vi.fn<Presets['scheduleDeck']>(async () => {
    if (state.isWriteDown) throw new Error('down')
    return state.write
  })
  const presets: Presets = {
    read: async () => state.read,
    list: async () => {
      state.listings += 1
      if (state.isListingDown) throw new Error('down')
      return state.list
    },
    createPreset: async () => asValue({ path: '' }),
    getDeckPreset: async () => {
      if (state.isReadDown) throw new Error('down')
      return state.read
    },
    scheduleDeck,
    write: async () => asValue({ at: '' }),
    curve: async () => {
      throw new Error('unused')
    },
  }
  return { presets, state, scheduleDeck }
}

const fakeStore = () => {
  const paths = new Map<string, string>([
    ['a', 'Animals.md'],
    ['b', 'Birds.md'],
  ])
  const applyPathChanges = vi.fn<ScheduledStore['applyPathChanges']>()
  const store: ScheduledStore = {
    getPath: (id) => paths.get(id) ?? '',
    getFilePath: (id) => `/vault/${paths.get(id) ?? ''}`,
    flushPending: async () => {},
    applyPathChanges,
  }
  return { store, paths, applyPathChanges }
}

const open = () => {
  const fake = fakePresets()
  const target = fakeStore()
  return { ...fake, ...target, schedule: useDeckSchedule(fake.presets, target.store) }
}

describe('the presets listed again', () => {
  it('are not requested for before any listing failed', () => {
    const { schedule, state } = open()

    schedule.listing.listPresetsAgain()

    expect(state.listings).toBe(0)
  })

  it('are requested for once a listing failed', async () => {
    const { schedule, state } = open()
    state.isListingDown = true
    await schedule.listing.listPresets()

    schedule.listing.listPresetsAgain()

    expect(state.listings).toBe(2)
  })

  it('are not requested for once a listing answered after the failure', async () => {
    const { schedule, state } = open()
    state.isListingDown = true
    await schedule.listing.listPresets()
    state.isListingDown = false
    await schedule.listing.listPresets()

    schedule.listing.listPresetsAgain()

    expect(state.listings).toBe(2)
  })
})

describe('the preset a deck is scheduled by', () => {
  it('is the defaults where nothing was read', () => {
    const { schedule } = open()

    expect(schedule.getDeckPreset('a')).toStrictEqual({
      path: '',
      name: WORDS.defaults,
      errorMessage: '',
    })
  })

  it('is the defaults where the vault rejected the read', async () => {
    const { schedule, state } = open()
    state.read = asFailure('missing')

    await schedule.refreshDeckPreset('Animals.md')

    expect(schedule.getDeckPreset('a').path).toBe('')
  })

  it('is the defaults where the read reached nothing', async () => {
    const { schedule, state } = open()
    state.read = asValue({ preset: null, at: '', bounds: NO_BOUNDS })

    await schedule.refreshDeckPreset('Animals.md')

    expect(schedule.getDeckPreset('a').name).toBe(WORDS.defaults)
  })

  it('remains where the read failed after a preset was read', async () => {
    const { schedule, state } = open()
    await schedule.refreshDeckPreset('Animals.md')
    state.isReadDown = true

    await schedule.refreshDeckPreset('Animals.md')

    expect(schedule.getDeckPreset('a').path).toBe('Slow.md')
  })

  it('is the preset the vault says the deck names, under its title', async () => {
    const { schedule } = open()

    await schedule.refreshDeckPreset('Animals.md')

    expect(schedule.getDeckPreset('a')).toStrictEqual({
      path: 'Slow.md',
      name: 'Slow',
      errorMessage: '',
    })
  })

  it('is named by its file where it carries no title', async () => {
    const { schedule, state } = open()
    state.read = named('presets/Slow.md', '')

    await schedule.refreshDeckPreset('Animals.md')

    expect(schedule.getDeckPreset('a').name).toBe(WORDS.unnamed('presets/Slow.md'))
  })

  it('says the first problem against what the deck names', async () => {
    const { schedule, state } = open()
    state.read = named('Slow.md', 'Slow', ['First', 'Second'])

    await schedule.refreshDeckPreset('Animals.md')

    expect(schedule.getDeckPreset('a').errorMessage).toBe('First')
  })

  it('is the defaults carrying the problem where the deck names no file', async () => {
    const { schedule, state } = open()
    state.read = named('', '', ['Gone'])

    await schedule.refreshDeckPreset('Animals.md')

    expect(schedule.getDeckPreset('a')).toStrictEqual({
      path: '',
      name: WORDS.defaults,
      errorMessage: 'Gone',
    })
  })
})

describe('the presets a deck may be put on', () => {
  it('are the defaults, then each preset the vault lists', async () => {
    const { schedule, state } = open()
    state.list = [
      { path: 'Slow.md', title: 'Slow' },
      { path: 'presets/Fast.md', title: '' },
    ]

    await schedule.listing.listPresets()

    expect(schedule.listing.choices.value).toStrictEqual([
      { path: '', name: WORDS.defaults },
      { path: 'Slow.md', name: 'Slow' },
      { path: 'presets/Fast.md', name: WORDS.unnamed('presets/Fast.md') },
    ])
  })
})

describe('a deck put on a preset', () => {
  it('is handed to the vault with the file the tab read, and the files told', async () => {
    const { schedule, scheduleDeck, applyPathChanges } = open()

    await schedule.scheduleDeck('a', 'Slow.md')

    expect(scheduleDeck).toHaveBeenCalledWith('Animals.md', 'Slow.md', '/vault/Animals.md')
    expect(applyPathChanges).toHaveBeenCalledWith(['Animals.md'])
  })

  it('draws the preset it was put on', async () => {
    const { schedule } = open()

    await schedule.scheduleDeck('a', 'Slow.md')

    expect(schedule.getDeckPreset('a').path).toBe('Slow.md')
  })

  it('says nothing once it was put on, whatever an earlier choice came to', async () => {
    const { schedule, state } = open()
    state.write = asFailure('missing')
    await schedule.scheduleDeck('a', 'Slow.md')
    expect(schedule.getDeckPreset('a').errorMessage).toBe(WORDS.notScheduled)
    state.write = asValue({ at: '' })

    await schedule.scheduleDeck('a', 'Slow.md')

    expect(schedule.getDeckPreset('a').errorMessage).toBe('')
  })

  it('says the file changed where the vault says so', async () => {
    const { schedule, state } = open()
    state.write = asFailure('changed' as const)

    await schedule.scheduleDeck('a', 'Slow.md')

    expect(schedule.getDeckPreset('a').errorMessage).toBe(WORDS.notScheduledChanged)
  })

  it('says the vault could not be reached, and notifies no file', async () => {
    const { schedule, state, applyPathChanges } = open()
    state.isWriteDown = true

    await schedule.scheduleDeck('a', 'Slow.md')

    expect(schedule.getDeckPreset('a').errorMessage).toBe(WORDS.unreachable)
    expect(applyPathChanges).not.toHaveBeenCalled()
  })
})

describe('what is known of a file', () => {
  it('is let go with the file', async () => {
    const { schedule } = open()
    await schedule.refreshDeckPreset('Animals.md')

    schedule.forgetFile('Animals.md')

    expect(schedule.getDeckPreset('a').path).toBe('')
  })

  it('is carried to where the file now remains, and left at no other name', async () => {
    const { schedule, paths } = open()
    await schedule.refreshDeckPreset('Animals.md')

    schedule.moveFile('Animals.md', 'Beasts.md')

    paths.set('c', 'Beasts.md')
    expect(schedule.getDeckPreset('c').path).toBe('Slow.md')
    expect(schedule.getDeckPreset('a').path).toBe('')
  })

  it('remains at the name it went to, where the old name retained nothing', async () => {
    const { schedule } = open()
    await schedule.refreshDeckPreset('Birds.md')

    schedule.moveFile('Missing.md', 'Birds.md')

    expect(schedule.getDeckPreset('b').path).toBe('Slow.md')
  })
})

describe('what a tab was told about its choice', () => {
  it('is returned over the preset, until the tab is forgotten', () => {
    const { schedule } = open()
    schedule.setChoiceMessage('a', 'Told')
    expect(schedule.getDeckPreset('a').errorMessage).toBe('Told')

    schedule.forgetTab('a')

    expect(schedule.getDeckPreset('a').errorMessage).toBe('')
  })
})
