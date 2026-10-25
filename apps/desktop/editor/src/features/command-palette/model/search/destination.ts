/**
 * Where the window is taken when something is chosen in the palette.
 *
 * A name is a thing and travels in the plex the person is looking at; a
 * heading and a passage are places in a source, and open it where they stand.
 * Which editor the source opens in is not decided here: a destination names the
 * file and the place in it, and `openers.ts` opens it.
 */
import type { SearchDestination } from './search'

/** What the window offers whatever was chosen. */
export interface DestinationDeps {
  /** A note put in front of the person, in the plex they are looking at. */
  travel(path: string): Promise<void>
  /**
   * A source put in front of the person at a span of its own text, in the
   * editor made for what it is.
   */
  openFileAt(path: string, run: { from: number; to: number }): Promise<void>
  /**
   * A file put in front of the person, in the editor made for what it is, at
   * the line it was chosen at.
   */
  openFile(path: string, title: string, line?: number): void
}

/** Somewhere chosen, taken. Nothing chosen takes the person nowhere. */
export async function openDestination(
  destination: SearchDestination | null,
  places: DestinationDeps,
): Promise<void> {
  if (!destination) return
  if (destination.at === 'plex') return void places.travel(destination.path)
  if (destination.at === 'document') {
    const from = destination.start ?? 0
    await places.openFileAt(destination.path, { from, to: from + (destination.length ?? 0) })
    return
  }
  places.openFile(destination.path, destination.title || destination.path, destination.line)
}
