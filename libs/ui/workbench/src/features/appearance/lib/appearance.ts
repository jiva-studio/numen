/** The theme entries and size and mode values drawn by the window. */
/** Which half of a `light-dark()` pair every token is read as. */
export type Mode = 'system' | 'light' | 'dark'

/** Where a theme comes from: shipped with the application, the person's own, or any other source. */
type ThemeOrigin = 'preset' | 'mine' | 'other'

/** One theme as the application lists it. */
export interface Theme {
  readonly name: string
  readonly title: string
  readonly origin: ThemeOrigin
  readonly pluginId: string
  readonly isPinned: boolean
}

/** One value for each of the two sizes: the interface scale and the text scale. */
export interface Sizes {
  readonly interfaceScale: number
  readonly textScale: number
}

/** Where the appearance reports a failure. */
export type AppearanceWriter = (text: string, severity: 'error') => void

/** The words the appearance reports a failure in. */
export interface AppearanceWords {
  readonly unlisted: string
  readonly themeUnreadable: string
}

/** The styles the window was served with and the changes applied as the appearance is read. */
export interface AppearanceOutput {
  readonly servedTheme: string
  readonly writeMode: (mode: Mode) => void
  readonly writeTheme: (css: string) => void
  readonly writeSizes: (sizes: Sizes) => void
}

/** The theme, mode and sizes the settings apply to the window, and every theme listed. */
export interface ThemeState {
  readonly themes: readonly Theme[]
  readonly applied: string
  readonly mode: Mode
  readonly sizes: Sizes
}

/** The themes the application provides to the appearance. */
export interface ThemeSource {
  getAppearance(): Promise<ThemeState>
  readTheme(name: string): Promise<string>
  /** The names of themes that changed, until the signal aborts. */
  watchThemes(signal: AbortSignal): AsyncIterable<readonly string[]>
}

/** What a follower reads of the window it follows for. */
interface StreamReaderDeps {
  isOpen(): boolean
  setLost(message: string): void
  wait(ms: number): Promise<void>
}

/** Follows a stream for as long as the window is open, resuming it after a loss. */
export type CreateStreamReader = (
  deps: StreamReaderDeps,
) => <T>(stream: () => AsyncIterable<T>, each: (item: T) => void | Promise<void>) => Promise<void>

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
