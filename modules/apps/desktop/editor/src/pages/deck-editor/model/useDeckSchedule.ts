/**
 * Which preset schedules a deck, and putting one on another.
 */
import { computed, ref, shallowRef } from 'vue'
import type { PresetChoice, Presets, ReadResult } from '@/entities/deck'
import { WORDS as words } from '@/entities/deck'

/** The preset a deck is scheduled by, as the line at the top of it draws it. */
export interface DeckPreset {
  /** The note the preset stands in. Empty is a deck scheduled by the defaults. */
  readonly path: string
  /** What that preset is called, in the words on the line. */
  readonly name: string
  /**
   * What is wrong with what the deck names — a preset the vault no longer
   * holds, a note that is not a preset — and nothing where nothing is.
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

/** The deck's own file, as the scheduler reaches it. */
export interface ScheduledStore {
  where(id: string): string
  at(id: string): string
  settles(id: string): Promise<void>
  changed(paths: readonly string[]): void
}

/** The scheduler one window has, over the decks that window holds. */
export function useDeckSchedule(presets: Presets, store: ScheduledStore) {
  /** The presets of the vault, as they were last listed. */
  const offered = shallowRef<readonly PresetChoice[]>([])
  /** Whether the last listing of the presets answered. */
  let offeredOk = true
  /** Which preset schedules each file, under the path it is filed at. */
  const scheduled = ref(new Map<string, DeckPreset>())
  /** What choosing a preset came to, under the tab that chose. */
  const chose = ref(new Map<string, string>())

  /** The presets of the vault, asked for again. */
  const listsPresets = async (): Promise<void> => {
    try {
      offered.value = await presets.list()
      offeredOk = true
    } catch {
      // Preset listing failed.
      offeredOk = false
    }
  }

  /** The presets asked for again, where the last listing did not answer. */
  const listsPresetsAgain = (): void => {
    if (!offeredOk) void listsPresets()
  }

  /** The preset a deck names, as the line at the top of it draws it. */
  const readDeckPreset = (read: ReadResult): DeckPreset => {
    if (read.preset === null) return BY_DEFAULT
    const errorMessage = read.preset.problems[0] ?? ''
    if (read.preset.path === '') return { ...BY_DEFAULT, errorMessage }
    return {
      path: read.preset.path,
      name: read.preset.title || words.unnamed(read.preset.path),
      errorMessage,
    }
  }

  /** Which preset schedules the deck at a path, asked of the vault. */
  const asks = async (path: string): Promise<void> => {
    let read: ReadResult
    try {
      read = await presets.scheduling(path)
    } catch {
      // Reading preset scheduling failed.
      return
    }
    scheduled.value.set(path, readDeckPreset(read))
  }

  /** What one tab was told about the preset it last chose. */
  const says = (id: string, text: string): void => {
    chose.value.set(id, text)
  }

  /**
   * A deck put on a preset.
   */
  const schedules = async (id: string, preset: string): Promise<void> => {
    const path = store.where(id)
    await store.settles(id)
    try {
      const answer = await presets.schedules(path, preset, store.at(id))
      if (answer.changed) says(id, words.notScheduledChanged)
      else if (answer.error !== null) says(id, words.notScheduled)
      else says(id, '')
    } catch {
      // Scheduling preset failed.
      says(id, words.unreachable)
      return
    }
    store.changed([path])
    await asks(path)
  }

  /** The presets a deck may be put on: the defaults, and every preset the vault holds. */
  const choices = computed<readonly Choice[]>(() => [
    { path: '', name: words.defaults },
    ...offered.value.map((one) => ({
      path: one.path,
      name: one.title || words.unnamed(one.path),
    })),
  ])

  /**
   * The preset one tab is scheduled by.
   */
  const getDeckPreset = (id: string): DeckPreset => {
    const held = scheduled.value.get(store.where(id)) ?? BY_DEFAULT
    const said = chose.value.get(id) ?? ''
    return said === '' ? held : { ...held, errorMessage: said }
  }

  /** What was known about a file no tab of this window stands at any longer. */
  const forgets = (path: string): void => {
    scheduled.value.delete(path)
  }

  /** The same, carried to where the file was filed instead. */
  const moveFile = (from: string, to: string): void => {
    const by = scheduled.value.get(from)
    if (by) scheduled.value.set(to, by)
    scheduled.value.delete(from)
  }

  /** What a tab that has closed was told about the choice it made. */
  const closes = (id: string): void => {
    chose.value.delete(id)
  }

  return {
    choices,
    listsPresets,
    listsPresetsAgain,
    asks,
    schedules,
    getDeckPreset,
    says,
    forgets,
    moveFile,
    closes,
  }
}
