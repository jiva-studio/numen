/**
 * The values that make up the window appearance: the modes in which a theme's
 * tokens are read, and the sizes.
 */
/** Which half of a `light-dark()` pair every token is read as. */
export type Mode = 'system' | 'light' | 'dark'

/** One theme as the list refers to it. */
export interface Theme {
  readonly name: string
  readonly title: string
  readonly isBuiltIn: boolean
  readonly pluginId: string
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
