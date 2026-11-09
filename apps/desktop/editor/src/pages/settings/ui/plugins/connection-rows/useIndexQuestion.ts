import { ref } from 'vue'
import { runInBackground } from '@numen/ui'
import type { StoredRow } from '@/entities/connection'

/** What the question about indexing the vault again needs from the page of the rows. */
export interface IndexQuestionSource {
  readonly countNotes: () => Promise<number | undefined>
  readonly getText: (rows: readonly StoredRow[], count: number | undefined) => string
  readonly onConfirm: (rows: readonly StoredRow[]) => void
  readonly onCancel: () => void
}

/**
 * The question a change of the rows of embeddings asks before it is written. `ask` shows it with the
 * number of notes once that is counted, `confirm` passes the rows on, and `cancel` drops them.
 */
export function useIndexQuestion(source: IndexQuestionSource) {
  const question = ref<{ readonly rows: readonly StoredRow[]; readonly text: string } | undefined>()

  const show = async (rows: readonly StoredRow[]): Promise<void> => {
    question.value = { rows, text: source.getText(rows, await source.countNotes()) }
  }

  return {
    question,
    ask: (rows: readonly StoredRow[]): void => {
      runInBackground(show(rows), () => undefined)
    },
    confirm: (): void => {
      const rows = question.value?.rows
      question.value = undefined
      if (rows) source.onConfirm(rows)
    },
    cancel: (): void => {
      question.value = undefined
      source.onCancel()
    },
  }
}
