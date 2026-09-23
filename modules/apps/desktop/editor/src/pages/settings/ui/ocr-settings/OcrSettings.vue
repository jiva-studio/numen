<script setup lang="ts">
/**
 * Settings for optical character recognition (OCR).
 */
import { computed } from 'vue'
import { Select, Switch } from '@numen/ui'
import type { SelectChoice } from '@numen/ui'
import SettingRow from '../setting-row/SettingRow.vue'
import type { SettingsTabState } from '../../types'
import AT from '../../paths.json'
import { choicesFor } from '../../lib/models'
import { write } from '@/entities/settings'
import { WORDS as words } from '../../words'

// --- Props & Emits ---
const props = defineProps<{
  state: SettingsTabState
  /** The proofreading profiles a person has, as the choice offers them. */
  profiles: readonly SelectChoice[]
}>()

// --- State ---
const installation = computed(() => props.state.installation)

// --- Handlers ---
function onModelChange(at: readonly string[], name: string) {
  const model = installation.value.getModels(at).find((one) => one.name === name)
  if (model) installation.value.write(model.writes)
  else setSetting(at, name)
}

function onSettingChange(at: readonly string[], value: unknown) {
  setSetting(at, value)
}

// --- Helpers ---
function getSettingString(at: readonly string[]): string {
  const value = installation.value.getSetting(at)
  return typeof value === 'string' ? value : ''
}
const said = getSettingString

function isSettingEnabled(at: readonly string[]): boolean {
  return installation.value.getSetting(at) === true
}
const on = isSettingEnabled

function getModels(at: readonly string[]): readonly SelectChoice[] {
  return choicesFor(installation.value.getModels(at), getSettingString(at), words)
}
const models = getModels

function setSetting(at: readonly string[], value: unknown): void {
  installation.value.write([{ at, value: write(value) }])
}
</script>

<template>
  <section class="settings__group" :aria-label="words.ocr">
    <h2 class="settings__heading">{{ words.ocr }}</h2>

    <SettingRow
      v-slot="{ labelledBy }"
      at="ocr"
      :name="words.ocrModel"
      :detail="words.ocrModelDetail"
    >
      <Select
        :model-value="said(AT.ocrModel)"
        :choices="models(AT.ocrModel)"
        :name="words.ocrModel"
        :aria-labelledby="labelledBy"
        class="settings__choice"
        @update:model-value="(name: string) => onModelChange(AT.ocrModel, name)"
      />
    </SettingRow>

    <SettingRow
      v-slot="{ labelledBy }"
      at="ocr-proofread"
      :name="words.ocrProofread"
      :detail="words.ocrProofreadDetail"
    >
      <Select
        :model-value="said(AT.ocrProofread)"
        :choices="props.profiles"
        :name="words.ocrProofread"
        :aria-labelledby="labelledBy"
        class="settings__choice"
        @update:model-value="(name: string) => onSettingChange(AT.ocrProofread, name)"
      />
    </SettingRow>

    <SettingRow
      v-slot="{ labelledBy }"
      at="ocr-always"
      :name="words.ocrProofreadAlways"
      :detail="words.ocrProofreadAlwaysDetail"
    >
      <Switch
        :model-value="on(AT.ocrProofreadAlways)"
        :aria-labelledby="labelledBy"
        @update:model-value="(kept: boolean) => onSettingChange(AT.ocrProofreadAlways, kept)"
      />
    </SettingRow>
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

.settings__choice {
  inline-size: var(--settings-choice, 18rem);
}
</style>
