/**
 * The player at the top of a url tab: how tall it is drawn while a person drags its bar, and the
 * messages a player in a frame takes from the page holding it.
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


/** What the text parses to as JSON, and nothing where it is not JSON. */
export function parseJson(text: string): unknown {
  try {
    return JSON.parse(text)
  } catch {
    return undefined
  }
}

/** Where a message to a frame is addressed: the origin of its address. */
export function getFrameOrigin(embed: string): string | null {
  try {
    return new URL(embed).origin
  } catch {
    // An embed that is not a parseable absolute URL has no origin to post the message to.
    return null
  }
}

/** The message that plays a frame from a moment, in milliseconds. */
function createSeekMessage(ms: number): string {
  return JSON.stringify({ event: 'command', func: 'seekTo', args: [ms / 1000, true] })
}

/** What a frame is told through. */
export type MessageTarget = Pick<Window, 'postMessage'>

/** A frame told to play from a moment, and nobody else is told. */
export function seekFrame(
  target: MessageTarget | null | undefined,
  embed: string,
  ms: number,
): void {
  const origin = getFrameOrigin(embed)
  if (origin === null) return
  target?.postMessage(createSeekMessage(ms), origin)
  target?.postMessage(JSON.stringify({ event: 'command', func: 'playVideo', args: [] }), origin)
}
