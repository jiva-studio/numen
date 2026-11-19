/** The theme entries and size and mode values drawn by the window. */
import type { Caught } from '@numen/ui'

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

/** The modes, in the order they are offered. */
export const MODES: readonly Mode[] = ['system', 'light', 'dark']

/** What `color-scheme` is written as for each mode. */
export const SCHEMES: Record<Mode, string> = {
  system: 'light dark',
  light: 'light',
  dark: 'dark',
}

/** The sentence a failure is written as. */
export const getFailureText = (error: Caught): string =>
  error instanceof Error ? error.message : String(error)
