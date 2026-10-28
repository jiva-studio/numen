<script setup lang="ts">
/**
 * Dedicated settings view for a single selected plugin: header card with
 * expandable README documentation, enabled switch, failure recovery, AI
 * consent, declarative settings fields, and vault data cleanup.
 */
import { computed, ref, watch } from 'vue'
import { ChevronDown } from '@lucide/vue'
import { Button, Prose, Switch } from '@numen/ui'
import SettingRow from '../setting-row/SettingRow.vue'
import PluginFields from './PluginFields.vue'
import type { SettingsTabState } from '../../types'
import { createSettingFields, isShown, write, type PluginInfo } from '@/entities/settings'
import { formatFailedAt } from '../../lib/failures'
import { parsePluginReadme } from '../../lib/readme'
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
const readmeText = ref('')
const isExpanded = ref(false)
const readmeParts = computed(() =>
  parsePluginReadme(readmeText.value, props.plugin.id, props.plugin.description ?? ''),
)

/* -------------------------------- Handlers -------------------------------- */
function onBack(): void {
  emit('back')
}

function onToggleEnabled(val: boolean): void {
  fields.setValue(['plugins', props.plugin.id, 'enabled'], val)
}

function onToggleReadme(): void {
  isExpanded.value = !isExpanded.value
}

function onRetry(): void {
  installation.value.retryPlugin(props.plugin.id)
}

function onConsentChange(isAllowed: boolean): void {
  installation.value.write([
    { at: ['plugins', props.plugin.id, 'aiConsent'], value: write(isAllowed) },
  ])
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

async function readReadme(pluginId: string): Promise<void> {
  try {
    if (typeof fetch !== 'function') return
    const response = await fetch(`/plugin/${pluginId}/README.md`)
    if (!response.ok) return
    const contentType = response.headers?.get?.('content-type') ?? ''
    if (contentType.includes('text/html')) return
    const text = await response.text()
    if (props.plugin.id === pluginId) {
      readmeText.value = text
    }
  } catch {
    // Fallback to manifest description when README cannot be fetched.
  }
}

watch(
  () => props.plugin.id,
  (pluginId) => {
    isAskingDelete.value = false
    deleteProblem.value = ''
    readmeText.value = ''
    isExpanded.value = false
    void readFileCount()
    void readReadme(pluginId)
  },
  { immediate: true },
)
</script>

<template>
  <div class="plugin-detail settings__plugins">
    <Button variant="ghost" size="small" class="plugin-detail__back" @click="onBack">
      &larr; {{ words.backToPlugins }}
    </Button>

    <div class="plugin-detail__header">
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

      <div v-if="readmeParts.summary" class="plugin-detail__description">
        <Prose :text="readmeParts.summary" />
      </div>

      <div v-if="isExpanded && readmeParts.more" class="plugin-detail__more">
        <Prose :text="readmeParts.more" />
      </div>

      <button
        v-if="readmeParts.more"
        type="button"
        class="plugin-detail__toggle"
        :aria-expanded="isExpanded"
        @click="onToggleReadme"
      >
        <span>{{ isExpanded ? words.showLess : words.readMore }}</span>
        <ChevronDown
          :size="14"
          class="plugin-detail__toggle-icon"
          :class="{ 'plugin-detail__toggle-icon--open': isExpanded }"
          aria-hidden="true"
        />
      </button>
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

    <div
      v-if="hasConsent || hasDeclarativeSettings || shownSecrets.length > 0 || fileCount > 0"
      class="plugin-detail__settings"
    >
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

      <!-- Declarative settings and secrets -->
      <PluginFields
        v-if="hasDeclarativeSettings || shownSecrets.length > 0"
        :state="props.state"
        :plugin="props.plugin"
      />

      <!-- Data the plugin keeps in the shown vault -->
      <SettingRow
        v-if="fileCount > 0"
        :at="`data-${props.plugin.id}`"
        :name="words.pluginData"
        :detail="words.pluginDataDetail(fileCount)"
      >
        <Button variant="danger" size="small" @click="onDeleteData">
          {{ words.pluginDataDelete }}
        </Button>
      </SettingRow>
    </div>
    <div
      v-if="isAskingDelete"
      class="plugin-detail__asking"
      role="alert"
      :aria-label="words.pluginDataDelete"
    >
      <span>{{ words.pluginDataAsking(props.plugin.name, fileCount) }}</span>
      <span class="plugin-detail__asking-answers">
        <Button variant="danger" size="small" @click="onConfirmDeleteData">
          {{ words.pluginDataConfirm }}
        </Button>
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
  color: var(--numen-hushed);
}

.plugin-detail__settings {
  padding-inline: 1rem;
  border: var(--numen-stroke, 1px) solid var(--numen-rule);
  border-radius: var(--numen-radius, 0.375rem);
  background-color: var(--numen-raised, rgba(125, 125, 125, 0.04));
}

.plugin-detail__settings :deep(.settings__row:last-child) {
  border-block-end: none;
}

.plugin-detail__header {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  column-gap: var(--numen-panel-gap, 0.75rem);
  row-gap: 0.625rem;
  align-items: center;
  padding: 0.875rem 1rem;
  margin-block-end: 1.25rem;
  border: var(--numen-stroke, 1px) solid var(--numen-rule);
  border-radius: var(--numen-radius, 0.375rem);
  background-color: var(--numen-raised, rgba(125, 125, 125, 0.04));
}

.plugin-detail__title {
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
  justify-self: end;
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.plugin-detail__description {
  grid-column: 1 / -1;
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
}

.plugin-detail__description :deep(.prose) {
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
}

.plugin-detail__description :deep(p:first-child),
.plugin-detail__more :deep(:first-child) {
  margin-block-start: 0;
}

.plugin-detail__description :deep(p:last-child),
.plugin-detail__more :deep(:last-child) {
  margin-block-end: 0;
}

.plugin-detail__more {
  grid-column: 1 / -1;
  padding-block-start: 0.625rem;
  border-block-start: var(--numen-stroke, 1px) solid var(--numen-rule);
  font-size: var(--numen-text-2);
}

.plugin-detail__toggle {
  grid-column: 1 / -1;
  justify-self: start;
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.125rem 0;
  border: none;
  background: none;
  color: var(--numen-hushed);
  font-family: inherit;
  font-size: var(--numen-text-1);
  font-weight: 500;
  cursor: pointer;
}

.plugin-detail__toggle:hover {
  color: var(--numen-ink);
}

.plugin-detail__toggle-icon {
  transition: transform var(--numen-motion-hover, 110ms) ease;
}

.plugin-detail__toggle-icon--open {
  transform: rotate(180deg);
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
