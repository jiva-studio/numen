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
  worn: [] as string[],
  /** How often an open editor was told to take its measurements again. */
  measured: 0,
  /** Every key the window handed down to the book it is showing. */
  pressed: [] as string[],
  /** The vaults the window asked to be shown, in the order it asked. */
  opened: [] as string[],
  /** Every file the window asked what it carries, in the order it asked. */
  carried: [] as string[],
  /** Each run the window asked for, and each transcript it dropped. */
  ran: [] as string[],
  /** How often a folder was asked for, which is a vault being added. */
  chose: 0,
  /** What the window said currently open, the last of it last. */
  openTabs: [] as { tabs: readonly Tab[]; front: string }[],
}

/** Nothing asked yet, which is where every test begins. */
export const forgetRequests = () => {
  requests.made = []
  requests.renamed = []
  requests.removed = []
  requests.moved = []
  requests.folders = []
  requests.worn = []
  requests.measured = 0
  requests.pressed = []
  requests.opened = []
  requests.carried = []
  requests.ran = []
  requests.chose = 0
  requests.openTabs = []
}
