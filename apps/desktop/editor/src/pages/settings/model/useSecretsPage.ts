/**
 * The state and actions of the secrets page: the sections by group, the drafts of new secrets and
 * empty groups, the deletion confirmation, and loading the secret states when the page opens.
 */
import { computed, onMounted, ref, type ComputedRef, type Ref } from 'vue'
import {
  getFirstSecretName,
  getNewGroupName,
  getNextSecretName,
  type PluginInfo,
  type Secret,
} from '@/entities/settings'
import {
  createPageSections,
  getDeclarations,
  type Declaration,
  type SecretSection,
} from '../lib/secretSections'
import type { InstallationSecrets } from '../types'
import { useSecretDeletion } from './secretDeletion'
import { useSecretDrafts } from './secretDrafts'

export interface SecretsPageDeps {
  readonly getSecrets: () => InstallationSecrets
  readonly getPlugins: () => readonly PluginInfo[]
}

function createSecretsPageActions(
  deps: SecretsPageDeps,
  drafts: ReturnType<typeof useSecretDrafts>,
  sections: ComputedRef<readonly SecretSection[]>,
  newGroup: Ref<string>,
) {
  function getNamesInGroup(group: string): readonly string[] {
    const section = sections.value.find((one) => one.group === group)
    return [
      ...(section?.secrets.map((one) => one.name) ?? []),
      ...(section?.secrets.length === 0 ? [getFirstSecretName(group)] : []),
      ...drafts.getDrafts(group).map((one) => one.name),
    ]
  }

  return {
    addSecret: (group: string): void => {
      drafts.addDraft(group, getNextSecretName(group, getNamesInGroup(group)))
    },
    addGroup: (): void => {
      newGroup.value = getNewGroupName(sections.value.map((one) => one.group))
      drafts.addEmptyGroup(newGroup.value)
    },
    renameGroup: (section: SecretSection, name: string): void => {
      newGroup.value = ''
      if (section.secrets.length > 0) deps.getSecrets().edit.renameSecretGroup(section.group, name)
      else drafts.renameEmptyGroup(section.group, name)
    },
    createSecret: async (
      group: string,
      name: string,
      value: string,
      draftId?: number,
    ): Promise<boolean> => {
      const id = await deps.getSecrets().edit.createSecret(group, name, value)
      if (id === '') return false
      if (draftId !== undefined) drafts.discardDraft(draftId)
      drafts.removeEmptyGroup(group)
      return true
    },
    toggleDeclaration: (secret: Secret, declaration: Declaration, isAllowed: boolean): void => {
      const { pluginId, name, group } = declaration
      if (isAllowed) deps.getSecrets().access.allowPluginSecret(secret.id, pluginId, name, group)
      else deps.getSecrets().access.denyPluginSecret(secret.id, pluginId, name, group)
    },
    moveSecret: (id: string, beforeId: string): void => {
      deps.getSecrets().edit.moveSecret(id, beforeId)
    },
    renameSecret: (id: string, name: string): void => {
      deps.getSecrets().edit.renameSecret(id, name)
    },
    writeSecret: (id: string, value: string): void => {
      deps.getSecrets().edit.writeSecret(id, value)
    },
  }
}

export function useSecretsPage(deps: SecretsPageDeps) {
  const credentialsPath = computed(() => deps.getSecrets().credentialsPath.value)
  const drafts = useSecretDrafts()
  /** The group whose name is edited as it appears. */
  const newGroup = ref('')
  const sections = computed(() =>
    createPageSections(
      deps.getSecrets().sections.value,
      drafts.emptyGroups.value,
      deps.getSecrets().states.value,
    ),
  )
  const groups = computed(() => sections.value.map((one) => one.group))
  const declarations = computed(() => getDeclarations(deps.getPlugins()))
  const deletion = useSecretDeletion(
    (id) => deps.getSecrets().edit.deleteSecret(id),
    (group) => deps.getSecrets().edit.deleteSecretGroup(group),
    () => deps.getPlugins(),
  )

  onMounted(() => deps.getSecrets().loadStates())

  const actions = createSecretsPageActions(deps, drafts, sections, newGroup)

  return {
    credentialsPath,
    drafts,
    newGroup,
    sections,
    groups,
    declarations,
    deletion,
    actions,
  }
}
