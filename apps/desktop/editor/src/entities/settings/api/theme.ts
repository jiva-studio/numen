/**
 * Asking the application how the window is drawn, and telling it what a person
 * chose: which theme, which half of a colour pair, and how large.
 *
 * A theme is one stylesheet and nothing here reads inside one: what travels is
 * its name, the text of its file, and which of them is chosen. A size is one
 * multiplier, and how far it goes is the application's to say.
 */
import { Mode as Modes, Origin } from '@numen/protocol'
import { getNames } from '@numen/plugin-sdk'
import { theme } from '@/shared/clients'

/** Which half of a `light-dark()` pair every token is read as. */
export type Mode = 'system' | 'light' | 'dark'

/** One theme as the list refers to it. */
export interface Theme {
  /** The origin and the filename, which is how the theme is asked for again. */
  readonly name: string
  /** What the file is called, without the origin. */
  readonly title: string
  /** Whether it ships inside the application. Every other theme is a file of the person's. */
  readonly isBuiltIn: boolean
  /** It declares light and dark itself, so the mode has nothing left to choose. */
  readonly isPinned: boolean
}

/**
 * One thing said of each of the two sizes: how large the interface is drawn,
 * and how large the text a person reads is set.
 */
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

/** How far each of the two goes. */
export type Ranges = Scales<Bounds>

/** Every theme there is, and what the settings say the window wears. */
export interface Appearance {
  readonly themes: readonly Theme[]
  readonly applied: string
  readonly mode: Mode
  readonly sizes: Sizes
  readonly bounds: Ranges
}

/** What the window asks about how it is drawn. */
export interface Themes {
  getAppearance(): Promise<Appearance>
  readTheme(name: string): Promise<string>
  writeAppearance(name: string, mode: Mode, sizes: Sizes): Promise<string>
  /**
   * The themes the person's folder changed, by name, for as long as the window
   * listens. A theme whose file is gone is named here too.
   */
  watchThemes(signal: AbortSignal): AsyncIterable<readonly string[]>
}

/** The same questions, in the shape the window asks them. */
export const themes: Themes = {
  getAppearance: async () => {
    const answer = await theme.listThemes({})
    return {
      themes: answer.themes.map((one) => ({
        name: one.name,
        title: one.title,
        isBuiltIn: ships[one.origin],
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
  readTheme: async (name) => (await theme.readTheme({ name })).css,
  writeAppearance: async (name, mode, sizes) =>
    (
      await theme.writeAppearance({
        name,
        mode: ASKED[mode],
        interfaceScale: sizes.interfaceScale,
        textScale: sizes.textScale,
      })
    ).error,
  watchThemes: async function* (signal) {
    for await (const update of theme.watchThemes({}, { signal })) yield update.names
  },
}

/** How far a size goes. A bound the application left out is no bound at all. */
const parseBounds = (bounds: { min: number; max: number } | undefined): Bounds => ({
  min: bounds?.min ?? 0,
  max: bounds?.max ?? 0,
})

/**
 * Whether a theme ships inside the application. Keyed by the schema, so an origin
 * added to it has to be answered here before this compiles.
 */
const ships: Record<Origin, boolean> = {
  [Origin.UNSPECIFIED]: false,
  [Origin.PRESET]: true,
  [Origin.MINE]: false,
}

/**
 * The mode in the window's own words. A mode it has no word for is the
 * system's. Keyed by the schema, so a mode added to it has to be given a word
 * here before this compiles.
 */
const WORDED: Record<Modes, Mode | null> = {
  [Modes.UNSPECIFIED]: null,
  [Modes.SYSTEM]: 'system',
  [Modes.LIGHT]: 'light',
  [Modes.DARK]: 'dark',
}

/** The mode as the schema carries it, read off the words above. */
const ASKED = getNames<Mode, Modes>(WORDED)
