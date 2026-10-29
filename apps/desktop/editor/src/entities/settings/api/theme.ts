/**
 * Queries the application for the window's appearance and sends it the user's
 * choices: the theme, the mode of a colour pair, and the sizes.
 *
 * A theme is one stylesheet, and nothing here reads inside one: the transferred
 * data is its name, the text of its file, and which theme is selected. A size is
 * one multiplier, and its permitted range is defined by the application.
 */
import {
  getNames,
  THEME_MODE,
  THEME_ORIGIN,
  type ThemeModeCode,
  type ThemeOriginCode,
} from '@numen/plugin-sdk'
import { theme } from '@/shared/clients'

import type { Bounds, Mode, Scales, Theme } from '../lib/appearance'
export type { Bounds, Mode, Scales, Theme }

/** The two multipliers the window is drawn at. One is as designed. */
export type Sizes = Scales<number>

/** How far each of the two goes. */
export type Ranges = Scales<Bounds>

/** Every available theme, and the theme and sizes the settings apply to the window. */
export interface Appearance {
  readonly themes: readonly Theme[]
  readonly applied: string
  readonly mode: Mode
  readonly sizes: Sizes
  readonly bounds: Ranges
}

/** The queries the window makes about its appearance. */
export interface Themes {
  getAppearance(): Promise<Appearance>
  readTheme(name: string): Promise<string>
  writeAppearance(name: string, mode: Mode, sizes: Sizes): Promise<string>
  /**
   * The names of themes changed in the user's folder, until the signal aborts.
   * A theme whose file was deleted is included.
   */
  watchThemes(signal: AbortSignal): AsyncIterable<readonly string[]>
}

/** The implementation of these queries used by the window. */
export const themes: Themes = {
  getAppearance: async () => {
    const answer = await theme.listThemes({})
    return {
      themes: answer.themes.map((one) => ({
        name: one.name,
        title: one.title,
        isBuiltIn: ships[one.origin],
        isPinned: one.isPinned,
      })),
      applied: answer.applied,
      mode: WORDED[answer.mode] ?? 'system',
      sizes: { interfaceScale: answer.interfaceScale, textScale: answer.textScale },
      bounds: {
        interfaceScale: parseBounds(answer.interfaceScaleBounds),
        textScale: parseBounds(answer.textScaleBounds),
      },
    }
  },
  readTheme: async (name) => (await theme.readTheme({ name })).css,
  writeAppearance: async (name, mode, sizes) =>
    (
      await theme.writeAppearance({
        name,
        mode: ASKED[mode],
        interfaceScale: sizes.interfaceScale,
        textScale: sizes.textScale,
      })
    ).error,
  watchThemes: async function* (signal) {
    for await (const update of theme.watchThemes({}, { signal })) yield update.names
  },
}

/** How far a size goes. A bound the application left out is no bound at all. */
const parseBounds = (bounds: { min: number; max: number } | undefined): Bounds => ({
  min: bounds?.min ?? 0,
  max: bounds?.max ?? 0,
})

/**
 * Whether a theme ships inside the application. Keyed by the schema, so an origin
 * added to it has to be answered here before this compiles.
 */
const ships: Record<ThemeOriginCode, boolean> = {
  [THEME_ORIGIN.UNSPECIFIED]: false,
  [THEME_ORIGIN.PRESET]: true,
  [THEME_ORIGIN.MINE]: false,
}

/**
 * The mode in the window's own words. A mode it has no word for is the
 * system's. Keyed by the schema, so a mode added to it has to be given a word
 * here before this compiles.
 */
const WORDED: Record<ThemeModeCode, Mode | null> = {
  [THEME_MODE.UNSPECIFIED]: null,
  [THEME_MODE.SYSTEM]: 'system',
  [THEME_MODE.LIGHT]: 'light',
  [THEME_MODE.DARK]: 'dark',
}

/** The mode as the schema carries it, read off the words above. */
const ASKED = getNames<Mode, ThemeModeCode>(WORDED)
