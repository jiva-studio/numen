/**
 * Which preset schedules each deck of a vault, and what a day under each of
 * them comes to.
 *
 * It is asked for when a vault is opened and again whenever that vault moves,
 * and the decks of one preset are counted together however many there are.
 */
import { computed, ref, shallowRef } from 'vue'
import type { StopReason } from '@numen/protocol'

import { deckName } from '@/entities/vault'
import type { VaultCardsDue } from '@/entities/vault'

import { readDeckPreset, UNREAD } from '../api/presets'
import type { DeckPresetResult, PresetsClient } from '../api/presets'
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
  const known = ref(false)

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
    known.value = false
  }

  // Today is the review day, which the application measures and this window is
  // told: it begins at the hour the settings name.
  const read = async (due: VaultCardsDue | null, today: string) => {
    if (!due) {
      forget()
      return
    }
    // A vault read again keeps what is known of it while the reading runs, so
    // the screen it is read behind does not empty and fill.
    if (of.value !== due.vault) known.value = false
    of.value = due.vault

    const held = await Promise.all(
      due.decks.map((deck) => readDeckPreset(deps.presets, due.vault, deck.deck)),
    )
    if (of.value !== due.vault) return

    presets.value = gather(due, held, today)
    known.value = true
  }

  return { presets, byDeck, of, known, read, forget }
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
const budgetOf = (settings: Settings): Budget => ({
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
const gather = (vault: VaultCardsDue, answered: readonly DeckPresetResult[], today: string): Preset[] => {
  const owed = new Map(vault.decks.map((one) => [one.deck, one]))
  const came = new Map(vault.presets.map((one) => [one.preset, one]))
  const at = new Map<string, PresetTally>()

  for (const one of answered) {
    if (!one.held) continue
    let into = at.get(one.held.path)
    if (!into) {
      into = {
        path: one.held.path,
        name: one.held.name || 'The defaults',
        settings: one.held.settings,
        stopsOn: one.held.stopsOn,
        decks: [],
        due: 0,
        fresh: 0,
        problems: new Set(),
      }
      at.set(one.held.path, into)
    }
    into.decks.push(one.deck)
    for (const problem of one.held.problems) into.problems.add(problem)
    const deck = owed.get(one.deck)
    into.due += deck?.due ?? 0
    into.fresh += deck?.new ?? 0
  }

  const out = [...at.values()].map((one): Preset => {
    const day = came.get(one.path)
    const budget = day ?? budgetOf(one.settings)
    return {
      path: one.path,
      name: one.name,
      settings: one.settings,
      decks: one.decks,
      named: day?.decks ?? one.decks.length,
      faces: day?.cards ?? 0,
      // What a session over it asks is the count's own figure, and a build that
      // answered none falls back to what its decks owe between them.
      cards: day?.owed ?? one.due + one.fresh,
      budget: { new: budget.new, reviews: budget.reviews, minutes: budget.minutes },
      closes: day?.closes ?? CLOSES_NOTHING,
      answered: day?.answered ?? 0,
      answeredNew: day?.answeredNew ?? 0,
      answeredReviews: day?.answeredReviews ?? 0,
      took: day?.took ?? 0,
      paused: getStoppedWords(one.stopsOn, one.settings, today),
      wrong: [...one.problems].join('; '),
    }
  })

  // A preset every one of whose decks is empty is answered for no deck, and so
  // is one whose settings could not be read, so both stand here on the count
  // alone, beside the presets nothing points at.
  const why = getRefusedReason(answered)
  for (const one of vault.presets) {
    if (at.has(one.preset)) continue
    out.push({
      path: one.preset,
      name: one.title || deckName(one.preset),
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
  }
  return out
}

/**
 * Why the presets no deck answered for could not be read. Every deck of one
 * preset is refused for the same reason, so a reason every refused deck gave is
 * the reason of each preset none of them could read.
 */
const getRefusedReason = (answered: readonly DeckPresetResult[]): string => {
  const why = new Set(answered.filter((one) => one.refused).map((one) => one.refused))
  if (why.size === 0) return ''
  return why.size === 1 ? ([...why][0] ?? '') : UNREAD
}
