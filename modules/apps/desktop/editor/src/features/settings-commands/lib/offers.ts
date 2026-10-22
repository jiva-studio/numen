/**
 * The lists the appearance commands offer: the themes on their two origins, the
 * three modes, and the sizes the window can be drawn at.
 *
 * Each list is the state currently, read again, so a row says only what is
 * true of it alone.
 */
import { percent } from '@numen/ui'
import type { StepGroup, StepRow } from '@/features/command-palette/@x/settings-commands'
import { MODES, ladder, parseSize, isInBounds } from '@/entities/settings'
import type { Mode, Ranges, Sizes, Theme } from '@/entities/settings'
import type { AppearanceWords } from '../words'
import { INTERFACE_SCALE, TEXT_SCALE, getModeId, getSizeId, its } from './appearanceValues'
import type { ScaleKind } from './appearanceValues'

/**
 * The rows one to a title. A size the list already holds is not held twice, and
 * the first of a pair is the one that stands.
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
 * The themes, in the two groups they come off. The group the theme worn came
 * off stands first, so opening the list relies on what the window wears.
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

/** What is said about a mode: why it cannot be chosen, or that it is the one. */
const getModeDetail = (
  one: Mode,
  options: { mode: Mode; isPinned: boolean },
  words: AppearanceWords,
): string => {
  if (options.isPinned) return words.pinned
  return one === options.mode ? words.current : ''
}

/**
 * The three modes. Each is drawn as not to be chosen while the theme worn pins
 * light and dark: it is there, it says why, and the keyboard passes over it.
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
      ...(isPinned ? { disabled: true } : {}),
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
 * The sizes one of the two commands offers, in one group of its own: every step
 * the range reaches, the size the window is drawn at, and the number a person
 * typed. Each appears once, in order, and the range is what a size has to be
 * inside to stand at all.
 *
 * The one row that says anything is the size the window is drawn at, which is
 * where a person is standing before they walk.
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
