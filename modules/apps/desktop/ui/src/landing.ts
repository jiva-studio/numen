/**
 * Where the window is taken when something is chosen in the palette.
 *
 * A name is a thing and travels in the plex the person is looking at; a
 * heading and a passage are places in a source, and open it where they stand.
 * Which editor the source opens in is not decided here: a landing names the
 * file and the place in it, and `putting.ts` opens it.
 */
import type { Landing } from './finding'

/** What the window offers whatever was chosen. */
export interface LandingDeps {
  /** A note put in front of the person, in the plex they are looking at. */
  travel(path: string): Promise<void>
  /**
   * A source put in front of the person at a stretch of its own text, in the
   * editor made for what it is.
   */
  opensAt(path: string, run: { start: number; length: number }): Promise<void>
  /**
   * A file put in front of the person, in the editor made for what it is, at
   * the line it was chosen at.
   */
  opens(path: string, title: string, line?: number): void
}

/** Somewhere chosen, taken. Nothing chosen takes the person nowhere. */
export async function lands(landing: Landing | null, places: LandingDeps): Promise<void> {
  if (!landing) return
  if (landing.at === 'plex') return void places.travel(landing.path)
  if (landing.at === 'document') {
    await places.opensAt(landing.path, {
      start: landing.start ?? 0,
      length: landing.length ?? 0,
    })
    return
  }
  places.opens(landing.path, landing.title || landing.path, landing.line)
}
