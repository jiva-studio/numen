/**
 * The preset that schedules each deck of a vault, and the day's totals under
 * each preset.
 *
 * It is loaded when a vault is opened and again when that vault changes, and
 * the decks of one preset are counted together.
 */
import { computed, ref, shallowRef } from 'vue'
import type { StopReason } from '../../../../api/types.js'

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

  /** The vault the loaded presets belong to. */
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

  // Today is the review day, which the application computes and this window
  // receives: it begins at the hour set in the settings.
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
  /** Problems found in the file, each listed once however many decks name it. */
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
    const { settings: preset } = result.value
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
const createTally = (preset: DeckPreset['settings']): PresetTally => ({
  path: preset.path,
  name: preset.name || 'Defaults',
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
    ...getDayPending(one, day),
    ...getDayReviewed(day),
    budget: { new: budget.new, reviews: budget.reviews, minutes: budget.minutes },
    paused: getStoppedWords(one.stopsOn, one.settings, today),
    wrong: [...one.problems].join('; '),
  }
}

/** What is waiting under a preset today, and what is due in its decks where the count answered nothing. */
const getDayPending = (one: PresetTally, day: PresetCardsDue | undefined) => ({
  named: day?.decks ?? one.decks.length,
  reviewCards: day?.cards ?? 0,
  // The number of cards in a session is the count's own figure; a build whose
  // count returned none uses the sum of cards due across its decks.
  cards: day?.pending ?? one.due + one.fresh,
  closes: day?.closes ?? CLOSES_NOTHING,
})

/** What has been answered under a preset since the day opened, and nothing where the count answered nothing. */
const getDayReviewed = (day: PresetCardsDue | undefined) => ({
  reviewed: day?.reviewed ?? 0,
  reviewedNew: day?.reviewedNew ?? 0,
  reviewedReviews: day?.reviewedReviews ?? 0,
  took: day?.took ?? 0,
})

/** A preset no deck answered for, standing on the count alone. */
const getPresetFromCount = (one: PresetCardsDue, why: string, today: string): Preset => ({
  path: one.preset,
  name: one.title || getDeckName(one.preset),
  settings: null,
  decks: [],
  named: one.decks,
  reviewCards: one.cards,
  // The count answered for this preset with its own figures, and the day is
  // drawn from them.
  cards: one.pending,
  budget: { new: one.new, reviews: one.reviews, minutes: one.minutes },
  closes: one.closes,
  reviewed: one.reviewed,
  reviewedNew: one.reviewedNew,
  reviewedReviews: one.reviewedReviews,
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
