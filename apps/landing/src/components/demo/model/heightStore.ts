/**
 * Where the height a person drew the player is kept between runs.
 */
import { LEAST } from '../lib/frame'

/** Keeps one height, and gives it back. */
export interface HeightStore {
  /** How tall a person drew the player last, and nothing where they have not drawn it. */
  read(): number | null
  /** Whether the height was kept. A machine that keeps nothing loses it when the page closes. */
  write(height: number): boolean
}

/** What the store needs of a storage. */
export type KeyValueStorage = Pick<Storage, 'getItem' | 'setItem'>

const KEY = 'numen.embed.height'

/** The storage is asked for on every call, and a call it refuses comes to nothing. */
export function createHeightStore(
  getStorage: () => KeyValueStorage = () => localStorage,
): HeightStore {
  return {
    read() {
      try {
        const kept = Number(getStorage().getItem(KEY))
        return kept >= LEAST ? kept : null
      } catch {
        // A browser that keeps nothing for this page has no height to give back.
        return null
      }
    },
    write(height) {
      try {
        getStorage().setItem(KEY, String(height))
        return true
      } catch {
        return false
      }
    },
  }
}
