/**
 * What the screen a window opens on is drawn over.
 *
 * Both windows open on it: the editor while it holds no tab, and flashcards
 * while it has not been sent into a vault. The rows differ and the screen does
 * not.
 */
import type { Component } from 'vue'
import type { PaletteKeys } from '../palette/item'

/** One way in: what it is called, what draws it, and the keystroke that reaches it. */
export interface Way {
  readonly id: string
  readonly text: string
  /** What is drawn in front of it. A way with none is drawn without one. */
  readonly icon?: Component
  readonly keys?: PaletteKeys
}

/** One vault of the list, as the screen draws it. */
export interface Held {
  readonly id: string
  readonly name: string
  /** The folder it stands for, absolute on this machine. */
  readonly path: string
  /** What is true of this row and not of the ones beside it. */
  readonly detail?: string
  /**
   * Whether what the window has to say about this row is still on its way. Such
   * a row is drawn and carries no letter, and it is not pressed until it is.
   */
  readonly working?: boolean
}

/** What the screen offers below the list, where a window offers anything. */
export interface Offer {
  readonly text: string
  readonly detail?: string
  readonly icon?: Component
  /** The keystroke that reaches it, where one does. */
  readonly keys?: PaletteKeys
}
