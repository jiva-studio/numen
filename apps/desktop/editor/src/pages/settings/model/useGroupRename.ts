import { onMounted, ref } from 'vue'
import {
  MAX_NAME_LENGTH,
  isGroupNameTaken,
  isNameTooLong,
  trimName,
} from '@/entities/settings'
import { WORDS as words } from '../words'
import { runAfterRender } from './afterRender'

export interface GroupRenameDeps {
  readonly getGroup: () => string
  readonly getGroups: () => readonly string[]
  readonly isNew: () => boolean
  readonly focusName: () => void
  readonly emitRename: (name: string) => void
}

type NameProblem = '' | 'long' | 'taken'

/**
 * The rename of a group in place: the name typed, why it is refused, and the commit that emits the
 * new name. A group the person just made opens in rename when the header is mounted.
 */
export function useGroupRename(deps: GroupRenameDeps) {
  const isRenaming = ref(false)
  const draftName = ref('')
  const nameProblem = ref<NameProblem>('')

  const getNameProblem = (name: string): NameProblem => {
    if (isNameTooLong(name)) return 'long'
    return isGroupNameTaken(deps.getGroups(), deps.getGroup(), name) ? 'taken' : ''
  }

  const startRename = (): void => {
    draftName.value = deps.getGroup()
    nameProblem.value = ''
    isRenaming.value = true
    runAfterRender(deps.focusName)
  }

  const commitRename = (): void => {
    if (!isRenaming.value) return
    const name = trimName(draftName.value)
    nameProblem.value = getNameProblem(name)
    if (nameProblem.value !== '') return
    isRenaming.value = false
    if (name !== '' && name !== deps.getGroup()) deps.emitRename(name)
  }

  const cancelRename = (): void => {
    isRenaming.value = false
    nameProblem.value = ''
  }

  const getNameProblemText = (): string =>
    nameProblem.value === 'long'
      ? words.secretGroupTooLong(MAX_NAME_LENGTH)
      : words.secretGroupTaken(trimName(draftName.value))

  onMounted(() => {
    if (deps.isNew()) startRename()
  })

  return {
    isRenaming,
    draftName,
    nameProblem,
    startRename,
    commitRename,
    cancelRename,
    getNameProblemText,
  }
}
