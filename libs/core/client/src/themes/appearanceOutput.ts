import type { ThemeMode, ThemeScales } from './themes'

/** The styles the window was served with and the changes applied as the appearance is read. */
export interface AppearanceOutput {
  readonly servedTheme: string
  readonly writeMode: (mode: ThemeMode) => void
  readonly writeTheme: (css: string) => void
  readonly writeSizes: (sizes: ThemeScales<number>) => void
}

export type AppearanceWriter = (text: string, severity: 'error') => void

export interface AppearanceWords {
  readonly unlisted: string
  readonly themeUnreadable: string
}

export const getFailureText = (error: unknown): string =>
  error instanceof Error ? error.message : String(error)
