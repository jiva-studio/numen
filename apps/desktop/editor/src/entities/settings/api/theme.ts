/**
 * Queries the application for the window's appearance: the theme, the mode of a color
 * pair, and the sizes.
 *
 * A theme is one stylesheet, and nothing here reads inside one: the transferred
 * data is its name, the text of its file, and which theme is selected. A size is
 * one multiplier, and its permitted range is defined by the application.
 */
import {
  THEME_MODE,
  THEME_ORIGIN,
  type ThemeModeCode,
  type ThemeOriginCode,
} from '@numen/core-client'
import { theme } from '@/shared/api/clients'

import type { Bounds, Mode, Scales, Theme } from '../lib/appearance'
export type { Mode, Scales }

/** The two multipliers the window is drawn at. One is as designed. */
export type Sizes = Scales<number>

/** How far each of the two goes. */
type Ranges = Scales<Bounds>

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
  /**
   * The names of themes changed in the user's folder, and of plugin themes that
   * appeared or vanished with the enabled plugins, until the signal aborts. A
   * theme whose file was deleted is included.
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
        pluginId: one.pluginId,
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
  watchThemes: async function* (signal) {
    for await (const update of theme.watchThemes({}, { signal })) yield update.names
  },
}

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
  [THEME_ORIGIN.PLUGIN]: false,
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
