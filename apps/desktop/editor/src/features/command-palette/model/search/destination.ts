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

/** What the window offers whatever was chosen. */
export interface DestinationDeps {
  /** A note put in front of the person, where the plugin that shows notes puts it. */
  travel(path: string): Promise<void>
  /**
   * A file put in front of the person, in the editor or reader made for it, at
   * the line or the selector it was chosen at.
   */
  openFile(path: string, title: string, line?: number, selector?: Selector): void
  /** The icon of the reader that opens a file that is no note, and nothing where none does. */
  findFileIcon?(path: string): Component | undefined
}

/** Somewhere chosen, taken. Nothing chosen takes the person nowhere. */
export async function openDestination(
  destination: SearchDestination | null,
  places: DestinationDeps,
): Promise<void> {
  if (!destination) return
  if (destination.at === 'travel') return void places.travel(destination.path)
  places.openFile(
    destination.path,
    destination.title || destination.path,
    destination.line,
    destination.selector,
  )
}
