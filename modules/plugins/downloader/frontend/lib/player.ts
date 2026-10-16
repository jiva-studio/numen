/**
 * The messages a player in a frame takes from the page holding it.
 */

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
export function createSeekMessage(ms: number): string {
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
}
