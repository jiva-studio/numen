/**
 * Asking the application what themes there are, and telling it which is worn.
 *
 * A theme is one stylesheet and nothing here reads inside one: what travels is
 * its name, the text of its file, and which of them is chosen.
 */
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { Mode as Modes, Shelf, ThemeService } from '@numen/protocol'

export const dressing = createClient(
  ThemeService,
  createConnectTransport({ baseUrl: window.location.origin }),
)

/** Which half of a `light-dark()` pair every token is read as. */
export type Mode = 'system' | 'light' | 'dark'

/** One theme as the list refers to it. */
export interface Wearable {
  /** The shelf and the filename, which is how the theme is asked for again. */
  readonly name: string
  /** What the file is called, without the shelf. */
  readonly title: string
  /** Whether it ships inside the application. Every other theme is a file of the person's. */
  readonly shipped: boolean
  /** It declares light and dark itself, so the mode has nothing left to choose. */
  readonly pinned: boolean
}

/** Every theme there is, and what the settings say is worn. */
export interface Catalogue {
  readonly themes: readonly Wearable[]
  readonly applied: string
  readonly mode: Mode
}

/** What the window asks about the themes it can wear. */
export interface Themes {
  catalogue(): Promise<Catalogue>
  /** The text of one theme's file, as the file stands when it is asked for. */
  text(name: string): Promise<string>
  /** The theme and the mode written into the settings, and why they were not. */
  chooses(name: string, mode: Mode): Promise<string>
  /**
   * The themes the person's folder changed, by name, for as long as the window
   * listens. A theme whose file is gone is named here too.
   */
  changed(signal: AbortSignal): AsyncIterable<readonly string[]>
}

/** The same questions, in the shape the window asks them. */
export const themes: Themes = {
  catalogue: async () => {
    const answer = await dressing.themes({})
    return {
      themes: answer.themes.map((one) => ({
        name: one.name,
        title: one.title,
        shipped: one.shelf === Shelf.PRESET,
        pinned: one.pinned,
      })),
      applied: answer.applied,
      mode: worded(answer.mode),
    }
  },
  text: async (name) => (await dressing.theme({ name })).css,
  chooses: async (name, mode) => (await dressing.choose({ name, mode: ASKED[mode] })).failed,
  changed: async function* (signal) {
    for await (const said of dressing.changed({}, { signal })) yield said.names
  },
}

/** The mode as the schema carries it. */
const ASKED: Record<Mode, Modes> = {
  system: Modes.SYSTEM,
  light: Modes.LIGHT,
  dark: Modes.DARK,
}

/** The mode in the window's own words. A mode it has no word for is the system's. */
const worded = (said: Modes): Mode => {
  switch (said) {
    case Modes.LIGHT:
      return 'light'
    case Modes.DARK:
      return 'dark'
    default:
      return 'system'
  }
}
