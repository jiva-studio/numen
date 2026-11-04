/**
 * Opens what is chosen in the palette. A destination names the file and the place in it, and
 * `openers.ts` decides which editor or reader the file opens in.
 */
import type { Component } from 'vue'
import type { Selector } from '@numen/plugin-host'
import type { SearchDestination } from './search'

/** The window operations that open a chosen destination. */
export interface DestinationDeps {
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
  places.openFile(
    destination.path,
    destination.title || destination.path,
    destination.line,
    destination.selector,
  )
}
