import { computed, ref, watch } from 'vue'
import {
  listModelsForOperation,
  type ModelInfo,
  type ModelListFailure,
  type ReasoningLevelInfo,
  type RowDraft,
  type StoredRow,
} from '@/entities/connection'
import { formatError } from '@/shared/api/errors'
import type { Result } from '@/shared/lib/result'
import { WORDS as words } from '../words'
import type { ListModels, ListReasoningLevels, SelectChoice } from '../types'

/** What a row reads from the provider of its connection and how it answers. */
export interface RowChoicesSource {
  readonly getRow: () => RowDraft
  readonly getOperation: () => string
  readonly getListModels: () => ListModels | undefined
  readonly getListReasoningLevels: () => ListReasoningLevels | undefined
  readonly update: (patch: Partial<StoredRow>) => void
}

async function readAnswer<T>(
  pending: Promise<Result<T, ModelListFailure>>,
): Promise<Result<T, string>> {
  try {
    const answer = await pending
    return answer.ok ? answer : { ok: false, error: answer.error.detail }
  } catch (caughtError) {
    return { ok: false, error: formatError(caughtError) }
  }
}

/**
 * The models and the reasoning levels of the connection a row names, asked from its provider
 * whenever the connection or the model changes. Choosing a connection for an operation other than
 * embedding gives the row the first of its models; embedding waits for the person to choose one,
 * because a choice there indexes the vault again. An answer that comes after the row changed again is dropped.
 */
export function useRowChoices(source: RowChoicesSource) {
  const models = ref<readonly ModelInfo[]>([])
  const levels = ref<readonly ReasoningLevelInfo[]>([])
  const problem = ref('')
  const levelsProblem = ref('')
  let modelsRequest = 0
  let levelsRequest = 0

  const modelChoices = computed<readonly SelectChoice[]>(() =>
    listModelsForOperation(models.value, source.getOperation()).map((one) => ({
      id: one.id,
      text: one.title,
      ...(one.title === one.id ? {} : { detail: one.id }),
    })),
  )

  const levelChoices = computed<readonly SelectChoice[]>(() => [
    { id: '', text: words.connectionRowsDefault },
    ...levels.value.map((one) => ({ id: one.value, text: one.title || one.value })),
  ])

  watch(
    () => source.getRow().connection,
    async (connection) => {
      const request = ++modelsRequest
      const listModels = source.getListModels()
      models.value = []
      problem.value = ''
      if (connection === '' || !listModels) return
      const answer = await readAnswer(listModels(connection, source.getOperation()))
      if (request !== modelsRequest) return
      if (!answer.ok) {
        problem.value = answer.error
        return
      }
      models.value = answer.value
      const first = listModelsForOperation(answer.value, source.getOperation())[0]
      if (source.getOperation() !== 'embed' && source.getRow().model === '' && first) {
        source.update({ model: first.id })
      }
    },
    { immediate: true },
  )

  watch(
    () => [source.getRow().connection, source.getRow().model] as const,
    async ([connection, model]) => {
      const request = ++levelsRequest
      const listLevels = source.getListReasoningLevels()
      levels.value = []
      levelsProblem.value = ''
      if (connection === '' || model === '' || !listLevels) return
      const answer = await readAnswer(listLevels(connection, model))
      if (request !== levelsRequest) return
      if (answer.ok) levels.value = answer.value
      else levelsProblem.value = answer.error
    },
    { immediate: true },
  )

  return { modelChoices, levelChoices, problem, levelsProblem }
}
