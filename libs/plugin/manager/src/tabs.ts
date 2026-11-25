/** The tabs of a plugin page and the moves between them by the keyboard. */
import { PLUGIN_WORDS as words } from './words'

/** A tab the page has of its own. */
type BuiltInPluginTabId = keyof typeof words.pluginTabs

/** A tab of a plugin page: a built-in one or the id of a tab the plugin contributes. */
export type PluginTabId = string

/** A tab a plugin contributes to its page. */
export interface ContributedTab {
  readonly id: string
  readonly title: string
}

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

const isTabId = (id: string): id is BuiltInPluginTabId => id in words.pluginTabs

/** What decides which tabs a plugin page has. */
export interface PluginTabsFacts {
  readonly hasSettings: boolean
  readonly hasSlots?: boolean
  readonly hasData: boolean
  readonly contributed?: readonly ContributedTab[]
}

const isShown = (id: BuiltInPluginTabId, facts: PluginTabsFacts): boolean => {
  if (id === 'settings') return facts.hasSettings
  if (id === 'slots') return facts.hasSlots === true
  if (id === 'data') return facts.hasData
  return true
}

/**
 * The tabs of a plugin page in their order; Settings only with settings, Slots only with slot
 * entries, Data only with data or a failed delete. The tabs the plugin contributes follow Settings,
 * or come first where there is none.
 */
export const getPluginTabs = (facts: PluginTabsFacts): readonly PluginTab[] => {
  const contributed = (facts.contributed ?? []).map((one) => ({ id: one.id, text: one.title }))
  const own = Object.entries(words.pluginTabs).flatMap(([id, text]) =>
    isTabId(id) && isShown(id, facts) ? [{ id, text }] : [],
  )
  const at = own[0]?.id === 'settings' ? 1 : 0
  return [...own.slice(0, at), ...contributed, ...own.slice(at)]
}

/** Whether the tab with this id is one the plugin contributes. */
export const isContributedTab = (contributed: readonly ContributedTab[], id: string): boolean =>
  contributed.some((one) => one.id === id)

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
