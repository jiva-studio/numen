import type { Caught } from '../errors'
import type { ThemeMode, ThemeScales } from './themes'

/** The styles the window was served with and the changes applied as the appearance is read. */
export interface AppearanceOutput {
  readonly servedTheme: string
  writeMode(mode: ThemeMode): void
  writeTheme(css: string): void
  writeSizes(sizes: ThemeScales<number>): void
}

export type AppearanceWriter = (text: string, severity: 'error') => void

export interface AppearanceWords {
  readonly unlisted: string
  readonly themeUnreadable: string
}

export const getFailureText = (error: Caught): string =>
  error instanceof Error ? error.message : String(error)
