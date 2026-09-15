<script setup lang="ts">
/**
 * Settings for the day a review stands in.
 */
import { computed } from 'vue'
import { TimeField } from '@numen/ui'
import SettingRow from '../setting-row/SettingRow.vue'
import type { SettingsTabState } from '../../types'
import { WORDS as words } from '../../words'

// --- Props & Emits ---
const props = defineProps<{ state: SettingsTabState }>()

// --- State ---
const installation = computed(() => props.state.installation)

// --- Handlers ---
function onDayStartsChange(hour: string) {
  installation.value.chooseDayStarts(hour)
}
</script>

<template>
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
        @settle="onDayStartsChange"
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
</style>
