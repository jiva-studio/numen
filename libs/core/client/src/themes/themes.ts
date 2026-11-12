/**
 * The theme service in the terms of the windows: the themes, the theme and mode applied, the two
 * sizes and how far each goes. A theme is one stylesheet, and nothing here reads inside one.
 */
import type { Client } from '@connectrpc/connect'
import type { ThemeService } from '@numen/protocol'
import { THEME_MODE, THEME_ORIGIN, type ThemeModeCode, type ThemeOriginCode } from '../clients'

/** Which half of a `light-dark()` pair every token is read as. */
export type ThemeMode = 'system' | 'light' | 'dark'

/** Where a theme comes from: shipped with the application, the person's own, or any other source. */
export type ThemeOrigin = 'preset' | 'mine' | 'other'

/** One theme as the application lists it. */
export interface ThemeEntry {
  readonly name: string
  readonly title: string
  readonly origin: ThemeOrigin
  readonly pluginId: string
  readonly isPinned: boolean
}

/** One value for each of the two sizes. */
export interface ThemeScales<T> {
  readonly interfaceScale: T
  readonly textScale: T
}

/** How far a size goes; both ends are zero where the application names none. */
export interface ThemeBounds {
  readonly min: number
  readonly max: number
}

/** Every theme, and the theme, mode and sizes the settings apply to the window. */
export interface ThemeAppearance {
  readonly themes: readonly ThemeEntry[]
  readonly applied: string
  readonly mode: ThemeMode
  readonly sizes: ThemeScales<number>
  readonly bounds: ThemeScales<ThemeBounds>
}

/** The calls of ThemeService the windows make. */
export type ThemeRpc = Pick<
  Client<typeof ThemeService>,
  'listThemes' | 'readTheme' | 'writeAppearance' | 'watchThemes'
>

/** The theme service as the windows use it. */
export interface ThemeClient {
  getAppearance(): Promise<ThemeAppearance>
  readTheme(name: string): Promise<string>
  /** Writes the theme and the mode chosen, and answers why they were not written; empty where they were. */
  writeAppearance(name: string, mode: ThemeMode): Promise<string>
  /**
   * The names of themes changed in the user's folder, and of plugin themes that appeared or vanished
   * with the enabled plugins, until the signal aborts. A theme whose file was deleted is included.
   */
  watchThemes(signal: AbortSignal): AsyncIterable<readonly string[]>
}

/** Keyed by the schema, so an origin added to it has to be answered here before this compiles. */
const ORIGINS: Record<ThemeOriginCode, ThemeOrigin> = {
  [THEME_ORIGIN.UNSPECIFIED]: 'other',
  [THEME_ORIGIN.PRESET]: 'preset',
  [THEME_ORIGIN.MINE]: 'mine',
  [THEME_ORIGIN.PLUGIN]: 'other',
}

/**
 * A mode without a word is the system's. Keyed by the schema, so a mode added to it has to be given
 * a word here before this compiles.
 */
const WORDED: Record<ThemeModeCode, ThemeMode | null> = {
  [THEME_MODE.UNSPECIFIED]: null,
  [THEME_MODE.SYSTEM]: 'system',
  [THEME_MODE.LIGHT]: 'light',
  [THEME_MODE.DARK]: 'dark',
}

const CODES: Record<ThemeMode, ThemeModeCode> = {
  system: THEME_MODE.SYSTEM,
  light: THEME_MODE.LIGHT,
  dark: THEME_MODE.DARK,
}

const parseBounds = (bounds: { min: number; max: number } | undefined): ThemeBounds => ({
  min: bounds?.min ?? 0,
  max: bounds?.max ?? 0,
})

/** The theme service over its client. */
export function createThemeClient(rpc: ThemeRpc): ThemeClient {
  return {
    getAppearance: async () => {
      const answer = await rpc.listThemes({})
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
        sizes: { interfaceScale: answer.interfaceScale, textScale: answer.textScale },
        bounds: {
          interfaceScale: parseBounds(answer.interfaceScaleBounds),
          textScale: parseBounds(answer.textScaleBounds),
        },
      }
    },
    readTheme: async (name) => (await rpc.readTheme({ name })).css,
    writeAppearance: async (name, mode) =>
      (await rpc.writeAppearance({ name, mode: CODES[mode] })).error,
    watchThemes: async function* (signal) {
      for await (const update of rpc.watchThemes({}, { signal })) yield update.names
    },
  }
}
