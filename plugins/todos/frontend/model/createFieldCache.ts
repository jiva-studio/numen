import type { TodosApi } from '../api/wire'
import type { ListFieldsResponse } from '../types'

export interface FieldCache {
  get(): ListFieldsResponse
  refresh(): Promise<void>
}

export function createFieldCache(api: Pick<TodosApi, 'listFields'>): FieldCache {
  let fields: ListFieldsResponse = { projects: [], contexts: [], tags: [] }
  let latest = 0
  return {
    get: () => fields,
    refresh: async () => {
      latest += 1
      const asked = latest
      const result = await api.listFields()
      if (result.ok && asked === latest) fields = result.value
    },
  }
}
