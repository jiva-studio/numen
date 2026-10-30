/**
 * Where the window is taken when something is chosen in the palette.
 *
 * A name is a thing and is put in front where the plugin that shows notes puts
 * it; a heading and a passage are places in a file, and open it where they stand.
 * Which editor or reader the file opens in is not decided here: a destination
 * names the file and the place in it, and `openers.ts` opens it.
 */
import type { Component } from 'vue'
import type { Selector } from '@numen/plugin-sdk'
import type { SearchDestination } from './search'

/** The window operations that open a chosen destination. */
export interface DestinationDeps {
  /** Shows a note in the plugin that displays notes. */
  travel(path: string): Promise<void>
  /**
   * Shows a file in the editor or reader for its type, at the chosen line or
   * selector.
   */
  openFile(path: string, title: string, line?: number, selector?: Selector): void
  /** The icon of the reader that opens a non-note file, or undefined when none does. */
  findFileIcon?(path: string): Component | undefined
}

/** Opens the chosen destination. A null destination does nothing. */
export async function openDestination(
  destination: SearchDestination | null,
  places: DestinationDeps,
): Promise<void> {
  if (!destination) return
  if (destination.at === 'locate') {
    await places.travel(destination.path)
    return
  }
  places.openFile(
    destination.path,
    destination.title || destination.path,
    destination.line,
    destination.selector,
  )
}
