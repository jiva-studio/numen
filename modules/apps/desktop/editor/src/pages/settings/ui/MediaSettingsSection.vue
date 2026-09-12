<script setup lang="ts">
/**
 * Settings for transcription and optical character recognition (OCR).
 */
import { computed } from 'vue'
import { NumberField, Select, Switch } from '@numen/ui'
import type { SelectChoice } from '@numen/ui'
import SettingRow from './setting-row/SettingRow.vue'
import type { SettingsTabState } from '../model/useSettingsTab'
import AT from '../paths.json'
import { choicesFor } from '../lib/models'
import { write } from '@/entities/settings'
import { WORDS as words } from '../words'

// --- Props & Emits ---
const props = defineProps<{ state: SettingsTabState }>()

// --- State ---
const installation = computed(() => props.state.installation)

const WHOLE = Number.MAX_SAFE_INTEGER
const UNDER = { least: -1, most: WHOLE }

const profiles = computed<readonly SelectChoice[]>(() => {
  const kept = installation.value.setting(AT.profiles)
  const names = kept && typeof kept === 'object' ? Object.keys(kept) : []
  return [{ id: '', text: words.proofreadingNone }, ...names.map((one) => ({ id: one, text: one }))]
})

// --- Handlers ---
function onModelChange(at: readonly string[], name: string) {
  const model = installation.value.models(at).find((one) => one.name === name)
  if (model) installation.value.write(model.writes)
  else setSetting(at, name)
}

function onSettingChange(at: readonly string[], value: unknown) {
  setSetting(at, value)
}

// --- Helpers ---
function getSettingString(at: readonly string[]): string {
  const value = installation.value.setting(at)
  return typeof value === 'string' ? value : ''
}
const said = getSettingString

function isSettingEnabled(at: readonly string[]): boolean {
  return installation.value.setting(at) === true
}
const on = isSettingEnabled

function getSettingNumber(at: readonly string[]): number | null {
  const value = installation.value.setting(at)
  return typeof value === 'number' ? value : null
}
const counted = getSettingNumber

function getModels(at: readonly string[]): readonly SelectChoice[] {
  return choicesFor(installation.value.models(at), getSettingString(at), words)
}
const models = getModels

function setSetting(at: readonly string[], value: unknown): void {
  installation.value.write([{ at, value: write(value) }])
}
</script>

<template>
  <section class="settings__group" :aria-label="words.transcription">
    <h2 class="settings__heading">{{ words.transcription }}</h2>

    <SettingRow
      v-slot="{ labelledBy }"
      at="transcribing"
      :name="words.transcribing"
      :detail="words.transcribingDetail"
    >
      <Switch
        :model-value="on(AT.transcribing)"
        :aria-labelledby="labelledBy"
        @update:model-value="(kept: boolean) => onSettingChange(AT.transcribing, kept)"
      />
    </SettingRow>

    <SettingRow
      v-slot="{ labelledBy }"
      at="under"
      :name="words.transcribeUnder"
      :detail="words.transcribeUnderDetail"
    >
      <NumberField
        :model-value="counted(AT.transcribeUnder)"
        :min="UNDER.least"
        :max="UNDER.most"
        :step="1"
        :aria-labelledby="labelledBy"
        class="settings__number"
        @settles="(size: number | null) => size !== null && onSettingChange(AT.transcribeUnder, size)"
      />
    </SettingRow>

    <SettingRow
      v-slot="{ labelledBy }"
      at="transcript-proofread"
      :name="words.transcriptProofread"
      :detail="words.transcriptProofreadDetail"
    >
      <Select
        :model-value="said(AT.transcriptProofread)"
        :choices="profiles"
        :name="words.transcriptProofread"
        :aria-labelledby="labelledBy"
        class="settings__choice"
        @update:model-value="(name: string) => onSettingChange(AT.transcriptProofread, name)"
      />
    </SettingRow>

    <SettingRow
      v-slot="{ labelledBy }"
      at="transcript-always"
      :name="words.transcriptProofreadAlways"
      :detail="words.transcriptProofreadAlwaysDetail"
    >
      <Switch
        :model-value="on(AT.transcriptProofreadAlways)"
        :aria-labelledby="labelledBy"
        @update:model-value="(kept: boolean) => onSettingChange(AT.transcriptProofreadAlways, kept)"
      />
    </SettingRow>
  </section>

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
        :choices="profiles"
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

.settings__number {
  inline-size: var(--settings-value, 6rem);
}

.settings__choice {
  inline-size: var(--settings-choice, 18rem);
}
</style>
