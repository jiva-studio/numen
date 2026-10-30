/**
 * The values being written, by path, and the reads of the file in order.
 *
 * after it are done; a read that answers after a newer one is not taken.
 */
import { shallowRef } from 'vue'
import type { SettingEdit } from '../lib/configuration'

const PATH_JOIN = '\u0000'

export const getPathKey = (path: readonly string[]): string => path.join(PATH_JOIN)

export function createPendingWrites() {
  const values = shallowRef<ReadonlyMap<string, { token: number; value: unknown }>>(new Map())
  let lastToken = 0

  let lastRead = 0
  let lastTaken = 0

  return {
    /** The value being written at a path, if any. */
    getPending: (path: readonly string[]) => values.value.get(getPathKey(path)),

    /** The edits begin to be written; returns the token that releases them. */
    add: (edits: readonly SettingEdit[]): number => {
      const token = ++lastToken
      const next = new Map(values.value)
      for (const edit of edits) {
        const key = getPathKey(edit.at)
        try {
          next.set(key, { token, value: JSON.parse(edit.value) })
        } catch {
          next.delete(key)
        }
      }
      values.value = next
      return token
    },

    /** The edits are written and read back; only those no newer write replaced are dropped. */
    release: (edits: readonly SettingEdit[], token: number): void => {
      const remaining = new Map(values.value)
      for (const edit of edits) {
        const key = getPathKey(edit.at)
        if (remaining.get(key)?.token === token) remaining.delete(key)
      }
      values.value = remaining
    },

    /** A read begins; returns its place in the order of reads. */
    beginRead: (): number => ++lastRead,

    /** Whether the answer of read `place` is newer than every answer taken. */
    takeRead: (place: number): boolean => {
      if (place < lastTaken) return false
      lastTaken = place
      return true
    },
  }
}
