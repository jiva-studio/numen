import type { Tally as HeatmapTally } from '../../../../ui/heatmap'
import type { VaultCardsDue } from '../../../entities/vault'
import type { Preset } from '../types'

export interface DecksProps {
  vault: VaultCardsDue
  /** How many cards were answered on each day, by the day it was. */
  days: ReadonlyMap<string, HeatmapTally>
  /** How many cards fall on each day still to come, by the day they fall on. */
  due: ReadonlyMap<string, number>
  /** The presets this vault's decks are scheduled by. */
  presets: readonly Preset[]
  /** The preset each deck is scheduled by, by the path the deck is filed under. */
  byDeck: ReadonlyMap<string, Preset>
  /** Whether those presets have been read, which a deck missing from them needs. */
  hasPresets: boolean
  /** The day this is being read on, as the year, the month and the day. */
  today: string
}
