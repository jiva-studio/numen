import { browserClock, type Clock } from '@numen/ui'
import type { TodosApi } from '../api/wire'
import type { ListFieldsResponse } from '../types'

export interface FieldCache {
  get(): ListFieldsResponse
  refresh(): Promise<void>
  ensure(): Promise<void>
}

const RETRY_AFTER_MS = 2000

export function createFieldCache(
  api: Pick<TodosApi, 'listFields'>,
  clock: Pick<Clock, 'now'> = browserClock,
): FieldCache {
  let fields: ListFieldsResponse = { projects: [], contexts: [], tags: [] }
  let latest = 0
  let askedAt = Number.NEGATIVE_INFINITY
  let running: Promise<void> | null = null
  const refresh = async (): Promise<void> => {
    latest += 1
    const asked = latest
    askedAt = clock.now()
    const result = await api.listFields()
    if (result.ok && asked === latest) fields = result.value
  }
  const isEmpty = (): boolean =>
    fields.projects.length + fields.contexts.length + fields.tags.length === 0
  return {
    get: () => fields,
    refresh,
    ensure: () => {
      if (!isEmpty() || clock.now() - askedAt < RETRY_AFTER_MS) return running ?? Promise.resolve()
      running ??= refresh().finally(() => {
        running = null
      })
      return running
    },
  }
}
