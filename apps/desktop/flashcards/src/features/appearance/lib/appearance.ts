/** Which half of a `light-dark()` pair every token is read as. */
export type Mode = 'system' | 'light' | 'dark'

/** The modes, in the order they are offered. */
export const MODES: readonly Mode[] = ['system', 'light', 'dark']

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

/** The themes and the choice the application reports. */
export interface ThemeList {
  readonly themes: readonly ThemeEntry[]
  readonly applied: string
  readonly mode: Mode
}
