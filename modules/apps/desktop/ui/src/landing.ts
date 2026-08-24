/**
 * Where the window is taken when something is chosen in the palette.
 *
 * A name is a thing and travels in the plex the person is looking at; a
 * heading and a passage are places in a note, and open it where they stand. A
 * passage from a source that is not a note opens that source where it stands.
 */
import type { Landing } from './finding'

/** What the window offers whatever was chosen. */
export interface Places {
  /** A note put in front of the person, in the plex they are looking at. */
  travel(path: string): Promise<void>
  /** A source opened at a stretch of its own text. */
  opensAt(path: string, run: { start: number; length: number }): Promise<void>
  /** A note opened in a tab of its own, under the name it is called by. */
  shows(path: string, title: string): void
  /** The line an open note is to stand on. */
  entersAt(path: string, line: number): void
}

/** Somewhere chosen, taken. Nothing chosen takes the person nowhere. */
export async function lands(landing: Landing | null, places: Places): Promise<void> {
  if (!landing) return
  if (landing.at === 'plex') return void places.travel(landing.path)
  if (landing.at === 'document') {
    await places.opensAt(landing.path, {
      start: landing.start ?? 0,
      length: landing.length ?? 0,
    })
    return
  }
  places.shows(landing.path, landing.title || landing.path)
  if (landing.line !== undefined) places.entersAt(landing.path, landing.line)
}
