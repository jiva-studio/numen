/**
 * The values that make up the window appearance: the themes, the modes in which a theme's tokens are
 * read, and the sizes, with the service the window reads them from.
 */
/** Which half of a `light-dark()` pair every token is read as. */
export type Mode = 'system' | 'light' | 'dark'

/** Where a theme comes from: shipped with the application, the person's own, or any other source. */
export type ThemeOrigin = 'preset' | 'mine' | 'other'

/** One theme as the application lists it. */
export interface Theme {
  readonly name: string
  readonly title: string
  readonly origin: ThemeOrigin
  readonly pluginId: string
  readonly isPinned: boolean
}

/** One value for each of the two sizes: the interface scale and the text scale. */
export interface Scales<T> {
  readonly interfaceScale: T
  readonly textScale: T
}

export interface Bounds {
  readonly min: number
  readonly max: number
}

/** The two multipliers the window is drawn at. One is as designed. */
export type Sizes = Scales<number>

/** Every available theme, and the theme, mode and sizes the settings apply to the window. */
export interface Appearance {
  readonly themes: readonly Theme[]
  readonly applied: string
  readonly mode: Mode
  readonly sizes: Sizes
  readonly bounds: Scales<Bounds>
}

/** The queries the window makes about its appearance. */
export interface Themes {
  getAppearance(): Promise<Appearance>
  readTheme(name: string): Promise<string>
  /** The names of the themes that changed, until the signal aborts. */
  watchThemes(signal: AbortSignal): AsyncIterable<readonly string[]>
}

/** Where the appearance reports a failure. */
export type AppearanceWriter = (text: string, severity: 'error') => void

/** The words the window appearance reports in. */
export interface AppearanceWords {
  readonly unlisted: string
  readonly themeUnreadable: string
}

/** The multiplier that draws everything the size it was designed at. */
export const DESIGNED = 1

/** The modes, in the order they are offered. */
export const MODES: readonly Mode[] = ['system', 'light', 'dark']

/** What `color-scheme` is written as for each mode. */
export const SCHEMES: Record<Mode, string> = {
  system: 'light dark',
  light: 'light',
  dark: 'dark',
}

/** The sentence a failure is written as. */
export const getFailureText = (error: unknown): string =>
  error instanceof Error ? error.message : String(error)
