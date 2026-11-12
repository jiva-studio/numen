/**
 * The state of the dialog that adds or edits a connection: the chosen type, what is typed, and the
 * request that saving sends. A value of a secret stays in this state until it is sent once.
 */
import { computed, reactive } from 'vue'
import { runInBackground } from '@numen/ui'
import { formatError } from '@/shared/api/errors'
import type {
  ConnectionChange,
  ConnectionDraft,
  ConnectionFieldInfo,
  ConnectionInfo,
  ConnectionTypeInfo,
  ConnectionValue,
} from '@/entities/connection'
import type { Result } from '@/shared/lib/result'
import {
  createFieldValues,
  createInitialValues,
  getEditableFields,
  getMissingFields,
  getSentFields,
  getSentSecrets,
  getStoredSecretKeys,
  hasPrivateNetworkChoice,
  isStoredKeyLost,
} from '@/entities/connection'
import { WORDS as words } from '../words'

/** What the form is made from and where it saves. */
export interface ConnectionFormInput {
  readonly types: () => readonly ConnectionTypeInfo[]
  readonly connection: () => ConnectionInfo | undefined
  readonly create: (draft: ConnectionDraft) => Promise<Result<ConnectionInfo, string>>
  readonly update: (change: ConnectionChange) => Promise<Result<ConnectionInfo, string>>
  readonly onSaved: () => void
}

/** What is typed in the form. */
export interface ConnectionFormState {
  typeId: string
  name: string
  values: Record<string, ConnectionValue>
  secrets: Record<string, string>
  clearedKeys: readonly string[]
  isPrivateAllowed: boolean
  problem: string
  isSaving: boolean
}

/** What the form shows for the chosen type. */
export interface ConnectionFormView {
  readonly type: ConnectionTypeInfo | undefined
  readonly fields: readonly ConnectionFieldInfo[]
  readonly storedKeys: readonly string[]
  readonly isKeyLost: boolean
  readonly hasPrivateChoice: boolean
}

function createState(input: ConnectionFormInput): ConnectionFormState {
  const stored = input.connection()
  return reactive({
    typeId: stored?.typeId ?? input.types().find((one) => one.isAvailable)?.id ?? '',
    name: stored?.name ?? '',
    values: createInitialValues(input.types(), stored),
    secrets: {},
    clearedKeys: [],
    isPrivateAllowed: stored?.isPrivateNetworkAllowed ?? false,
    problem: '',
    isSaving: false,
  })
}

function createView(input: ConnectionFormInput, state: ConnectionFormState) {
  return computed<ConnectionFormView>(() => {
    const type = input.types().find((one) => one.id === state.typeId)
    if (!type) {
      return { type, fields: [], storedKeys: [], isKeyLost: false, hasPrivateChoice: false }
    }
    const stored = input.connection()
    return {
      type,
      fields: getEditableFields(type),
      storedKeys: getStoredSecretKeys(type, stored, state.values),
      isKeyLost: isStoredKeyLost(type, stored, state.values),
      hasPrivateChoice: hasPrivateNetworkChoice(type),
    }
  })
}

function createActions(
  input: ConnectionFormInput,
  state: ConnectionFormState,
  view: () => ConnectionFormView,
) {
  const chooseType = (id: string): void => {
    state.typeId = id
    const target = input.types().find((one) => one.id === id)
    state.values = target ? createFieldValues(target) : {}
    state.secrets = {}
    state.clearedKeys = []
    state.problem = ''
  }

  const updateValue = (key: string, value: ConnectionValue): void => {
    state.values = { ...state.values, [key]: value }
  }

  const updateSecret = (key: string, value: ConnectionValue): void => {
    state.secrets = { ...state.secrets, [key]: String(value) }
    state.clearedKeys = state.clearedKeys.filter((one) => one !== key)
  }

  const clearSecret = (key: string): void => {
    state.secrets = { ...state.secrets, [key]: '' }
    state.clearedKeys = [...state.clearedKeys, key]
  }

  const send = (target: ConnectionTypeInfo): Promise<Result<ConnectionInfo, string>> => {
    const stored = input.connection()
    const request = {
      name: state.name,
      fields: getSentFields(target, state.values),
      secrets: getSentSecrets(state.secrets, state.clearedKeys),
      isPrivateNetworkAllowed: hasPrivateNetworkChoice(target) && state.isPrivateAllowed,
    }
    if (stored) return input.update({ ...request, id: stored.id, revision: stored.revision })
    return input.create({ ...request, typeId: target.id })
  }

  /** Sends the form. Resolves to whether it was saved; the reason it was not is in `problem`. */
  const save = async (): Promise<boolean> => {
    const { type, storedKeys } = view()
    if (!type || state.isSaving) return false
    const missing = getMissingFields(type, state.values, state.secrets, storedKeys)
    if (missing.length > 0) {
      const titles = type.fields.filter((one) => missing.includes(one.key)).map((one) => one.title)
      state.problem = `${words.connectionRequired} ${titles.join(', ')}`
      return false
    }
    state.isSaving = true
    state.problem = ''
    const answer = await send(type)
    state.isSaving = false
    if (!answer.ok) state.problem = answer.error
    return answer.ok
  }

  /** Sends the form and calls `onSaved` where it was saved. */
  const submit = (): void => {
    runInBackground(
      save().then((isSaved) => {
        if (isSaved) input.onSaved()
      }),
      (error) => {
        state.isSaving = false
        state.problem = formatError(error)
      },
    )
  }

  return { chooseType, updateValue, updateSecret, clearSecret, submit }
}

export function useConnectionForm(input: ConnectionFormInput) {
  const state = createState(input)
  const view = createView(input, state)
  return {
    state,
    view,
    availableTypes: computed(() => input.types().filter((one) => one.isAvailable)),
    isEditing: input.connection() !== undefined,
    actions: createActions(input, state, () => view.value),
  }
}
