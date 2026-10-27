<script setup lang="ts">
/**
 * Dedicated settings view for a single selected plugin: status, enabled switch,
 * failure recovery, secrets, AI consent, and declarative settings fields.
 */
import { computed, ref, watch } from 'vue'
import { Button, Switch } from '@numen/ui'
import SettingRow from '../setting-row/SettingRow.vue'
import { SecretField } from '../secret-field'
import PluginFields from './PluginFields.vue'
import type { SettingsTabState } from '../../types'
import { createSettingFields, isShown, write, type PluginInfo } from '@/entities/settings'
import { formatFailedAt, getPluginState, type PluginState } from '../../lib/failures'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  state: SettingsTabState
  plugin: PluginInfo
}>()

const emit = defineEmits<{
  (e: 'back'): void
}>()

/* --------------------------------- State ---------------------------------- */
const installation = computed(() => props.state.installation)
const fields = createSettingFields(() => props.state.installation)

const isEnabled = computed<boolean>(() => {
  const val = installation.value.getSetting(['plugins', props.plugin.id, 'enabled'])
  return typeof val === 'boolean' ? val : true
})

const failure = computed(() => props.plugin.failure)
const read = (key: string) => installation.value.getSetting(['plugins', props.plugin.id, key])
const shownSecrets = computed(() =>
  props.plugin.secrets.filter((one) => isShown(props.plugin, one.when, read)),
)
const hasConsent = computed(() => props.plugin.consent !== 'none')
const hasDeclarativeSettings = computed(() => props.plugin.settings.length > 0)
/** How many files the plugin keeps in the shown vault; zero hides the delete action. */
const fileCount = ref(0)
const isAskingDelete = ref(false)
const deleteProblem = ref('')
const hasBody = computed(
  () =>
    failure.value !== undefined ||
    hasConsent.value ||
    shownSecrets.value.length > 0 ||
    hasDeclarativeSettings.value ||
    fileCount.value > 0,
)
const pluginState = computed<PluginState>(() => getPluginState(props.plugin, isEnabled.value))

/* -------------------------------- Handlers -------------------------------- */
function onBack(): void {
  emit('back')
}

function onToggleEnabled(val: boolean): void {
  fields.setValue(['plugins', props.plugin.id, 'enabled'], val)
}

function onRetry(): void {
  installation.value.retryPlugin(props.plugin.id)
}

function onConsentChange(isAllowed: boolean): void {
  installation.value.write([
    { at: ['plugins', props.plugin.id, 'aiConsent'], value: write(isAllowed) },
  ])
}

function onSecretSave(name: string, key: string): void {
  installation.value.writeSecret(props.plugin.id, name, key)
}

function onSecretDelete(name: string): void {
  installation.value.deleteSecret(props.plugin.id, name)
}

function onDeleteData(): void {
  isAskingDelete.value = true
}

function onKeepData(): void {
  isAskingDelete.value = false
}

async function onConfirmDeleteData(): Promise<void> {
  isAskingDelete.value = false
  deleteProblem.value = (await installation.value.deletePluginData?.(props.plugin.id)) ?? ''
  await readFileCount()
}

/* -------------------------------- Helpers --------------------------------- */
async function readFileCount(): Promise<void> {
  try {
    fileCount.value = (await installation.value.getPluginData?.(props.plugin.id)) ?? 0
  } catch {
    fileCount.value = 0
  }
}

watch(
  () => props.plugin.id,
  () => {
    isAskingDelete.value = false
    deleteProblem.value = ''
    void readFileCount()
  },
  { immediate: true },
)
</script>

<template>
  <div class="plugin-detail settings__plugins">
    <Button variant="ghost" size="small" class="plugin-detail__back" @click="onBack">
      &larr; {{ words.backToPlugins }}
    </Button>

    <div class="plugin-detail__header" :class="{ 'plugin-detail__header--ruled': hasBody }">
      <h2 class="plugin-detail__title">
        <span>{{ props.plugin.name }}</span>
        <span v-if="props.plugin.version" class="plugin-version-badge">
          v{{ props.plugin.version }}
        </span>
      </h2>
      <div class="plugin-detail__master-switch">
        <Switch
          :model-value="isEnabled"
          :aria-label="`${words.pluginEnabled} ${props.plugin.name}`"
          @update:model-value="onToggleEnabled"
        />
      </div>
      <span class="plugin-detail__state" :class="`plugin-detail__state--${pluginState}`">
        {{ words.pluginState[pluginState] }}
      </span>
      <p v-if="props.plugin.description" class="plugin-detail__description">
        {{ props.plugin.description }}
      </p>
    </div>

    <div
      v-if="failure"
      class="plugin-detail__stopped"
      role="group"
      :aria-label="words.pluginState.stopped"
    >
      <span class="plugin-detail__stopped-reason">{{ failure.reason }}</span>
      <span class="plugin-detail__stopped-when">{{ formatFailedAt(failure.failedAt) }}</span>
      <details v-if="failure.detail" class="plugin-detail__stopped-detail">
        <summary>{{ words.pluginDetails }}</summary>
        <code>{{ failure.detail }}</code>
      </details>
      <div class="plugin-detail__stopped-actions">
        <Button size="small" @click="onRetry">{{ words.pluginRetry }}</Button>
        <Button variant="outline" size="small" @click="() => onToggleEnabled(false)">
          {{ words.pluginDisable }}
        </Button>
      </div>
    </div>

    <!-- AI consent -->
    <SettingRow
      v-if="hasConsent"
      v-slot="{ labelledBy }"
      :at="`ai-consent-${props.plugin.id}`"
      :name="words.consentAllowed"
      :detail="words.consentDetail[props.plugin.consent]"
    >
      <Switch
        :model-value="props.plugin.consent === 'allowed'"
        :aria-labelledby="labelledBy"
        @update:model-value="onConsentChange"
      />
    </SettingRow>

    <!-- Secrets / API keys -->
    <SecretField
      v-for="secret in shownSecrets"
      :key="`${props.plugin.id}-${secret.name}`"
      :at="`key-${props.plugin.id}-${secret.name}`"
      :name="secret.title"
      :is-stored="props.plugin.storedSecrets.includes(secret.name)"
      :words="words"
      @save="(key) => onSecretSave(secret.name, key)"
      @delete="onSecretDelete(secret.name)"
    />

    <!-- Declarative settings -->
    <PluginFields v-if="hasDeclarativeSettings" :state="props.state" :plugin="props.plugin" />

    <!-- Data the plugin keeps in the shown vault -->
    <SettingRow
      v-if="fileCount > 0"
      :at="`data-${props.plugin.id}`"
      :name="words.pluginData"
      :detail="words.pluginDataDetail(fileCount)"
    >
      <Button variant="outline" size="small" @click="onDeleteData">
        {{ words.pluginDataDelete }}
      </Button>
    </SettingRow>
    <div
      v-if="isAskingDelete"
      class="plugin-detail__asking"
      role="alert"
      :aria-label="words.pluginDataDelete"
    >
      <span>{{ words.pluginDataAsking(props.plugin.name, fileCount) }}</span>
      <span class="plugin-detail__asking-answers">
        <Button size="small" @click="onConfirmDeleteData">{{ words.pluginDataConfirm }}</Button>
        <Button variant="outline" size="small" @click="onKeepData">
          {{ words.pluginDataKeep }}
        </Button>
      </span>
    </div>
    <p v-if="deleteProblem" class="plugin-detail__problem" role="alert">{{ deleteProblem }}</p>
  </div>
</template>

<style scoped>
.plugin-detail {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  max-inline-size: var(--settings-measure, 54rem);
  margin-inline: auto;
}

.plugin-detail__back {
  align-self: flex-start;
  margin-block-end: 0.75rem;
  margin-inline-start: -0.5rem;
  color: var(--numen-hushed);
}

.plugin-detail__header {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  grid-template-areas:
    'title switch'
    'state state'
    'desc  desc';
  column-gap: var(--numen-panel-gap, 0.75rem);
  row-gap: 0.375rem;
  align-items: center;
  justify-items: start;
}

.plugin-detail__header--ruled {
  padding-block-end: 1rem;
  border-block-end: var(--numen-stroke, 1px) solid var(--numen-rule);
  margin-block-end: 1rem;
}

.plugin-detail__title {
  grid-area: title;
  margin: 0;
  font-size: var(--numen-text-3, 1.25rem);
  font-weight: 600;
  color: var(--numen-ink);
}

.plugin-version-badge {
  display: inline-flex;
  align-items: center;
  padding: 0.125rem 0.375rem;
  border-radius: var(--numen-radius-pill, 9999px);
  background: var(--numen-surface, rgba(125, 125, 125, 0.12));
  border: 1px solid var(--numen-rule, rgba(125, 125, 125, 0.2));
  color: var(--numen-hushed);
  font-size: var(--numen-text-1, 0.75rem);
  font-weight: 500;
  line-height: 1;
  font-variant-numeric: tabular-nums;
  vertical-align: middle;
  margin-inline-start: 0.375rem;
}

.plugin-detail__master-switch {
  grid-area: switch;
  justify-self: end;
  display: flex;
  align-items: center;
}

.plugin-detail__state {
  grid-area: state;
  border: var(--numen-stroke, 1px) solid var(--numen-rule);
  padding: 0.125rem 0.375rem;
  border-radius: var(--numen-radius-small, 3px);
  font-size: var(--numen-text-1, 0.75rem);
}

.plugin-detail__state--enabled {
  color: var(--numen-ink);
  background-color: var(--numen-surface, rgba(125, 125, 125, 0.12));
}

.plugin-detail__state--disabled {
  color: var(--numen-hushed);
  background-color: var(--numen-surface, rgba(125, 125, 125, 0.12));
}

.plugin-detail__state--stopped {
  color: var(--numen-danger, #d14);
  background-color: var(--numen-danger-soft, rgba(221, 17, 68, 0.1));
}

.plugin-detail__description {
  grid-area: desc;
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

.plugin-detail__stopped {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  grid-template-areas:
    'reason  actions'
    'when    actions'
    'detail  detail';
  column-gap: var(--numen-panel-gap, 0.75rem);
  row-gap: 0.125rem;
  align-items: start;
  margin-block-end: var(--settings-apart, 1.75rem);
  padding: 0.5rem 0.75rem;
  border: var(--numen-stroke, 1px) solid var(--numen-alarm, #d14);
  border-radius: var(--numen-radius-small, 4px);
  color: var(--numen-ink);
}

.plugin-detail__stopped:last-child {
  margin-block-end: 0;
}

.plugin-detail__stopped-reason {
  grid-area: reason;
  overflow-wrap: anywhere;
}

.plugin-detail__stopped-when {
  grid-area: when;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

.plugin-detail__stopped-detail {
  grid-area: detail;
  margin-block-start: 0.25rem;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

.plugin-detail__stopped-detail summary {
  cursor: pointer;
}

.plugin-detail__stopped-detail code {
  display: block;
  margin-block-start: 0.25rem;
  font-family: var(--numen-font-mono, monospace);
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.plugin-detail__stopped-actions {
  grid-area: actions;
  display: flex;
  gap: var(--settings-near, 0.375rem);
}

.plugin-detail__asking {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--settings-near, 0.375rem);
  margin-block-start: var(--settings-near, 0.375rem);
  padding: 0.5rem 0.75rem;
  border: var(--numen-stroke, 1px) solid var(--numen-alarm, #d14);
  border-radius: var(--numen-radius-small, 4px);
  color: var(--numen-ink);
}

.plugin-detail__asking-answers {
  display: flex;
  gap: var(--settings-near, 0.375rem);
}

.plugin-detail__problem {
  margin: var(--settings-near, 0.375rem) 0 0;
  color: var(--numen-alarm, #d14);
  font-size: var(--numen-text-1);
}
</style>
