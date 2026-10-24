<script setup lang="ts">
/**
 * Dedicated settings view for a single selected plugin: status, enabled switch,
 * failure recovery, secrets, AI consent, and declarative settings fields.
 */
import { computed } from 'vue'
import { Button, Switch } from '@numen/ui'
import SettingRow from '../setting-row/SettingRow.vue'
import { SecretField } from '../secret-field'
import PluginFields from './PluginFields.vue'
import type { SettingsTabState } from '../../types'
import { createSettingFields, write, type PluginInfo } from '@/entities/settings'
import { formatFailedAt } from '../../lib/failures'
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
const hasSecrets = computed(() => props.plugin.secrets.length > 0)
const hasConsent = computed(() => props.plugin.consent !== 'none')
const hasDeclarativeSettings = computed(() => props.plugin.settings.length > 0)

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
</script>

<template>
  <div class="plugin-detail">
    <div class="plugin-detail__navigation">
      <Button variant="outline" size="small" class="plugin-detail__back" @click="onBack">
        &larr; {{ words.backToPlugins }}
      </Button>
    </div>

    <div class="plugin-detail__header">
      <h2 class="plugin-detail__title">
        <span>{{ props.plugin.name }}</span>
        <span v-if="props.plugin.version" class="plugin-version-badge">
          v{{ props.plugin.version }}
        </span>
      </h2>
      <p v-if="props.plugin.description" class="plugin-detail__description">
        {{ props.plugin.description }}
      </p>

      <div class="plugin-detail__master-switch">
        <Switch
          :model-value="isEnabled"
          :aria-label="`${words.pluginEnabled} ${props.plugin.name}`"
          @update:model-value="onToggleEnabled"
        />
      </div>
    </div>

    <!-- Failure warning card -->
    <div
      v-if="failure"
      class="plugin-detail__stopped"
      role="group"
      :aria-label="words.pluginsStopped"
    >
      <div class="plugin-detail__stopped-why">
        <span class="plugin-detail__stopped-reason">{{ failure.reason }}</span>
        <span class="plugin-detail__stopped-when">
          {{ words.pluginStoppedAt(formatFailedAt(failure.failedAt)) }}
        </span>
      </div>
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
    <template v-if="hasSecrets">
      <SecretField
        v-for="name in props.plugin.secrets"
        :key="`${props.plugin.id}-${name}`"
        :at="`key-${props.plugin.id}-${name}`"
        :name="`${words.keys}: ${name}`"
        :is-stored="props.plugin.storedSecrets.includes(name)"
        :words="words"
        @save="(key) => onSecretSave(name, key)"
        @delete="onSecretDelete(name)"
      />
    </template>

    <!-- Declarative settings -->
    <PluginFields v-if="hasDeclarativeSettings" :state="props.state" :plugin="props.plugin" />
  </div>
</template>

<style scoped>
.plugin-detail {
  display: flex;
  flex-direction: column;
  max-inline-size: var(--settings-measure, 54rem);
  margin-inline: auto;
}

.plugin-detail__navigation {
  margin-block-end: 1rem;
}

.plugin-detail__header {
  display: grid;
  grid-template-columns: 1fr auto;
  grid-template-areas:
    'title switch'
    'desc  switch';
  column-gap: var(--numen-panel-gap, 0.75rem);
  row-gap: 0.25rem;
  align-items: center;
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

.plugin-detail__description {
  grid-area: desc;
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

.plugin-detail__master-switch {
  grid-area: switch;
  display: flex;
  align-items: center;
}

.plugin-detail__stopped {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--settings-near, 0.375rem);
  margin-block-end: var(--settings-apart, 1.75rem);
  padding: 0.5rem 0.75rem;
  border: var(--numen-stroke, 1px) solid var(--numen-alarm, #d14);
  border-radius: var(--numen-radius-small, 4px);
  color: var(--numen-ink);
}

.plugin-detail__stopped-why {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.plugin-detail__stopped-reason {
  overflow-wrap: anywhere;
}

.plugin-detail__stopped-when {
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

.plugin-detail__stopped-actions {
  display: flex;
  gap: var(--settings-near, 0.375rem);
}
</style>
