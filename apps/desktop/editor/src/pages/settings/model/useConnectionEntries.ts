/**
 * The connections a row can choose from, for one operation: each with whether it can serve the
 * operation and why not. The connections are read when the rows open.
 */
import { computed, onMounted } from 'vue'
import {
  getConnectionUsability,
  sortByUsability,
  type ConnectionInfo,
  type ConnectionTypeInfo,
} from '@/entities/connection'
import type { InstallationConnections, SelectChoice } from '../types'
import { WORDS as words } from '../words'

const USABILITY_WORDS = {
  unavailable: words.connectionUnavailable,
  incomplete: words.connectionIncomplete,
  cannot: words.connectionCannot,
}

export function useConnectionEntries(
  getConnections: () => InstallationConnections,
  getOperation: () => string,
) {
  const stored = computed<readonly ConnectionInfo[]>(() => getConnections().list.value)
  const types = computed<readonly ConnectionTypeInfo[]>(() => getConnections().types.value)

  const entries = computed(() =>
    sortByUsability(
      stored.value.map((connection) => {
        const type = types.value.find((one) => one.id === connection.typeId)
        return {
          connection,
          type,
          ...getConnectionUsability(connection, type, getOperation(), USABILITY_WORDS),
        }
      }),
    ),
  )

  const choices = computed<readonly SelectChoice[]>(() =>
    entries.value.map((one) => ({
      id: one.connection.id,
      text: one.connection.name,
      detail: one.isUsable ? (one.type?.title ?? '') : one.reason,
      ...(one.isUsable ? {} : { isDisabled: true }),
    })),
  )

  onMounted(() => getConnections().load())

  return {
    stored,
    choices,
    getReason: (connectionId: string): string =>
      entries.value.find((one) => one.connection.id === connectionId)?.reason ?? '',
  }
}
