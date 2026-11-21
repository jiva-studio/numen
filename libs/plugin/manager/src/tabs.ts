/** The tabs of a plugin page and the moves between them by the keyboard. */
import { PLUGIN_WORDS as words } from './words'

/** A tab of a plugin page. */
export type PluginTabId = keyof typeof words.pluginTabs

/** One tab with its label. */
export interface PluginTab {
  readonly id: PluginTabId
  readonly text: string
}

/** The id of the element of a tab of the page whose ids start with `base`. */
export const getTabElementId = (base: string, id: PluginTabId): string => `${base}-tab-${id}`

/** The id of the panel a tab controls. */
export const getPanelElementId = (base: string, id: PluginTabId): string => `${base}-panel-${id}`

const STEPS: Readonly<Record<string, number>> = { ArrowRight: 1, ArrowLeft: -1 }

const isTabId = (id: string): id is PluginTabId => id in words.pluginTabs

/** What decides which tabs a plugin page has. */
export interface PluginTabsFacts {
  readonly hasSettings: boolean
  readonly hasSlots?: boolean
  readonly hasData: boolean
}

/** The tabs of a plugin page in their order; Settings only with settings, Slots only with slot entries, Data only with data or a failed delete. */
export const getPluginTabs = (facts: PluginTabsFacts): readonly PluginTab[] =>
  Object.entries(words.pluginTabs).flatMap(([id, text]) => {
    if (!isTabId(id)) return []
    if (id === 'settings' && !facts.hasSettings) return []
    if (id === 'slots' && !facts.hasSlots) return []
    if (id === 'data' && !facts.hasData) return []
    return [{ id, text }]
  })

/** The tab a plugin page opens on: Settings where it has settings, else About. */
export const getFirstPluginTab = (hasSettings: boolean): PluginTabId =>
  hasSettings ? 'settings' : 'about'

/**
 * The index of the tab a key moves to from the tab at `index` of `count`: the arrow keys step and
 * wrap around, Home and End go to the first and the last. Undefined for any other key.
 */
export const getTabIndexForKey = (
  key: string,
  index: number,
  count: number,
): number | undefined => {
  if (key === 'Home') return 0
  if (key === 'End') return count - 1
  const step = STEPS[key]
  return step === undefined ? undefined : (index + step + count) % count
}
