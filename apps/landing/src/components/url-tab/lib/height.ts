/**
 * How tall the player is drawn while a person drags its bar.
 *
 * The room a window has is a number handed in, so what the drag works out is
 * the same on any machine.
 */

/** The shortest the player is drawn, whatever the drag asks for. */
export const LEAST = 120

/** What is left for the rest of the page above and below the player. */
const KEPT = 160

/** The tallest the player is drawn in a window of this height. */
export function getTallest(windowHeight: number): number {
  return Math.max(LEAST, windowHeight - KEPT)
}

/**
 * How tall the player stands, given how tall it was when the drag began, how
 * far the pointer has come, and the room the window has.
 */
export function getDraggedHeight(was: number, distance: number, windowHeight: number): number {
  return Math.min(getTallest(windowHeight), Math.max(LEAST, was + distance))
}
