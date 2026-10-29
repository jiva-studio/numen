/** The plain values of a palette that every part of the slice names. */

/**
 * One action available on an item, named by the item's provider. The first is
 * triggered by Enter and the second by Shift+Enter; every action is available
 * by name in the action panel.
 */
export interface PaletteAction {
  readonly id: string
  /** What is written on it. */
  readonly text: string
}
