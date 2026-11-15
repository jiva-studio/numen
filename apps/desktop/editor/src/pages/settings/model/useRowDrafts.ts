import { ref, watch } from 'vue'
import type { JsonValue } from '@/shared/api/json'
import {
  getCompleteRows,
  getRowKey,
  moveRow,
  readRows,
  type ConnectionFieldInfo,
  type RowDraft,
  type StoredRow,
} from '@/entities/connection'

/**
 * Where the rows come from and where a change of them goes. A row is complete once each required row field
 * of the type of its connection has a value.
 */
export interface RowDraftsSource {
  readonly getValue: () => JsonValue | undefined
  readonly getRowFields: (connectionId: string) => readonly ConnectionFieldInfo[]
  readonly onChange: (rows: readonly StoredRow[]) => void
}

function isSameRows(first: readonly StoredRow[], second: readonly StoredRow[]): boolean {
  return (
    first.length === second.length &&
    first.every((one, index) => {
      const other = second[index]
      return other !== undefined && getRowKey(one) === getRowKey(other)
    })
  )
}

/**
 * The rows on the page: the rows of the value in force, and the rows added and not yet complete. A
 * change writes only the rows that name a connection and a value for each required row field. A value that differs from the
 * complete rows on the page replaces them.
 */
export function useRowDrafts(source: RowDraftsSource) {
  let keySeed = 0
  const createKey = (): string => `row-${++keySeed}`
  const createDrafts = (rows: readonly StoredRow[]): readonly RowDraft[] =>
    rows.map((one) => ({ ...one, key: createKey() }))

  const drafts = ref<readonly RowDraft[]>(createDrafts(readRows(source.getValue())))

  const write = (): void => {
    const complete = getCompleteRows(drafts.value, source.getRowFields)
    if (isSameRows(complete, readRows(source.getValue())) && inFlight.length === 0) return
    inFlight = [...inFlight, complete]
    source.onChange(complete)
  }

  /* The rows written and not yet answered, oldest first. An answer that is one of the older ones is
   * stale: a newer write is on its way, and the rows on the page already hold it. */
  let inFlight: (readonly StoredRow[])[] = []

  watch(source.getValue, (next) => {
    const stored = readRows(next)
    const at = inFlight.findIndex((one) => isSameRows(one, stored))
    if (at >= 0) {
      inFlight = inFlight.slice(at + 1)
      return
    }
    inFlight = []
    if (!isSameRows(stored, getCompleteRows(drafts.value, source.getRowFields)))
      drafts.value = createDrafts(stored)
  })

  return {
    drafts,
    write,
    /**
     * Drops a change that was not confirmed. The rows go back to the stored ones, except that a row
     * with no stored rows behind it stays with its connection, and its row fields are chosen again.
     */
    revert: (): void => {
      inFlight = []
      const stored = readRows(source.getValue())
      drafts.value =
        stored.length > 0
          ? createDrafts(stored)
          : drafts.value.map((one) => ({ ...one, options: {} }))
    },
    add: (): void => {
      drafts.value = [...drafts.value, { key: createKey(), connection: '', options: {} }]
    },
    remove: (key: string): void => {
      drafts.value = drafts.value.filter((one) => one.key !== key)
      write()
    },
    /** Changes a row; the rows are written only once that row is complete. */
    update: (key: string, patch: Partial<StoredRow>): void => {
      drafts.value = drafts.value.map((one) => (one.key === key ? { ...one, ...patch } : one))
      const changed = drafts.value.filter((one) => one.key === key)
      if (getCompleteRows(changed, source.getRowFields).length === 0) return
      write()
    },
    move: (from: number, to: number): void => {
      drafts.value = moveRow(drafts.value, from, to)
      write()
    },
  }
}
