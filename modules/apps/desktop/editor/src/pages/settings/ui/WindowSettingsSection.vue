<script setup lang="ts">
/**
 * Settings for the window appearance, naming/syncing, and day schedule.
 */
import { computed } from 'vue'
import { NumberField, SegmentedControl, Select, Switch, TimeField } from '@numen/ui'
import type { SelectChoice } from '@numen/ui'
import SettingRow from './setting-row/SettingRow.vue'
import type { SettingsTabState } from '../model/useSettingsTab'
import type { Mode } from '@/entities/settings'
import { INTERFACE_SCALE, MODE, TEXT_SCALE } from '@/features/settings-commands'
import { WORDS as words } from '../words'

// --- Props & Emits ---
const props = defineProps<{ state: SettingsTabState }>()

// --- State ---
const installation = computed(() => props.state.installation)

const bounds = computed(() => installation.value.bounds.value)

const modes = [
  { id: 'system', text: words.system },
  { id: 'light', text: words.light },
  { id: 'dark', text: words.dark },
]

const themes = computed<readonly SelectChoice[]>(() =>
  [...installation.value.themes.value]
    .sort(
      (one, other) =>
        Number(other.isBuiltIn) - Number(one.isBuiltIn),
    )
    .map((one) => ({
      id: one.name,
      text: one.title,
      group: one.isBuiltIn ? words.shipped : words.owned,
    })),
)

const STEP = 0.1

// --- Handlers ---
function onThemeChange(name: string) {
  installation.value.chooses(name)
}

function onModeChange(modeChoice: string) {
  installation.value.chooses(`${MODE}:${modeChoice as Mode}`)
}

function onInterfaceScaleChange(size: number | null) {
  if (size !== null) {
    installation.value.chooses(`${INTERFACE_SCALE}:${size}`)
  }
}

function onTextScaleChange(size: number | null) {
  if (size !== null) {
    installation.value.chooses(`${TEXT_SCALE}:${size}`)
  }
}

function onPartsChange(count: number | null) {
  if (count !== null) {
    installation.value.choosesParts(count)
  }
}

function onDayStartsChange(hour: string) {
  installation.value.choosesDayStarts(hour)
}

// --- Helpers ---
</script>

<template>
  <section class="settings__group" :aria-label="words.window">
    <h2 class="settings__heading">{{ words.window }}</h2>

    <SettingRow
      v-slot="{ labelledBy }"
      at="theme"
      :name="words.theme"
      :detail="words.themeDetail"
    >
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
      :detail="installation.pinned.value ? words.pinned : words.modeDetail"
    >
      <SegmentedControl
        :model-value="installation.mode.value"
        :choices="modes"
        :disabled="installation.pinned.value"
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
        :min="bounds.interfaceScale.least"
        :max="bounds.interfaceScale.most"
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
        :min="bounds.textScale.least"
        :max="bounds.textScale.most"
        :step="STEP"
        :aria-labelledby="labelledBy"
        class="settings__number"
        @update:model-value="onTextScaleChange"
      />
    </SettingRow>

    <SettingRow
      v-slot="{ labelledBy }"
      at="hanging"
      :name="words.hanging"
      :detail="words.hangingDetail"
    >
      <Switch v-model="installation.hangs.value" :aria-labelledby="labelledBy" />
    </SettingRow>

    <SettingRow
      v-slot="{ labelledBy }"
      at="parts"
      :name="words.parts"
      :detail="words.partsDetail"
    >
      <NumberField
        :model-value="installation.parts.value"
        :min="installation.partsBounds.value.least"
        :max="installation.partsBounds.value.most"
        :step="1"
        :aria-labelledby="labelledBy"
        class="settings__number"
        @update:model-value="onPartsChange"
      />
    </SettingRow>
  </section>

  <section class="settings__group" :aria-label="words.naming">
    <h2 class="settings__heading">{{ words.naming }}</h2>

    <SettingRow
      v-slot="{ labelledBy }"
      at="syncing"
      :name="words.syncing"
      :detail="words.syncingDetail"
    >
      <Switch v-model="installation.syncing.value" :aria-labelledby="labelledBy" />
    </SettingRow>
  </section>

  <section class="settings__group" :aria-label="words.review">
    <h2 class="settings__heading">{{ words.review }}</h2>

    <SettingRow
      v-slot="{ labelledBy }"
      at="day-starts"
      :name="words.dayStarts"
      :detail="words.dayStartsDetail"
    >
      <TimeField
        :model-value="installation.dayStarts.value"
        :max="installation.latestDayStarts.value"
        :aria-labelledby="labelledBy"
        class="settings__number"
        @settles="onDayStartsChange"
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
