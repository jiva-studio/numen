/**
 * Which preset schedules each deck of a vault, and what a day under each of
 * them evaluates to.
 *
 * It is asked for when a vault is opened and again whenever that vault moves,
 * and the decks of one preset are counted together however many there are.
 */
import { computed, ref, shallowRef } from 'vue'
import type { StopReason } from '@numen/protocol'

import { getDeckName } from '../../../entities/vault'
import type { DeckCardsDue, PresetCardsDue, VaultCardsDue } from '../../../entities/vault'

import { readDeckPreset, PRESET_UNREADABLE_MESSAGE } from '../api/presets'
import type { DeckPreset, DeckPresetResult, PresetsClient } from '../api/presets'
import { CLOSES_NOTHING } from '../types'
import type { Budget, Preset, Settings } from '../types'
import { getStoppedWords } from '../words'

export interface VaultPresetsDeps {
  presets: PresetsClient
}

export function useVaultPresets(deps: VaultPresetsDeps) {
  const presets = shallowRef<readonly Preset[]>([])

  /** Which vault the presets on hand belong to. */
  const of = ref('')

  /**
   * Whether the presets of that vault have been read. A deck the reading passed
   * over is a deck whose preset could not be read, and until it has run nothing
   * is known either way.
   */
  const recognized = ref(false)

  /** The preset each deck is scheduled by, by the path the deck is filed under. */
  const byDeck = computed(() => {
    const out = new Map<string, Preset>()
    for (const one of presets.value) {
      for (const deck of one.decks) out.set(deck, one)
    }
    return out
  })

  const forget = () => {
    presets.value = []
    of.value = ''
    recognized.value = false
  }

  // Today is the review day, which the application measures and this window is
  // told: it begins at the hour the settings name.
  const readItem = async (due: VaultCardsDue | null, today: string) => {
    if (!due) {
      forget()
      return
    }
    // A vault read again keeps what is known of it while the reading runs, so
    // the screen it is read behind does not empty and fill.
    if (of.value !== due.vault) recognized.value = false
    of.value = due.vault

    const target = await Promise.all(
      due.decks.map(async (deck) => ({
        deck,
        result: await readDeckPreset(deps.presets, due.vault, deck.deck),
      })),
    )
    if (of.value !== due.vault) return

    presets.value = groupDecksIntoPresets(due, target, today)
    recognized.value = true
  }

  return { presets, byDeck, of, known: recognized, read: readItem, forget }
}

/** One deck of a vault, with the preset it is scheduled by or why that could not be read. */
interface DeckRead {
  deck: DeckCardsDue
  result: DeckPresetResult
}

/** One preset while its decks are still being counted into it. */
interface PresetTally {
  path: string
  name: string
  settings: Settings
  stopsOn: StopReason
  decks: string[]
  due: number
  fresh: number
  /** What was wrong in the file, each said once however many decks name it. */
  problems: Set<string>
}

/** What the day holds under a preset the application counted nothing for. */
const createBudget = (settings: Settings): Budget => ({
  new: settings.newADay,
  reviews: settings.reviewsADay,
  minutes: settings.minutesADay,
})

/**
 * The presets of a vault, each carrying the decks that name it.
 *
 * Every preset the vault holds gets a row. The ones whose decks hold cards are
 * gathered from what each deck answered, and the rest stand on the count alone.
 */
const groupDecksIntoPresets = (
  vault: VaultCardsDue,
  reads: readonly DeckRead[],
  today: string,
): Preset[] => {
  const came = new Map(vault.presets.map((one) => [one.preset, one]))
  const tallies = countDecksIntoPresets(reads)

  const out = [...tallies.values()].map((one) => getPresetFromDecks(one, came.get(one.path), today))

  // A preset every one of whose decks is empty is answered for no deck, and so
  // is one whose settings could not be read, so both stand here on the count
  // alone, beside the presets nothing points at.
  const why = getCommonError(reads.map((one) => one.result))
  for (const one of vault.presets) {
    if (tallies.has(one.preset)) continue
    out.push(getPresetFromCount(one, why, today))
  }
  return out
}

/** The decks each preset schedules, counted into it, by the path it is filed under. */
const countDecksIntoPresets = (reads: readonly DeckRead[]): Map<string, PresetTally> => {
  const tallies = new Map<string, PresetTally>()
  for (const { deck, result } of reads) {
    if (!result.ok) continue
    const { held: preset } = result.value
    let into = tallies.get(preset.path)
    if (!into) {
      into = createTally(preset)
      tallies.set(preset.path, into)
    }
    into.decks.push(deck.deck)
    for (const problem of preset.problems) into.problems.add(problem)
    into.due += deck.due
    into.fresh += deck.new
  }
  return tallies
}

/** One preset currently before a deck has been counted into it. */
const createTally = (preset: DeckPreset['held']): PresetTally => ({
  path: preset.path,
  name: preset.name || 'The defaults',
  settings: preset.settings,
  stopsOn: preset.stopsOn,
  decks: [],
  due: 0,
  fresh: 0,
  problems: new Set(),
})

/** A preset the decks answered for, standing on the day the count gave it. */
const getPresetFromDecks = (
  one: PresetTally,
  day: PresetCardsDue | undefined,
  today: string,
): Preset => {
  const budget = day ?? createBudget(one.settings)
  return {
    path: one.path,
    name: one.name,
    settings: one.settings,
    decks: one.decks,
    ...getDayOwed(one, day),
    ...getDayAnswered(day),
    budget: { new: budget.new, reviews: budget.reviews, minutes: budget.minutes },
    paused: getStoppedWords(one.stopsOn, one.settings, today),
    wrong: [...one.problems].join('; '),
  }
}

/** What is waiting under a preset today, and what its decks owe where the count answered nothing. */
const getDayOwed = (one: PresetTally, day: PresetCardsDue | undefined) => ({
  named: day?.decks ?? one.decks.length,
  faces: day?.cards ?? 0,
  // What a session over it asks is the count's own figure, and a build that
  // answered none falls back to what its decks owe between them.
  cards: day?.owed ?? one.due + one.fresh,
  closes: day?.closes ?? CLOSES_NOTHING,
})

/** What has been answered under a preset since the day opened, and nothing where the count answered nothing. */
const getDayAnswered = (day: PresetCardsDue | undefined) => ({
  answered: day?.answered ?? 0,
  answeredNew: day?.answeredNew ?? 0,
  answeredReviews: day?.answeredReviews ?? 0,
  took: day?.took ?? 0,
})

/** A preset no deck answered for, standing on the count alone. */
const getPresetFromCount = (one: PresetCardsDue, why: string, today: string): Preset => ({
  path: one.preset,
  name: one.title || getDeckName(one.preset),
  settings: null,
  decks: [],
  named: one.decks,
  faces: one.cards,
  // The count answered for this preset with its own figures, and the day is
  // drawn from them.
  cards: one.owed,
  budget: { new: one.new, reviews: one.reviews, minutes: one.minutes },
  closes: one.closes,
  answered: one.answered,
  answeredNew: one.answeredNew,
  answeredReviews: one.answeredReviews,
  took: one.took,
  // The count answered for this preset, so its verdict is the count's.
  paused: getStoppedWords(one.stopsOn, null, today),
  wrong: one.decks > 0 ? why : '',
})

/**
 * What stopped the presets no deck answered for. Every deck of one preset fails
 * alike, so an error all of them gave is the error of each preset none could
 * read; where they gave several, none of them is that preset's.
 */
const getCommonError = (results: readonly DeckPresetResult[]): string => {
  const errors = new Set<string>()
  for (const one of results) if (!one.ok) errors.add(one.error)
  const [first, second] = errors
  if (first === undefined) return ''
  return second === undefined ? first : PRESET_UNREADABLE_MESSAGE
}
