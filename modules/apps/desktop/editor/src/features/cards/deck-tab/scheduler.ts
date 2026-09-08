/**
 * Which preset schedules a deck, and putting one on another.
 *
 * The line at the top of a deck draws it, and choosing on that line is a write
 * to the deck's own file. What the vault last said stands until it answers
 * again, so a deck goes on being scheduled by what it was scheduled by while
 * the installation is unreachable.
 */
import { computed, ref, shallowRef } from 'vue'
import type { PresetChoice, Presets, ReadResult } from '../../preset/core'
import { WORDS as words } from '../words'

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
  readonly saying: string
}

/** A deck naming no preset, which is scheduled by the defaults. */
const BY_DEFAULT: DeckPreset = { path: '', name: words.defaults, saying: '' }

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
export function scheduler(presets: Presets, store: ScheduledStore) {
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
      // The presets the window last heard of stand, and a deck is put on one
      // of them until the vault answers again.
      offeredOk = false
    }
  }

  /** The presets asked for again, where the last listing did not answer. */
  const listsPresetsAgain = (): void => {
    if (!offeredOk) void listsPresets()
  }

  /** The preset a deck names, as the line at the top of it draws it. */
  const scheduledOf = (read: ReadResult): DeckPreset => {
    if (read.preset === null) return BY_DEFAULT
    const saying = read.preset.problems[0] ?? ''
    if (read.preset.path === '') return { ...BY_DEFAULT, saying }
    return {
      path: read.preset.path,
      name: read.preset.title || words.unnamed(read.preset.path),
      saying,
    }
  }

  /** Which preset schedules the deck at a path, asked of the vault. */
  const asks = async (path: string): Promise<void> => {
    let read: ReadResult
    try {
      read = await presets.scheduling(path)
    } catch {
      // The preset the window last heard of stands.
      return
    }
    scheduled.value.set(path, scheduledOf(read))
  }

  /** What one tab was told about the preset it last chose. */
  const says = (id: string, text: string): void => {
    chose.value.set(id, text)
  }

  /**
   * A deck put on a preset. What the deck owes reaches the file first, so the
   * write lands on the deck the window read; the file the write made is then
   * read again, and the tab writes against it from there.
   */
  const schedules = async (id: string, preset: string): Promise<void> => {
    const path = store.where(id)
    await store.settles(id)
    try {
      const answer = await presets.schedules(path, preset, store.at(id))
      if (answer.changed) says(id, words.notScheduledChanged)
      else if (answer.refusal !== null) says(id, words.notScheduled)
      else says(id, '')
    } catch {
      // The vault did not answer, and the deck is on the preset it was on. The
      // tab says so where it says what choosing came to.
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
   * The preset one tab is scheduled by. What the tab was last told about a
   * choice it made stands over what the file says, until a choice lands.
   */
  const scheduledAt = (id: string): DeckPreset => {
    const held = scheduled.value.get(store.where(id)) ?? BY_DEFAULT
    const said = chose.value.get(id) ?? ''
    return said === '' ? held : { ...held, saying: said }
  }

  /** What was known about a file no tab of this window stands at any longer. */
  const forgets = (path: string): void => {
    scheduled.value.delete(path)
  }

  /** The same, carried to where the file was filed instead. */
  const moved = (from: string, to: string): void => {
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
    scheduledAt,
    says,
    forgets,
    moved,
    closes,
  }
}
