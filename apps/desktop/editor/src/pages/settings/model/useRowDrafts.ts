import { ref, watch } from 'vue'
import type { JsonValue } from '@/shared/api/json'
import {
  getCompleteRows,
  moveRow,
  readRows,
  type RowDraft,
  type StoredRow,
} from '@/entities/connection'

/** Where the rows come from and where a change of them goes. */
export interface RowDraftsSource {
  readonly getValue: () => JsonValue | undefined
  readonly onChange: (rows: readonly StoredRow[]) => void
}

function sameRows(first: readonly StoredRow[], second: readonly StoredRow[]): boolean {
  return (
    first.length === second.length &&
    first.every((one, index) => {
      const other = second[index]
      return (
        other !== undefined &&
        one.connection === other.connection &&
        one.model === other.model &&
        one.reasoning === other.reasoning
      )
    })
  )
}

/**
 * The rows on the page: the rows of the value in force, and the rows added and not yet complete. A
 * change writes only the rows that name a connection and a model. A value that differs from the
 * complete rows on the page replaces them.
 */
export function useRowDrafts(source: RowDraftsSource) {
  let keySeed = 0
  const createKey = (): string => `row-${++keySeed}`
  const createDrafts = (rows: readonly StoredRow[]): readonly RowDraft[] =>
    rows.map((one) => ({ ...one, key: createKey() }))

  const drafts = ref<readonly RowDraft[]>(createDrafts(readRows(source.getValue())))

  const write = (): void => {
    const complete = getCompleteRows(drafts.value)
    if (!sameRows(complete, readRows(source.getValue()))) source.onChange(complete)
  }

  watch(source.getValue, (next) => {
    const stored = readRows(next)
    if (!sameRows(stored, getCompleteRows(drafts.value))) drafts.value = createDrafts(stored)
  })

  return {
    drafts,
    write,
    /**
     * Drops a change that was not confirmed. The rows go back to the stored ones, except that a row
     * with no stored rows behind it stays with its connection, and its model is chosen again.
     */
    revert: (): void => {
      const stored = readRows(source.getValue())
      drafts.value =
        stored.length > 0
          ? createDrafts(stored)
          : drafts.value.map((one) => ({ ...one, model: '', reasoning: '' }))
    },
    add: (): void => {
      drafts.value = [
        ...drafts.value,
        { key: createKey(), connection: '', model: '', reasoning: '' },
      ]
    },
    remove: (key: string): void => {
      drafts.value = drafts.value.filter((one) => one.key !== key)
      write()
    },
    update: (key: string, patch: Partial<StoredRow>): void => {
      drafts.value = drafts.value.map((one) => (one.key === key ? { ...one, ...patch } : one))
      write()
    },
    move: (from: number, to: number): void => {
      drafts.value = moveRow(drafts.value, from, to)
      write()
    },
  }
}
