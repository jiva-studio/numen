/**
 * Which preset schedules a deck, and putting one on another.
 */
import { computed, ref, shallowRef } from 'vue'
import { type PresetChoice, type Presets, type PresetReadResult, WORDS as words } from '../../cards'

/** The preset a deck is scheduled by, as the line at the top of it draws it. */
export interface DeckPreset {
  readonly path: string
  /** What that preset is called, in the words on the line. */
  readonly name: string
  /**
   * What is wrong with what the deck names — a preset missing from the vault, a
   * note that is not a preset — and empty where nothing is.
   */
  readonly errorMessage: string
}

/** A deck naming no preset, which is scheduled by the defaults. */
const BY_DEFAULT: DeckPreset = { path: '', name: words.defaults, errorMessage: '' }

/** One preset a deck may be put on, as the line offers it. */
export interface Choice {
  readonly path: string
  readonly name: string
}

/** The deck's own file, as the scheduler accesses it. */
export interface ScheduledStore {
  getPath(id: string): string
  getFilePath(id: string): string
  flushPending(id: string): Promise<void>
  applyPathChanges(paths: readonly string[]): void
}

/** The preset a deck names, as the line at the top of it draws it. */
const readDeckPreset = (answer: PresetReadResult): DeckPreset => {
  const readItem = answer.ok ? answer.value.preset : null
  if (readItem === null) return BY_DEFAULT
  const errorMessage = readItem.problems[0] ?? ''
  if (readItem.path === '') return { ...BY_DEFAULT, errorMessage }
  return { path: readItem.path, name: readItem.title || words.unnamed(readItem.path), errorMessage }
}

export function usePresetListing(presets: Presets) {
  const listedPresets = shallowRef<readonly PresetChoice[]>([])
  let presetsWereListed = true

  const listPresets = async (): Promise<void> => {
    try {
      listedPresets.value = await presets.list()
      presetsWereListed = true
    } catch {
      presetsWereListed = false
    }
  }

  const listPresetsAgain = (): void => {
    if (!presetsWereListed) void listPresets()
  }

  const choices = computed<readonly Choice[]>(() => [
    { path: '', name: words.defaults },
    ...listedPresets.value.map((one) => ({
      path: one.path,
      name: one.title || words.unnamed(one.path),
    })),
  ])

  return { choices, listPresets, listPresetsAgain }
}

/** The scheduler one window has, over the decks that window holds. */
export function useDeckSchedule(presets: Presets, store: ScheduledStore) {
  const listing = usePresetListing(presets)
  const scheduled = ref(new Map<string, DeckPreset>())
  const chose = ref(new Map<string, string>())

  const refreshDeckPreset = async (path: string): Promise<void> => {
    let readItem: PresetReadResult
    try {
      readItem = await presets.getDeckPreset(path)
    } catch {
      return
    }
    scheduled.value.set(path, readDeckPreset(readItem))
  }

  const setChoiceMessage = (id: string, text: string): void => {
    chose.value.set(id, text)
  }

  const scheduleDeck = async (id: string, preset: string): Promise<void> => {
    const path = store.getPath(id)
    await store.flushPending(id)
    try {
      const answer = await presets.scheduleDeck(path, preset, store.getFilePath(id))
      if (answer.ok) setChoiceMessage(id, '')
      else if (answer.error === 'changed') setChoiceMessage(id, words.notScheduledChanged)
      else setChoiceMessage(id, words.notScheduled)
    } catch {
      setChoiceMessage(id, words.unreachable)
      return
    }
    store.applyPathChanges([path])
    await refreshDeckPreset(path)
  }

  /**
   * The preset one tab is scheduled by.
   */
  const getDeckPreset = (id: string): DeckPreset => {
    const target = scheduled.value.get(store.getPath(id)) ?? BY_DEFAULT
    const error = chose.value.get(id) ?? ''
    return error === '' ? target : { ...target, errorMessage: error }
  }

  const forgetFile = (path: string): void => {
    scheduled.value.delete(path)
  }

  /** The same, carried to where the file was filed instead. */
  const moveFile = (from: string, to: string): void => {
    const preset = scheduled.value.get(from)
    if (preset) scheduled.value.set(to, preset)
    scheduled.value.delete(from)
  }

  /** Forgets the choice a closed tab made. */
  const forgetTab = (id: string): void => {
    chose.value.delete(id)
  }

  return {
    listing,
    refreshDeckPreset,
    scheduleDeck,
    getDeckPreset,
    setChoiceMessage,
    forgetFile,
    moveFile,
    forgetTab,
  }
}
