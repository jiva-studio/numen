<script setup lang="ts">
/**
 * Settings for the window appearance and the naming of notes.
 */
import { computed } from 'vue'
import { NumberField, SegmentedControl, Select } from '@numen/ui'
import type { SelectChoice } from '@numen/ui'
import SettingRow from './setting-row/SettingRow.vue'
import { NamingSettings } from './naming-settings'
import type { SettingsTabState } from '../types'
import { toThemeChoices } from '../lib/enums'
import type { Mode } from '@/entities/settings'
import { INTERFACE_SCALE, MODE, TEXT_SCALE } from '@/features/settings-commands'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ state: SettingsTabState }>()

/* --------------------------------- State ---------------------------------- */
const installation = computed(() => props.state.installation)

const bounds = computed(() => installation.value.bounds.value)

const modes = [
  { id: 'system', text: words.system },
  { id: 'light', text: words.light },
  { id: 'dark', text: words.dark },
]

const themes = computed<readonly SelectChoice[]>(() =>
  toThemeChoices(installation.value.themes.value, words.shipped, words.owned),
)

const STEP = 0.1

/* -------------------------------- Handlers -------------------------------- */
function onThemeChange(name: string) {
  installation.value.choose(name)
}

function onModeChange(modeChoice: string) {
  installation.value.choose(`${MODE}:${modeChoice as Mode}`)
}

function onInterfaceScaleChange(size: number | null) {
  if (size !== null) {
    installation.value.choose(`${INTERFACE_SCALE}:${size}`)
  }
}

function onTextScaleChange(size: number | null) {
  if (size !== null) {
    installation.value.choose(`${TEXT_SCALE}:${size}`)
  }
}
</script>

<template>
  <section class="settings__group" :aria-label="words.window">
    <h2 class="settings__heading">{{ words.window }}</h2>

    <SettingRow v-slot="{ labelledBy }" at="theme" :name="words.theme" :detail="words.themeDetail">
      <Select
        :model-value="installation.applied.value"
        :choices="themes"
        :name="words.theme"
        :aria-labelledby="labelledBy"
        class="settings__choice"
        @update:model-value="onThemeChange"
      />
    </SettingRow>

    <SettingRow
      v-slot="{ labelledBy }"
      at="mode"
      :name="words.mode"
      :detail="installation.isPinned.value ? words.pinned : words.modeDetail"
    >
      <SegmentedControl
        :model-value="installation.mode.value"
        :choices="modes"
        :disabled="installation.isPinned.value"
        :aria-labelledby="labelledBy"
        @update:model-value="onModeChange"
      />
    </SettingRow>

    <SettingRow
      v-slot="{ labelledBy }"
      at="interface"
      :name="words.interfaceScale"
      :detail="words.interfaceScaleDetail"
    >
      <NumberField
        :model-value="installation.sizes.value.interfaceScale"
        :min="bounds.interfaceScale.min"
        :max="bounds.interfaceScale.max"
        :step="STEP"
        :aria-labelledby="labelledBy"
        class="settings__number"
        @update:model-value="onInterfaceScaleChange"
      />
    </SettingRow>

    <SettingRow
      v-slot="{ labelledBy }"
      at="text"
      :name="words.textScale"
      :detail="words.textScaleDetail"
    >
      <NumberField
        :model-value="installation.sizes.value.textScale"
        :min="bounds.textScale.min"
        :max="bounds.textScale.max"
        :step="STEP"
        :aria-labelledby="labelledBy"
        class="settings__number"
        @update:model-value="onTextScaleChange"
      />
    </SettingRow>
  </section>

  <NamingSettings :state="props.state" />
</template>

<style scoped>
.settings__group {
  inline-size: 100%;
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
  text-align: end;
}

.settings__choice {
  inline-size: var(--settings-choice, 16rem);
}
</style>
