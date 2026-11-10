/** The tab that shows and edits a review schedule. */
import PresetTab from './ui/PresetTab.vue'
import { usePresetTab as usePresetTabOf, type PresetTabDeps } from './model/kind'

export { PRESET } from './model/kind'
export { PresetTab }
export type { PresetTabState } from './model/types'

/** The preset tabs of a window, drawn by `PresetTab`. */
export const usePresetTab = (deps: Omit<PresetTabDeps, 'component'>) =>
  usePresetTabOf({ ...deps, component: PresetTab })
