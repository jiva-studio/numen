<script setup lang="ts">
/**
 * The page of secrets: every secret by group, in priority order, with the plugin declarations that
 * may read it. A value is entered here once and is never shown again.
 */
import { computed, ref } from 'vue'
import { Plus } from '@lucide/vue'
import { Button } from '@numen/ui'
import type { Secret } from '@/entities/settings'
import {
  createSecretSections,
  getDeclarations,
  getNextSecretName,
  type Declaration,
  type SecretSection as SectionData,
} from '../../lib/secretSections'
import type { SettingsTabState } from '../../types'
import { useSecretDeletion } from '../../model/secretDeletion'
import { MAX_GROUP_LENGTH, trimGroupName } from '../../lib/trimGroupName'
import { WORDS as words } from '../../words'
import SecretDeleteDialog from './SecretDeleteDialog.vue'
import SecretDraftRow from './SecretDraftRow.vue'
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
const sections = computed(() =>
  createSecretSections(plugins.value, installation.value.secrets?.value ?? []),
)
const declarations = computed(() => getDeclarations(plugins.value))
const drafts = ref<readonly Draft[]>([])
/** The group name typed for a secret in a group no section shows yet, or null while none is added. */
const newGroup = ref<string | null>(null)
const newGroupName = computed(() => trimGroupName(newGroup.value ?? ''))
const isNewGroupTooLong = computed(() => [...newGroupName.value].length > MAX_GROUP_LENGTH)
const newName = computed(() => getNextSecretName(getNamesInGroup(newGroupName.value)))
const deletion = useSecretDeletion(
  (id) => installation.value.deleteSecret(id),
  () => plugins.value,
)

/* -------------------------------- Handlers -------------------------------- */
function onAdd(group: string): void {
  drafts.value = [...drafts.value, { group, name: getNextSecretName(getNamesInGroup(group)) }]
}

function onOpenNewGroup(): void {
  newGroup.value = ''
}

async function onCreateInNewGroup(value: string): Promise<boolean> {
  if (isNewGroupTooLong.value) return false
  const id = await installation.value.createSecret(newGroupName.value, newName.value, value)
  if (id === '') return false
  newGroup.value = null
  return true
}

function onDiscard(group: string, name: string): void {
  drafts.value = drafts.value.filter((one) => one.group !== group || one.name !== name)
}

async function onCreate(group: string, name: string, value: string): Promise<boolean> {
  const id = await installation.value.createSecret(group, name, value)
  if (id !== '') onDiscard(group, name)
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
    ...(section?.isAsked && section.secrets.length === 0 ? [words.secretDefault] : []),
    ...getDraftNames(group),
  ]
}
</script>

<template>
  <div class="secrets-page">
    <Button variant="ghost" size="small" class="secrets-page__back" @click="emit('back')">
      &larr; {{ words.backToPlugins }}
    </Button>

    <h1 class="secrets-page__title">{{ words.secretsTitle }}</h1>
    <p class="secrets-page__about">{{ words.secretsAbout }}</p>
    <p class="secrets-page__where">
      {{ credentialsPath === '' ? words.secretsKeychain : words.secretsFile(credentialsPath) }}
    </p>

    <SecretSection
      v-for="section in sections"
      :key="section.group"
      :section="section"
      :drafts="getDraftNames(section.group)"
      :declarations="declarations"
      @add="onAdd(section.group)"
      @rename-group="(name) => installation.renameSecretGroup(section.group, name)"
      :create="(name, value) => onCreate(section.group, name, value)"
      @discard="(name) => onDiscard(section.group, name)"
      @move="(id, beforeId) => installation.moveSecret(id, beforeId)"
      @rename="(id, name) => installation.renameSecret(id, name)"
      @write="(id, value) => installation.writeSecret(id, value)"
      @delete="onAskDelete"
      @delete-group="(opener) => onAskDeleteGroup(section, opener)"
      @toggle="onToggle"
    />

    <p v-if="sections.length === 0 && newGroup === null" class="secrets-page__empty">
      {{ words.secretsEmpty }}
    </p>

    <div v-if="newGroup !== null" class="secrets-page__new">
      <input
        v-model="newGroup"
        type="text"
        :aria-label="words.secretNewGroup"
        :placeholder="words.secretNewGroupPlaceholder"
        class="secrets-page__new-group"
        @keydown.esc="newGroup = null"
      />
      <p v-if="isNewGroupTooLong" role="alert" class="secrets-page__new-error">
        {{ words.secretGroupTooLong(MAX_GROUP_LENGTH) }}
      </p>
      <ul v-if="newGroupName !== ''" class="secrets-page__new-secrets">
        <SecretDraftRow
          :key="newName"
          :name="newName"
          can-discard
          :submit="onCreateInNewGroup"
          @discard="newGroup = null"
        />
      </ul>
    </div>
    <Button v-else variant="ghost" size="small" class="secrets-page__add" @click="onOpenNewGroup">
      <Plus :size="14" aria-hidden="true" />
      {{ words.secretAdd }}
    </Button>

    <SecretDeleteDialog
      v-if="deletion.target.value"
      :text="deletion.text.value"
      :label="deletion.isGroup.value ? words.secretGroupDelete : words.secretDelete"
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
  margin-inline-start: -0.5rem;
}

.secrets-page__title {
  margin: 0 0 0.5rem;
  color: var(--numen-ink);
  font-size: var(--numen-text-3);
  font-weight: 600;
}

.secrets-page__about {
  margin: 0 0 0.5rem;
  color: var(--numen-hushed);
}

.secrets-page__where {
  margin: 0 0 var(--settings-apart);
  padding-block-end: 1rem;
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  overflow-wrap: anywhere;
}

.secrets-page__empty {
  margin: 0 0 var(--settings-apart);
  color: var(--numen-hushed);
}

.secrets-page__add {
  align-self: flex-start;
  margin-inline-start: -0.5rem;
}

.secrets-page__new {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-block-end: var(--settings-apart);
}

.secrets-page__new-group {
  box-sizing: border-box;
  block-size: var(--numen-action-size);
  padding: 0.25rem 0.625rem;
  background-color: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
  font-family: inherit;
  font-size: var(--numen-text-2);
}

.secrets-page__new-error {
  margin: 0.5rem 0 0;
  color: var(--numen-ink);
  font-size: var(--numen-text-1);
}

.secrets-page__new-secrets {
  margin: 0;
  padding: 0;
  list-style: none;
}
</style>
