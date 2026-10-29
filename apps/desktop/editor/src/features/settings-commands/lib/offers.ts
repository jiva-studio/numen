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
 * The themes of one origin, the one the settings name first. Where a theme comes
 */
/**
 * The themes of one origin, the one the settings name first. Where a theme comes
 */
const getThemesOfOrigin = (
  themes: readonly Theme[],
  isBuiltIn: boolean,
  context: { themeName: string; words: AppearanceWords },
): readonly StepRow[] => {
  const { themeName, words } = context
  const off = themes.filter((one) => one.isBuiltIn === isBuiltIn)
  const getThemeRow = (one: Theme): StepRow => ({
    id: one.name,
    title: one.title,
    ...(one.name === themeName ? { detail: words.current, isCurrent: true } : {}),
  })
  return [
    ...off.filter((one) => one.name === themeName).map(getThemeRow),
    ...off.filter((one) => one.name !== themeName).map(getThemeRow),
  ]
}

/**
 * The themes in their two origin groups. The group containing the active theme
 * is first, so the list opens on the active theme.
 */
export const getThemeGroups = (
  themes: readonly Theme[],
  themeName: string,
  words: AppearanceWords,
): readonly StepGroup[] => {
  const shipping = {
    id: 'shipping',
    title: words.shipping,
    items: getThemesOfOrigin(themes, true, { themeName, words }),
  }
  const own = {
    id: 'owned',
    title: words.owned,
    items: getThemesOfOrigin(themes, false, { themeName, words }),
    silence: words.noneOwned,
  }
  const aged = themes.find((one) => one.name === themeName)
  return aged && !aged.isBuiltIn ? [own, shipping] : [shipping, own]
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
