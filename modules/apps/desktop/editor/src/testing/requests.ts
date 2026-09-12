/**
 * What the window asked the application for, in the order it asked.
 *
 * Every mock writes down what it was handed here, and a test reads it back.
 */
import type { Tab } from '@/entities/tab'

export const requests = {
  made: [] as string[],
  renamed: [] as string[],
  removed: [] as string[],
  moved: [] as string[],
  folders: [] as string[],
  urls: [] as string[],
  /** The decks and the stencils the window asked for, in the order it asked. */
  cards: [] as string[],
  /** Each field rename the window asked the vault for. */
  renamedField: [] as string[],
  /** The cards each of those deck writes carried, by name. */
  wrote: [] as string[],
  worn: [] as string[],
  /** How often an open editor was told to take its measurements again. */
  measured: 0,
  /** Every key the window handed down to the book it is showing. */
  pressed: [] as string[],
  /** The vaults the window asked to be shown, in the order it asked. */
  opened: [] as string[],
  /** The recordings the window listened to, in the order it asked. */
  listened: [] as string[],
  /** The transcripts the window wrote, as the words each carried. */
  transcribed: [] as string[],
  /** Every file the window asked what it carries, in the order it asked. */
  carried: [] as string[],
  /** Each run the window asked for, and each transcript it dropped. */
  ran: [] as string[],
  /** How often a folder was asked for, which is a vault being added. */
  chose: 0,
  /** What the window said the person has open, the last of it last. */
  openTabs: [] as { tabs: readonly Tab[]; front: string }[],
}

/** Nothing asked yet, which is where every test begins. */
export const forgetRequests = () => {
  requests.made = []
  requests.cards = []
  requests.renamedField = []
  requests.wrote = []
  requests.renamed = []
  requests.removed = []
  requests.moved = []
  requests.folders = []
  requests.worn = []
  requests.measured = 0
  requests.pressed = []
  requests.opened = []
  requests.listened = []
  requests.transcribed = []
  requests.carried = []
  requests.ran = []
  requests.chose = 0
  requests.openTabs = []
}
