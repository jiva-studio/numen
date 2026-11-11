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

/** A value that survives JSON.stringify and JSON.parse unchanged. */
type JsonValue =
  | string
  | number
  | boolean
  | null
  | readonly JsonValue[]
  | { readonly [key: string]: JsonValue }

function parseJson(text: string): JsonValue | undefined {
  try {
    return JSON.parse(text)
  } catch {
    return undefined
  }
}

type JsonObject = { readonly [key: string]: JsonValue }

function isObject(value: JsonValue | undefined): value is JsonObject {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

/** What a player in a frame reports about itself. */
interface PlayerInfo {
  currentTime?: number
  playerState?: number
}

/** The position and state a player message reports; null for a message that is not a report. */
export function readPlayerInfo(text: string): PlayerInfo | null {
  const payload = parseJson(text)
  if (!isObject(payload) || payload['event'] !== 'infoDelivery') return null
  const info = payload['info']
  if (!isObject(info)) return null
  const { currentTime, playerState } = info
  const result: PlayerInfo = {}
  if (typeof currentTime === 'number') result.currentTime = currentTime
  if (typeof playerState === 'number') result.playerState = playerState
  return result
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

function createSeekMessage(ms: number): string {
  return JSON.stringify({ event: 'command', func: 'seekTo', args: [ms / 1000, true] })
}

/** What a frame is told through. */
export interface MessageTarget {
  postMessage(message: string, targetOrigin: string): void
}

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
