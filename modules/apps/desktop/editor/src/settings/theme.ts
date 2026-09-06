/**
 * Asking the application how the window is drawn, and telling it what a person
 * chose: which theme, which half of a colour pair, and how large.
 *
 * A theme is one stylesheet and nothing here reads inside one: what travels is
 * its name, the text of its file, and which of them is chosen. A size is one
 * multiplier, and how far it goes is the application's to say.
 */
import { createClient } from '@connectrpc/connect'
import { Mode as Modes, Shelf, ThemeService } from '@numen/protocol'
import { namesOf, transport } from '@numen/wire'

const theme = createClient(ThemeService, transport)

/** Which half of a `light-dark()` pair every token is read as. */
export type Mode = 'system' | 'light' | 'dark'

/** One theme as the list refers to it. */
export interface Theme {
  /** The shelf and the filename, which is how the theme is asked for again. */
  readonly name: string
  /** What the file is called, without the shelf. */
  readonly title: string
  /** Whether it ships inside the application. Every other theme is a file of the person's. */
  readonly shipped: boolean
  /** It declares light and dark itself, so the mode has nothing left to choose. */
  readonly pinned: boolean
}

/**
 * One thing said of each of the two sizes: how large the interface is drawn,
 * and how large the text a person reads is set.
 */
export interface Scales<T> {
  readonly interfaceScale: T
  readonly textScale: T
}

/** How far a size goes, at each end. A number outside them is refused. */
export interface Bounds {
  readonly least: number
  readonly most: number
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
  appearance(): Promise<Appearance>
  /** The text of one theme's file, as the file stands when it is asked for. */
  text(name: string): Promise<string>
  /**
   * The theme, the mode and the two sizes written into the settings, and why
   * they were not. A size outside its bounds is refused and nothing is written.
   */
  chooses(name: string, mode: Mode, sizes: Sizes): Promise<string>
  /**
   * The themes the person's folder changed, by name, for as long as the window
   * listens. A theme whose file is gone is named here too.
   */
  changed(signal: AbortSignal): AsyncIterable<readonly string[]>
}

/** The same questions, in the shape the window asks them. */
export const themes: Themes = {
  appearance: async () => {
    const answer = await theme.listThemes({})
    return {
      themes: answer.themes.map((one) => ({
        name: one.name,
        title: one.title,
        shipped: ships[one.shelf],
        pinned: one.pinned,
      })),
      applied: answer.applied,
      mode: WORDED[answer.mode] ?? 'system',
      sizes: { interfaceScale: answer.interfaceScale, textScale: answer.textScale },
      bounds: {
        interfaceScale: ranged(answer.interfaceScaleBounds),
        textScale: ranged(answer.textScaleBounds),
      },
    }
  },
  text: async (name) => (await theme.readTheme({ name })).css,
  chooses: async (name, mode, sizes) =>
    (
      await theme.writeAppearance({
        name,
        mode: ASKED[mode],
        interfaceScale: sizes.interfaceScale,
        textScale: sizes.textScale,
      })
    ).failed,
  changed: async function* (signal) {
    for await (const said of theme.watchThemes({}, { signal })) yield said.names
  },
}

/** How far a size goes. A bound the application left out is no bound at all. */
const ranged = (said: { least: number; most: number } | undefined): Bounds => ({
  least: said?.least ?? 0,
  most: said?.most ?? 0,
})

/**
 * Whether a theme ships inside the application. Keyed by the schema, so a shelf
 * added to it has to be answered here before this compiles.
 */
const ships: Record<Shelf, boolean> = {
  [Shelf.UNSPECIFIED]: false,
  [Shelf.PRESET]: true,
  [Shelf.MINE]: false,
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
const ASKED = namesOf<Mode, Modes>(WORDED)
