<script setup lang="ts">
/**
 * The page of secrets: every secret by group, in priority order, with the plugin declarations that
 * may read it. A value is entered here once and is never shown again.
 */
import { computed, ref } from 'vue'
import { Plus } from '@lucide/vue'
import { Button, ConfirmDialog } from '@numen/ui'
import type { Secret } from '@/entities/settings'
import {
  createSecretSections,
  getDeclarations,
  getNewGroupName,
  getFirstSecretName,
  getNextSecretName,
  type Declaration,
  type SecretSection as SectionData,
} from '../../lib/secretSections'
import type { SettingsTabState } from '../../types'
import { useSecretDeletion } from '../../model/secretDeletion'
import { WORDS as words } from '../../words'
import SecretSection from './SecretSection.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ state: SettingsTabState }>()

const emit = defineEmits<{
  (e: 'back'): void
}>()

/** A secret the person added and has not given a value yet. */
interface Draft {
  readonly group: string
  readonly name: string
}

/* --------------------------------- State ---------------------------------- */
const installation = computed(() => props.state.installation)
const plugins = computed(() => installation.value.plugins?.value ?? [])
const credentialsPath = computed(() => installation.value.credentialsPath?.value ?? '')
/** The groups the person made that hold no secret yet; a group exists on the core through its secrets. */
const emptyGroups = ref<readonly string[]>([])
/** The group whose name is edited as it appears. */
const newGroup = ref('')
const sections = computed(() =>
  createSecretSections(plugins.value, installation.value.secrets?.value ?? [], emptyGroups.value),
)
const declarations = computed(() => getDeclarations(plugins.value))
const drafts = ref<readonly Draft[]>([])
const deletion = useSecretDeletion(
  (id) => installation.value.deleteSecret(id),
  () => plugins.value,
)

/* -------------------------------- Handlers -------------------------------- */
function onAdd(group: string): void {
  drafts.value = [
    ...drafts.value,
    { group, name: getNextSecretName(group, getNamesInGroup(group)) },
  ]
}

function onAddGroup(): void {
  newGroup.value = getNewGroupName(sections.value.map((one) => one.group))
  emptyGroups.value = [...emptyGroups.value, newGroup.value]
}

function onRenameGroup(section: SectionData, name: string): void {
  newGroup.value = ''
  if (section.secrets.length > 0) installation.value.renameSecretGroup(section.group, name)
  else emptyGroups.value = emptyGroups.value.map((one) => (one === section.group ? name : one))
}

function onDiscard(group: string, name: string): void {
  drafts.value = drafts.value.filter((one) => one.group !== group || one.name !== name)
}

async function onCreate(group: string, name: string, value: string): Promise<boolean> {
  const id = await installation.value.createSecret(group, name, value)
  if (id !== '') {
    onDiscard(group, name)
    emptyGroups.value = emptyGroups.value.filter((one) => one !== group)
  }
  return id !== ''
}

function onToggle(secret: Secret, declaration: Declaration, isAllowed: boolean): void {
  const { pluginId, name, group } = declaration
  if (isAllowed) installation.value.allowPluginSecret(secret.id, pluginId, name, group)
  else installation.value.denyPluginSecret(secret.id, pluginId, name, group)
}

function onAskDelete(secret: Secret, opener: HTMLElement): void {
  deletion.ask({ kind: 'secret', secret }, opener)
}

function onAskDeleteGroup(section: SectionData, opener: HTMLElement): void {
  if (section.secrets.length === 0) {
    emptyGroups.value = emptyGroups.value.filter((one) => one !== section.group)
    return
  }
  deletion.ask({ kind: 'group', section }, opener)
}

/* -------------------------------- Helpers --------------------------------- */
function getDraftNames(group: string): readonly string[] {
  return drafts.value.filter((one) => one.group === group).map((one) => one.name)
}

/** The names a new secret of the group must not take: those in use and the Default of an empty asked group. */
function getNamesInGroup(group: string): readonly string[] {
  const section = sections.value.find((one) => one.group === group)
  return [
    ...(section?.secrets.map((one) => one.name) ?? []),
    ...(section?.isAsked && section.secrets.length === 0 ? [getFirstSecretName(group)] : []),
    ...getDraftNames(group),
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
      :drafts="getDraftNames(section.group)"
      :declarations="declarations"
      :is-new="section.group === newGroup"
      @add="onAdd(section.group)"
      @rename-group="(name) => onRenameGroup(section, name)"
      :create="(name, value) => onCreate(section.group, name, value)"
      @discard="(name) => onDiscard(section.group, name)"
      @move="(id, beforeId) => installation.moveSecret(id, beforeId)"
      @rename="(id, name) => installation.renameSecret(id, name)"
      @write="(id, value) => installation.writeSecret(id, value)"
      @delete="onAskDelete"
      @delete-group="(opener) => onAskDeleteGroup(section, opener)"
      @toggle="onToggle"
    />

    <Button variant="ghost" size="small" class="secrets-page__add" @click="onAddGroup">
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
  align-self: flex-start;
  margin-block-end: 0.75rem;
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
