<script setup lang="ts">
/**
 * Settings for transcription and optical character recognition (OCR).
 */
import { computed } from 'vue'
import { NumberField, Select, Switch } from '@numen/ui'
import type { SelectChoice } from '@numen/ui'
import SettingRow from './setting-row/SettingRow.vue'
import { OcrSettings } from './ocr-settings'
import type { SettingsTabState } from '../types'
import paths from '../paths.json'
import { createSettingFields } from '@/entities/settings'
import { WORDS as words } from '../words'

// --- Props & Emits ---
const props = defineProps<{ state: SettingsTabState }>()

// --- State ---
const installation = computed(() => props.state.installation)
const fields = createSettingFields(() => props.state.installation)

const WHOLE = Number.MAX_SAFE_INTEGER
const UNDER = { least: -1, most: WHOLE }

const profiles = computed<readonly SelectChoice[]>(() => {
  const storedProfiles = installation.value.getSetting(paths.profiles)
  const names =
    storedProfiles && typeof storedProfiles === 'object' ? Object.keys(storedProfiles) : []
  return [{ id: '', text: words.proofreadingNone }, ...names.map((one) => ({ id: one, text: one }))]
})

// --- Handlers ---
function onTranscribingChange(value: boolean) {
  fields.setValue(paths.transcribing, value)
}

function onTranscribeUnderChange(size: number | null) {
  if (size !== null) fields.setValue(paths.transcribeUnder, size)
}

function onTranscriptProofreadChange(name: string) {
  fields.setValue(paths.transcriptProofread, name)
}

function onTranscriptProofreadAlwaysChange(value: boolean) {
  fields.setValue(paths.transcriptProofreadAlways, value)
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
        :model-value="fields.isEnabled(paths.transcribing)"
        :aria-labelledby="labelledBy"
        @update:model-value="onTranscribingChange"
      />
    </SettingRow>

    <SettingRow
      v-slot="{ labelledBy }"
      at="under"
      :name="words.transcribeUnder"
      :detail="words.transcribeUnderDetail"
    >
      <NumberField
        :model-value="fields.getNumber(paths.transcribeUnder)"
        :min="UNDER.least"
        :max="UNDER.most"
        :step="1"
        :aria-labelledby="labelledBy"
        class="settings__number"
        @commit="onTranscribeUnderChange"
      />
    </SettingRow>

    <SettingRow
      v-slot="{ labelledBy }"
      at="transcript-proofread"
      :name="words.transcriptProofread"
      :detail="words.transcriptProofreadDetail"
    >
      <Select
        :model-value="fields.getString(paths.transcriptProofread)"
        :choices="profiles"
        :name="words.transcriptProofread"
        :aria-labelledby="labelledBy"
        class="settings__choice"
        @update:model-value="onTranscriptProofreadChange"
      />
    </SettingRow>

    <SettingRow
      v-slot="{ labelledBy }"
      at="transcript-always"
      :name="words.transcriptProofreadAlways"
      :detail="words.transcriptProofreadAlwaysDetail"
    >
      <Switch
        :model-value="fields.isEnabled(paths.transcriptProofreadAlways)"
        :aria-labelledby="labelledBy"
        @update:model-value="onTranscriptProofreadAlwaysChange"
      />
    </SettingRow>
  </section>

  <OcrSettings :state="props.state" :profiles="profiles" />
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
