<script setup lang="ts">
/**
 * Settings for workload performance profiles, batching, and hardware resource allocation.
 */
import { computed } from 'vue'
import { NumberField, SegmentedControl } from '@numen/ui'
import SettingRow from './setting-row/SettingRow.vue'
import type { SettingsTabState } from '../types'
import paths from '../paths.json'
import { write } from '@/entities/settings'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ state: SettingsTabState }>()

/* --------------------------------- State ---------------------------------- */
const installation = computed(() => props.state.installation)

const profiles = [
  { id: 'eco', text: words.eco },
  { id: 'balanced', text: words.balanced },
  { id: 'maximum', text: words.maximum },
  { id: 'custom', text: words.custom },
]

const currentProfile = computed<string>(() => {
  const profile = getSettingString(paths.performanceProfile)
  return profile || 'balanced'
})

const isCustom = computed<boolean>(() => currentProfile.value === 'custom')

const profileDetail = computed<string>(() => {
  switch (currentProfile.value) {
    case 'eco':
      return words.ecoDetail
    case 'maximum':
      return words.maximumDetail
    case 'custom':
      return words.customDetail
    case 'balanced':
    default:
      return words.balancedDetail
  }
})

/* -------------------------------- Handlers -------------------------------- */
function onProfileChange(choice: string): void {
  setSetting(paths.performanceProfile, choice)
}

function onEmbeddingBatchSizeChange(size: number | null): void {
  if (size !== null) setSetting(paths.performanceEmbeddingBatchSize, size)
}

function onOcrThreadsChange(threads: number | null): void {
  if (threads !== null) setSetting(paths.performanceOcrThreads, threads)
}

function onOcrSessionsChange(sessions: number | null): void {
  if (sessions !== null) setSetting(paths.performanceOcrSessions, sessions)
}

function onLlmConcurrencyChange(concurrency: number | null): void {
  if (concurrency !== null) setSetting(paths.performanceLlmConcurrency, concurrency)
}

/* -------------------------------- Helpers --------------------------------- */
function getSettingString(at: readonly string[]): string {
  const value = installation.value.getSetting(at)
  return typeof value === 'string' ? value : ''
}

function getSettingNumber(at: readonly string[], fallback: number): number {
  const value = installation.value.getSetting(at)
  return typeof value === 'number' ? value : fallback
}

function setSetting(at: readonly string[], value: unknown): void {
  installation.value.write([{ at, value: write(value) }])
}
</script>

<template>
  <section class="settings__group" :aria-label="words.performance">
    <h2 class="settings__heading">{{ words.performance }}</h2>

    <SettingRow
      v-slot="{ labelledBy }"
      at="performance-profile"
      :name="words.performanceProfile"
      :detail="profileDetail"
    >
      <SegmentedControl
        :model-value="currentProfile"
        :choices="profiles"
        :aria-labelledby="labelledBy"
        @update:model-value="onProfileChange"
      />
    </SettingRow>

    <template v-if="isCustom">
      <SettingRow
        v-slot="{ labelledBy }"
        at="embedding-batch-size"
        :name="words.embeddingBatchSize"
        :detail="words.embeddingBatchSizeDetail"
      >
        <NumberField
          :model-value="getSettingNumber(paths.performanceEmbeddingBatchSize, 32)"
          :min="1"
          :max="512"
          :step="1"
          :aria-labelledby="labelledBy"
          class="settings__number"
          @settle="onEmbeddingBatchSizeChange"
        />
      </SettingRow>

      <SettingRow
        v-slot="{ labelledBy }"
        at="ocr-threads"
        :name="words.ocrThreads"
        :detail="words.ocrThreadsDetail"
      >
        <NumberField
          :model-value="getSettingNumber(paths.performanceOcrThreads, 4)"
          :min="1"
          :max="64"
          :step="1"
          :aria-labelledby="labelledBy"
          class="settings__number"
          @settle="onOcrThreadsChange"
        />
      </SettingRow>

      <SettingRow
        v-slot="{ labelledBy }"
        at="ocr-sessions"
        :name="words.ocrSessions"
        :detail="words.ocrSessionsDetail"
      >
        <NumberField
          :model-value="getSettingNumber(paths.performanceOcrSessions, 1)"
          :min="1"
          :max="16"
          :step="1"
          :aria-labelledby="labelledBy"
          class="settings__number"
          @settle="onOcrSessionsChange"
        />
      </SettingRow>

      <SettingRow
        v-slot="{ labelledBy }"
        at="llm-concurrency"
        :name="words.llmConcurrency"
        :detail="words.llmConcurrencyDetail"
      >
        <NumberField
          :model-value="getSettingNumber(paths.performanceLlmConcurrency, 2)"
          :min="1"
          :max="16"
          :step="1"
          :aria-labelledby="labelledBy"
          class="settings__number"
          @settle="onLlmConcurrencyChange"
        />
      </SettingRow>
    </template>
  </section>
</template>

<style scoped>
.settings__group {
  max-inline-size: var(--settings-measure, 54rem);
  margin-block-end: var(--settings-apart, 1.75rem);
  margin-inline: auto;
}

.settings__heading {
  margin: 0 0 var(--settings-near, 0.375rem);
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  font-weight: 600;
  letter-spacing: var(--numen-caps-tracking);
  text-transform: uppercase;
}

.settings__number {
  inline-size: var(--settings-value, 6rem);
}
</style>
