/**
 * The state of the page of connections: the connections and types it lists, the dialog that adds
 * or edits one, and the question asked before one is deleted. The page reads the connections when
 * it opens. The dialog shows the connection it edits as the list last read it, so a save sends the
 * revision the core holds; a connection gone from the list closes the dialog.
 */
import { computed, onMounted, ref, type Ref } from 'vue'
import { runInBackground } from '@numen/ui'
import { formatError } from '@/shared/api/errors'
import {
  getUsedByNames,
  hasAvailableType,
  hasUnavailableConnection,
  type ConnectionInfo,
} from '@/entities/connection'
import type { ConnectionEdits, InstallationConnections } from '../types'
import { WORDS as words } from '../words'

function useConnectionDialog(connections: Readonly<Ref<readonly ConnectionInfo[]>>) {
  const editingId = ref<string | null>(null)
  const isAdding = ref(false)
  const editing = computed(
    () => connections.value.find((one) => one.id === editingId.value) ?? null,
  )
  return {
    editing,
    isOpen: computed(() => isAdding.value || editing.value !== null),
    openAdd: (): void => {
      isAdding.value = true
    },
    openEdit: (connection: ConnectionInfo): void => {
      editingId.value = connection.id
    },
    close: (): void => {
      isAdding.value = false
      editingId.value = null
    },
  }
}

function getDeleteText(connection: ConnectionInfo, problem: string): string {
  const question = words.connectionDeleteAsking(
    connection.name,
    connection.secretKeysSet.length > 0,
  )
  const usedBy = getUsedByNames(connection.uses)
  const text = usedBy === '' ? question : `${question} ${words.connectionUsedByDeleted(usedBy)}`
  return problem === '' ? text : `${text} ${words.connectionDeleteFailed} ${problem}`
}

function useConnectionDeletion(getConnections: () => InstallationConnections) {
  const target = ref<ConnectionInfo | null>(null)
  const problem = ref('')

  const confirmNow = async (): Promise<boolean> => {
    const asked = target.value
    if (!asked) return false
    const answer = await getConnections().edits.delete(asked.id)
    if (!answer.ok) {
      problem.value = answer.error
      return false
    }
    target.value = null
    return true
  }

  return {
    target,
    text: computed(() => (target.value ? getDeleteText(target.value, problem.value) : '')),
    ask: (connection: ConnectionInfo): void => {
      target.value = connection
      problem.value = ''
    },
    /** Deletes the connection asked about, then calls `onDeleted`. */
    confirm: (onDeleted: () => void): void => {
      runInBackground(
        confirmNow().then((isDeleted) => {
          if (isDeleted) onDeleted()
        }),
        (error) => {
          problem.value = formatError(error)
        },
      )
    },
    cancel: (): void => {
      target.value = null
    },
  }
}

export function useConnectionsPage(getConnections: () => InstallationConnections) {
  const connections = computed(() => getConnections().list.value)
  const types = computed(() => getConnections().types.value)

  onMounted(() => getConnections().load())

  return {
    connections,
    types,
    hasType: computed(() => hasAvailableType(types.value)),
    hasUnavailable: computed(() => hasUnavailableConnection(connections.value)),
    dialog: useConnectionDialog(connections),
    deletion: useConnectionDeletion(getConnections),
    create: (draft: Parameters<ConnectionEdits['create']>[0]) =>
      getConnections().edits.create(draft),
    update: (change: Parameters<ConnectionEdits['update']>[0]) =>
      getConnections().edits.update(change),
  }
}
