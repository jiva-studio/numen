/**
 * The values that make up the window appearance: the modes in which a theme's
 * tokens are read, and the sizes.
 */
/** Which half of a `light-dark()` pair every token is read as. */
export type Mode = 'system' | 'light' | 'dark'

/** One theme as the list refers to it. */
export interface Theme {
  /** The origin and the identifier, which together identify the theme when it is requested again. */
  readonly name: string
  /** What the theme is listed as: the file name without the origin, or the name a plugin gives it. */
  readonly title: string
  /** Whether it ships inside the application. */
  readonly isBuiltIn: boolean
  /** The plugin that contributes the theme, and empty for a theme of any other origin. */
  readonly pluginId: string
  /** It declares light and dark itself, so the mode has nothing left to choose. */
  readonly isPinned: boolean
}

/**
 * One value for each of the two sizes: the interface scale and the text scale.
 */
export interface Scales<T> {
  readonly interfaceScale: T
  readonly textScale: T
}

export interface Bounds {
  readonly min: number
  readonly max: number
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

