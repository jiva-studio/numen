import type { Tally as HeatmapTally } from '../../lib/heatmap/heatmap'
import type { VaultCardsDue } from '../../lib/vault/types'
import type { Preset } from '../../lib/decks/types'

export interface DecksProps {
  vault: VaultCardsDue
  days: ReadonlyMap<string, HeatmapTally>
  due: ReadonlyMap<string, number>
  presets: readonly Preset[]
  byDeck: ReadonlyMap<string, Preset>
  hasPresets: boolean
  today: string
}
