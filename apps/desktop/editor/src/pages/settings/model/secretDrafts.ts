/**
 * What the person has added on the page of secrets and the core does not hold yet: secrets without
 * a value, each with an identifier of its own, and groups without a secret.
 */
import { ref } from 'vue'

/** A secret the person added and has not given a value yet. */
export interface SecretDraft {
  readonly id: number
  readonly group: string
  readonly name: string
}

export function useSecretDrafts() {
  const drafts = ref<readonly SecretDraft[]>([])
  const emptyGroups = ref<readonly string[]>([])
  let lastId = 0

  const getDrafts = (group: string): readonly SecretDraft[] =>
    drafts.value.filter((one) => one.group === group)

  const addDraft = (group: string, name: string): void => {
    lastId++
    drafts.value = [...drafts.value, { id: lastId, group, name }]
  }

  const discardDraft = (id: number): void => {
    drafts.value = drafts.value.filter((one) => one.id !== id)
  }

  const addEmptyGroup = (group: string): void => {
    emptyGroups.value = [...emptyGroups.value, group]
  }

  const renameEmptyGroup = (group: string, name: string): void => {
    emptyGroups.value = emptyGroups.value.map((one) => (one === group ? name : one))
  }

  const removeEmptyGroup = (group: string): void => {
    emptyGroups.value = emptyGroups.value.filter((one) => one !== group)
  }

  return {
    emptyGroups,
    getDrafts,
    addDraft,
    discardDraft,
    addEmptyGroup,
    renameEmptyGroup,
    removeEmptyGroup,
  }
}
