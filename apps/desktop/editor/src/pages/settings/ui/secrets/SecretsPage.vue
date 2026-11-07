<script setup lang="ts">
/**
 * The page of secrets: every secret by group, in priority order, with the plugin declarations that
 * may read it. A value is entered here once and is never shown again.
 */
import { computed, ref } from 'vue'
import { Button } from '@numen/ui'
import type { Secret } from '@/entities/settings'
import {
  createSecretSections,
  getDeclarations,
  getNextSecretName,
  getReaderNames,
  type Declaration,
} from '../../lib/secretSections'
import type { SettingsTabState } from '../../types'
import { WORDS as words } from '../../words'
import SecretDeleteDialog from './SecretDeleteDialog.vue'
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
const deleting = ref<Secret | null>(null)
const readers = computed(() =>
  deleting.value ? getReaderNames(deleting.value, plugins.value) : [],
)

/* -------------------------------- Handlers -------------------------------- */
function onAdd(group: string): void {
  const section = sections.value.find((one) => one.group === group)
  const namesInGroup = [
    ...(section?.secrets.map((one) => one.name) ?? []),
    ...(section?.isAsked && section.secrets.length === 0 ? [words.secretDefault] : []),
    ...getDraftNames(group),
  ]
  drafts.value = [...drafts.value, { group, name: getNextSecretName(namesInGroup) }]
}

function onDiscard(group: string, name: string): void {
  drafts.value = drafts.value.filter((one) => one.group !== group || one.name !== name)
}

async function onCreate(group: string, name: string, value: string): Promise<void> {
  const id = await installation.value.createSecret(group, name, value)
  if (id !== '') onDiscard(group, name)
}

function onToggle(secret: Secret, declaration: Declaration, isAllowed: boolean): void {
  const { pluginId, name, group } = declaration
  if (isAllowed) installation.value.allowPluginSecret(secret.id, pluginId, name, group)
  else installation.value.denyPluginSecret(secret.id, pluginId, name, group)
}

function onConfirmDelete(): void {
  if (deleting.value) installation.value.deleteSecret(deleting.value.id)
  deleting.value = null
}

/* -------------------------------- Helpers --------------------------------- */
function getDraftNames(group: string): readonly string[] {
  return drafts.value.filter((one) => one.group === group).map((one) => one.name)
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
      @create="(name, value) => onCreate(section.group, name, value)"
      @discard="(name) => onDiscard(section.group, name)"
      @move="(id, beforeId) => installation.moveSecret(id, beforeId)"
      @rename="(id, name) => installation.renameSecret(id, name)"
      @write="(id, value) => installation.writeSecret(id, value)"
      @delete="(secret) => (deleting = secret)"
      @toggle="onToggle"
    />

    <SecretDeleteDialog
      v-if="deleting"
      :name="deleting.name"
      :readers="readers"
      @confirm="onConfirmDelete"
      @cancel="deleting = null"
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
</style>
