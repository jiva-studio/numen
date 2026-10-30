/**
 * The values being written, by path, and the reads of the file in order.
 *
 * after it are done; a read that answers after a newer one is not taken.
 */
import { shallowRef } from 'vue'
import type { SettingEdit } from '../lib/configuration'

const PATH_JOIN = '\u0000'

/** How long a written value stays shown while the committed settings have not caught up. */
const EXPIRY_MS = 20_000

interface PendingEntry {
  token: number
  value: unknown
  /** Set once the write has finished and been read back. */
  releasedAt?: number
}

export const getPathKey = (path: readonly string[]): string => path.join(PATH_JOIN)

const isSameValue = (left: unknown, right: unknown): boolean =>
  JSON.stringify(left) === JSON.stringify(right)

export function createPendingWrites() {
  const values = shallowRef<ReadonlyMap<string, PendingEntry>>(new Map())
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

    /**
     * The edits are written and read back. An edit the read already confirms is
     * dropped; any other stays until a later read confirms it or it expires.
     * Edits a newer write replaced are left alone.
     */
    release: (
      edits: readonly SettingEdit[],
      token: number,
      getCommitted: (path: readonly string[]) => unknown,
    ): void => {
      const remaining = new Map(values.value)
      const now = Date.now()
      for (const edit of edits) {
        const key = getPathKey(edit.at)
        const entry = remaining.get(key)
        if (entry?.token !== token) continue
        if (isSameValue(getCommitted(edit.at), entry.value)) remaining.delete(key)
        else remaining.set(key, { ...entry, releasedAt: now })
      }
      values.value = remaining
    },

    /** Drops the released edits that the committed settings confirm or that have expired. */
    prune: (getCommitted: (path: readonly string[]) => unknown): void => {
      const now = Date.now()
      let next: Map<string, PendingEntry> | undefined
      for (const [key, entry] of values.value) {
        if (entry.releasedAt === undefined) continue
        const isConfirmed = isSameValue(getCommitted(key.split(PATH_JOIN)), entry.value)
        if (!isConfirmed && now - entry.releasedAt < EXPIRY_MS) continue
        next ??= new Map(values.value)
        next.delete(key)
      }
      if (next) values.value = next
    },

    /** Drops the edits of a write that failed. */
    discard: (edits: readonly SettingEdit[], token: number): void => {
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
