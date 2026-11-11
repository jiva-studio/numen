/**
 * The files the window has an editor open on, as the commands reach them.
 */
import type { Selector } from '@numen/plugin-loader'

/** Where in the file a command that opens it moves to: a line of a note, or the selector of a reader. */
export interface CommandPlace {
  readonly line?: number
  readonly selector?: Selector
}

/**
 * The files the window has an editor open on, as accessed by commands. Notes
 * of every type are included.
 */
export interface Notes {
  /** The identity of the tab standing at a file, and nothing where none does. */
  getTabAt(path: string): string | null
  getPath(id: string): string
  asking(id: string): boolean
  /** Answers once nothing of that note is on its way to the file. */
  flushPending(id: string): Promise<void>
  /** The tab holding a note releases it. */
  close(id: string): void
  /**
   * Opens a file in the editor matching its type, in its own tab or beside the
   * current one.
   */
  openFile(path: string, title: string, placement: 'current' | 'split', at?: CommandPlace): void
}

/** The result of a flush request on a tab: which tab it was, and whether it is still waiting. */
export interface FlushResult {
  readonly tabId: string | null
  readonly isWaiting: boolean
}

/**
 * The tab holding a note, once no write of the note is pending. A tab waiting on
 * the user flushes nothing and reports so.
 */
export const flushTab = async (path: string, notes: Notes): Promise<FlushResult> => {
  const tabId = notes.getTabAt(path)
  if (tabId === null) return { tabId, isWaiting: false }
  if (notes.asking(tabId)) return { tabId, isWaiting: true }
  await notes.flushPending(tabId)
  return { tabId, isWaiting: false }
}
