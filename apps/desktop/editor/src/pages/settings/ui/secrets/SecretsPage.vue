<script setup lang="ts">
/**
 * The page of secrets: every secret by group, in priority order, with the plugin declarations that
 * may read it. A value is entered here once and is never shown again.
 */
import { computed, nextTick, onMounted, ref, useTemplateRef } from 'vue'
import { Plus } from '@lucide/vue'
import { Button } from '@numen/ui'
import {
  getFirstSecretName,
  getNewGroupName,
  getNextSecretName,
  type Secret,
} from '@/entities/settings'
import {
  createPageSections,
  getDeclarations,
  type Declaration,
  type SecretSection as SectionData,
} from '../../lib/secretSections'
import type { SettingsTabState } from '../../types'
import { useSecretDeletion } from '../../model/secretDeletion'
import { useSecretDrafts } from '../../model/secretDrafts'
import { WORDS as words } from '../../words'
import { ConfirmDialog } from '../confirm-dialog'
import SecretSection from './SecretSection.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ state: SettingsTabState }>()

const emit = defineEmits<{
  (e: 'back'): void
}>()

/* --------------------------------- State ---------------------------------- */
const installation = computed(() => props.state.installation)
const plugins = computed(() => installation.value.plugins?.value ?? [])
const credentialsPath = computed(() => installation.value.credentialsPath?.value ?? '')
const drafts = useSecretDrafts()
/** The group whose name is edited as it appears. */
const newGroup = ref('')
const addGroupButton = useTemplateRef<InstanceType<typeof Button>>('addGroupButton')
const sections = computed(() =>
  createPageSections(
    installation.value.secretSections?.value ?? [],
    drafts.emptyGroups.value,
    installation.value.secretStates?.value ?? new Map(),
  ),
)
const groups = computed(() => sections.value.map((one) => one.group))
const declarations = computed(() => getDeclarations(plugins.value))
const deletion = useSecretDeletion(
  (id) => installation.value.deleteSecret(id),
  (group) => installation.value.deleteSecretGroup(group),
  () => plugins.value,
)

onMounted(() => installation.value.loadSecretStates?.())

/* -------------------------------- Handlers -------------------------------- */
function onAdd(group: string): void {
  drafts.addDraft(group, getNextSecretName(group, getNamesInGroup(group)))
}

function onAddGroup(): void {
  newGroup.value = getNewGroupName(groups.value)
  drafts.addEmptyGroup(newGroup.value)
}

function onRenameGroup(section: SectionData, name: string): void {
  newGroup.value = ''
  if (section.secrets.length > 0) installation.value.renameSecretGroup(section.group, name)
  else drafts.renameEmptyGroup(section.group, name)
}

async function onCreate(
  group: string,
  name: string,
  value: string,
  draftId?: number,
): Promise<boolean> {
  const id = await installation.value.createSecret(group, name, value)
  if (id === '') return false
  if (draftId !== undefined) drafts.discardDraft(draftId)
  drafts.removeEmptyGroup(group)
  return true
}

function onToggle(secret: Secret, declaration: Declaration, isAllowed: boolean): void {
  const { pluginId, name, group } = declaration
  if (isAllowed) installation.value.allowPluginSecret(secret.id, pluginId, name, group)
  else installation.value.denyPluginSecret(secret.id, pluginId, name, group)
}

function onAskDelete(secret: Secret, opener: HTMLElement): void {
  deletion.ask({ kind: 'secret', secret }, opener)
}

async function onAskDeleteGroup(section: SectionData, opener: HTMLElement): Promise<void> {
  if (section.secrets.length > 0) {
    deletion.ask({ kind: 'group', section }, opener)
    return
  }
  drafts.removeEmptyGroup(section.group)
  await nextTick()
  ;(addGroupButton.value?.$el as HTMLElement | undefined)?.focus()
}

/* -------------------------------- Helpers --------------------------------- */
function getNamesInGroup(group: string): readonly string[] {
  const section = sections.value.find((one) => one.group === group)
  return [
    ...(section?.secrets.map((one) => one.name) ?? []),
    ...(section?.secrets.length === 0 ? [getFirstSecretName(group)] : []),
    ...drafts.getDrafts(group).map((one) => one.name),
  ]
}
</script>

<template>
  <div class="secrets-page">
    <Button variant="ghost" size="small" class="secrets-page__back" @click="emit('back')">
      &larr; {{ words.backToPlugins }}
    </Button>

    <p v-if="credentialsPath !== ''" class="secrets-page__where">
      {{ words.secretsFile(credentialsPath) }}
    </p>

    <SecretSection
      v-for="section in sections"
      :key="section.group"
      :section="section"
      :groups="groups"
      :drafts="drafts.getDrafts(section.group)"
      :declarations="declarations"
      :is-new="section.group === newGroup"
      :create="(name, value, draftId) => onCreate(section.group, name, value, draftId)"
      @add="onAdd(section.group)"
      @rename-group="(name) => onRenameGroup(section, name)"
      @discard="drafts.discardDraft"
      @move="(id, beforeId) => installation.moveSecret(id, beforeId)"
      @rename="(id, name) => installation.renameSecret(id, name)"
      @write="(id, value) => installation.writeSecret(id, value)"
      @delete="onAskDelete"
      @delete-group="(opener) => onAskDeleteGroup(section, opener)"
      @toggle="onToggle"
    />

    <Button
      ref="addGroupButton"
      variant="ghost"
      size="small"
      class="secrets-page__add"
      @click="onAddGroup"
    >
      <Plus :size="14" aria-hidden="true" />
      {{ words.secretAddGroup }}
    </Button>

    <ConfirmDialog
      v-if="deletion.target.value"
      tone="danger"
      :text="deletion.text.value"
      :label="deletion.isGroup.value ? words.secretGroupDelete : words.secretDelete"
      :confirm-text="words.secretDeleteConfirm"
      :cancel-text="words.secretDeleteCancel"
      @confirm="deletion.confirm"
      @cancel="deletion.cancel"
    />
  </div>
</template>

<style scoped>
.secrets-page {
  display: flex;
  flex-direction: column;
  inline-size: 100%;
  max-inline-size: var(--settings-measure);
  margin-inline: auto;
}

.secrets-page__back {
  --margin-end: 0.75rem;
  align-self: flex-start;
  margin-block-end: var(--margin-end);
}

.secrets-page__where {
  margin: 0 0 var(--settings-apart);
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  overflow-wrap: anywhere;
}

.secrets-page__add {
  align-self: flex-start;
}
</style>
