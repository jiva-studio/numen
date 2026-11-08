import type { Tally as HeatmapTally } from '../../../../ui/heatmap'
import type { VaultCardsDue } from '../../../entities/vault'
import type { Preset } from '../types'

export interface DecksProps {
  vault: VaultCardsDue
  days: ReadonlyMap<string, HeatmapTally>
  due: ReadonlyMap<string, number>
  presets: readonly Preset[]
  byDeck: ReadonlyMap<string, Preset>
  hasPresets: boolean
  today: string
}
