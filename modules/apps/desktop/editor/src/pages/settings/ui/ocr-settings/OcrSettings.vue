<script setup lang="ts">
/**
 * Settings for optical character recognition (OCR).
 */
import { Select, Switch } from '@numen/ui'
import type { SelectChoice } from '@numen/ui'
import SettingRow from '../setting-row/SettingRow.vue'
import type { SettingsTabState } from '../../types'
import paths from '../../paths.json'
import { createModelChoices } from '../../lib/models'
import { createSettingFields } from '@/entities/settings'
import { WORDS as words } from '../../words'

// --- Props & Emits ---
const props = defineProps<{
  state: SettingsTabState
  /** The proofreading profiles a person has, as the choice offers them. */
  profiles: readonly SelectChoice[]
}>()

// --- State ---
const fields = createSettingFields(() => props.state.installation)

const getChoices = (path: readonly string[]): readonly SelectChoice[] =>
  createModelChoices(fields, path, words)

// --- Handlers ---
function onOcrModelChange(name: string) {
  fields.chooseModel(paths.ocrModel, name)
}

function onOcrProofreadChange(name: string) {
  fields.setValue(paths.ocrProofread, name)
}

function onOcrProofreadAlwaysChange(value: boolean) {
  fields.setValue(paths.ocrProofreadAlways, value)
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
        :model-value="fields.getString(paths.ocrModel)"
        :choices="getChoices(paths.ocrModel)"
        :name="words.ocrModel"
        :aria-labelledby="labelledBy"
        class="settings__choice"
        @update:model-value="onOcrModelChange"
      />
    </SettingRow>

    <SettingRow
      v-slot="{ labelledBy }"
      at="ocr-proofread"
      :name="words.ocrProofread"
      :detail="words.ocrProofreadDetail"
    >
      <Select
        :model-value="fields.getString(paths.ocrProofread)"
        :choices="props.profiles"
        :name="words.ocrProofread"
        :aria-labelledby="labelledBy"
        class="settings__choice"
        @update:model-value="onOcrProofreadChange"
      />
    </SettingRow>

    <SettingRow
      v-slot="{ labelledBy }"
      at="ocr-always"
      :name="words.ocrProofreadAlways"
      :detail="words.ocrProofreadAlwaysDetail"
    >
      <Switch
        :model-value="fields.isEnabled(paths.ocrProofreadAlways)"
        :aria-labelledby="labelledBy"
        @update:model-value="onOcrProofreadAlwaysChange"
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
