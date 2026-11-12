/** The theme service of the application, in the terms the appearance commands use. */
import {
  THEME_MODE,
  THEME_ORIGIN,
  type ThemeModeCode,
  type ThemeOriginCode,
} from '@numen/core-client'
import { theme } from '@/shared/api/clients'
import type { Mode, ThemeList, ThemeOrigin } from '../lib/appearance'

/** The theme service as the window uses it. */
export interface ThemeClient {
  listThemes(): Promise<ThemeList>
  readTheme(name: string): Promise<string>
  /** Writes the choice, and answers why it was not written; empty where it was. */
  writeAppearance(name: string, mode: Mode): Promise<string>
  /** The names of themes that changed, until the signal aborts. */
  watchThemes(signal: AbortSignal): AsyncIterable<readonly string[]>
}

/**
 * The origin in the window's own words. Keyed by the schema, so an origin added to it has to be
 * answered here before this compiles.
 */
const ORIGINS: Record<ThemeOriginCode, ThemeOrigin> = {
  [THEME_ORIGIN.UNSPECIFIED]: 'other',
  [THEME_ORIGIN.PRESET]: 'preset',
  [THEME_ORIGIN.MINE]: 'mine',
  [THEME_ORIGIN.PLUGIN]: 'other',
}

/**
 * The mode in the window's own words; a mode it has no word for is the system's. Keyed by the
 * schema, so a mode added to it has to be given a word here before this compiles.
 */
const WORDED: Record<ThemeModeCode, Mode | null> = {
  [THEME_MODE.UNSPECIFIED]: null,
  [THEME_MODE.SYSTEM]: 'system',
  [THEME_MODE.LIGHT]: 'light',
  [THEME_MODE.DARK]: 'dark',
}

/** The mode as the schema carries it. */
const CODES: Record<Mode, ThemeModeCode> = {
  system: THEME_MODE.SYSTEM,
  light: THEME_MODE.LIGHT,
  dark: THEME_MODE.DARK,
}

export const themeClient: ThemeClient = {
  listThemes: async () => {
    const answer = await theme.listThemes({})
    return {
      themes: answer.themes.map((one) => ({
        name: one.name,
        title: one.title,
        origin: ORIGINS[one.origin],
        pluginId: one.pluginId,
        isPinned: one.isPinned,
      })),
      applied: answer.applied,
      mode: WORDED[answer.mode] ?? 'system',
    }
  },
  readTheme: async (name) => (await theme.readTheme({ name })).css,
  writeAppearance: async (name, mode) =>
    (await theme.writeAppearance({ name, mode: CODES[mode] })).error,
  watchThemes: async function* (signal) {
    for await (const update of theme.watchThemes({}, { signal })) yield update.names
  },
}
