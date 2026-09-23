/** The plain values of a palette that every part of the slice names. */

/**
 * One thing that can be done to an item, named by whoever offers it. The first
 * is what Enter reaches and the second what Shift and Enter reach; every one of
 * them is reached by its name in the action panel.
 */
export interface PaletteAction {
  readonly id: string
  /** What is written on it. */
  readonly text: string
}
