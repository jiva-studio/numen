/**
 * The lists the appearance commands offer: the themes on their two origins, the
 * three modes, and the sizes the window can be drawn at.
 *
 * Each list is derived from the current state on every read, so each row
 * describes only itself.
 */
import { percent } from '@numen/ui'
import type { StepGroup, StepRow } from '@/shared/steps'
import { MODES, ladder, parseSize, isInBounds } from '@/entities/settings'
import type { Mode, Ranges, Sizes, Theme } from '@/entities/settings'
import type { AppearanceWords } from '../words'
import { INTERFACE_SCALE, TEXT_SCALE, getModeId, getSizeId, its } from './appearanceValues'
import type { ScaleKind } from './appearanceValues'

/**
 * The rows with one row per title. A size already in the list is not repeated,
 * and the first of a duplicate pair is kept.
 */
const once = (rows: readonly StepRow[]): readonly StepRow[] => {
  const seenTitles = new Set<string>()
  const only: StepRow[] = []
  for (const one of rows) {
    if (seenTitles.has(one.title)) continue
    seenTitles.add(one.title)
    only.push(one)
  }
  return only
}

/**
 * The rows of the themes given, the one the settings name first.
 */
const getThemeRows = (
  themes: readonly Theme[],
  context: { themeName: string; words: AppearanceWords },
): readonly StepRow[] => {
  const { themeName, words } = context
  const getThemeRow = (one: Theme): StepRow => ({
    id: one.name,
    title: one.title,
    ...(one.name === themeName ? { detail: words.current, isCurrent: true } : {}),
  })
  return [
    ...themes.filter((one) => one.name === themeName).map(getThemeRow),
    ...themes.filter((one) => one.name !== themeName).map(getThemeRow),
  ]
}

/** Resolves the name a plugin is listed under from its id. */
export type PluginNamer = (pluginId: string) => string

/**
 * The themes in their groups: the ones that ship, one group for each plugin
 * that contributes themes, titled with the plugin's name, and the person's own.
 * The group containing the active theme is first, so the list opens on the
 * active theme.
 */
export const getThemeGroups = (
  themes: readonly Theme[],
  themeName: string,
  words: AppearanceWords,
  getPluginName: PluginNamer = (pluginId) => pluginId,
): readonly StepGroup[] => {
  const context = { themeName, words }
  const shipping: StepGroup = {
    id: 'shipping',
    title: words.shipping,
    items: getThemeRows(
      themes.filter((one) => one.isBuiltIn),
      context,
    ),
  }
  const pluginIds = [
    ...new Set(themes.filter((one) => one.pluginId !== '').map((one) => one.pluginId)),
  ]
  const plugins: StepGroup[] = pluginIds.map((pluginId) => ({
    id: `plugin:${pluginId}`,
    title: getPluginName(pluginId),
    items: getThemeRows(
      themes.filter((one) => one.pluginId === pluginId),
      context,
    ),
  }))
  const own: StepGroup = {
    id: 'owned',
    title: words.owned,
    items: getThemeRows(
      themes.filter((one) => !one.isBuiltIn && one.pluginId === ''),
      context,
    ),
    silence: words.noneOwned,
  }
  const groups = [shipping, ...plugins, own]
  const active = groups.find((group) => group.items.some((item) => item.id === themeName))
  return active ? [active, ...groups.filter((group) => group !== active)] : groups
}

/** The detail text of a mode: why it cannot be chosen, or that it is the current one. */
const getModeDetail = (
  one: Mode,
  options: { mode: Mode; isPinned: boolean },
  words: AppearanceWords,
): string => {
  if (options.isPinned) return words.pinned
  return one === options.mode ? words.current : ''
}

/**
 * The three modes. Each is marked unselectable while the active theme pins
 * light or dark: the row is shown, states why, and keyboard navigation skips it.
 */
export const getModeGroups = (
  mode: Mode,
  isPinned: boolean,
  words: AppearanceWords,
): readonly StepGroup[] => {
  const row = (one: Mode): StepRow => {
    const detail = getModeDetail(one, { mode, isPinned }, words)
    return {
      id: getModeId(one),
      title: words[one],
      ...(detail ? { detail } : {}),
      ...(one === mode ? { isCurrent: true } : {}),
      ...(isPinned ? { isDisabled: true } : {}),
    }
  }
  return [{ id: 'half', title: words.half, items: MODES.map(row) }]
}

export interface SizeGroupContext {
  sizes: Sizes
  bounds: Ranges
  words: AppearanceWords
}

/**
 * The sizes one of the two commands offers, in one group: every step within
 * the range, the current size, and the number the user typed. Each appears
 * once, in order, and a size must lie inside the range to be listed.
 *
 * Only the current size row carries detail text, marking the starting position.
 */
export const getSizeGroups = (
  command: string,
  text: string,
  context: SizeGroupContext,
): readonly StepGroup[] => {
  const { sizes, bounds, words } = context
  const which: ScaleKind = command === TEXT_SCALE ? TEXT_SCALE : INTERFACE_SCALE
  const range = its(bounds, which)
  const now = its(sizes, which)
  const parsedSize = parseSize(text)
  const row = (size: number): StepRow => ({
    id: getSizeId(which, size),
    title: percent(size),
    ...(size === now ? { detail: words.current, isCurrent: true } : {}),
  })
  const scaleValues = [...ladder(range), now, parsedSize ?? 0]
    .filter((size) => isInBounds(range, size))
    .sort((first, second) => first - second)
  const title = which === INTERFACE_SCALE ? words.drawing : words.setting
  return [{ id: which, title, items: once(scaleValues.map(row)) }]
}
