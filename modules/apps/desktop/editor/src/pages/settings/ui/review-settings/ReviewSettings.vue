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
function onDayStartTimeChange(hour: string) {
  installation.value.chooseDayStartTime(hour)
}
</script>

<template>
  <section class="settings__group" :aria-label="words.review">
    <h2 class="settings__heading">{{ words.review }}</h2>

    <SettingRow
      v-slot="{ labelledBy }"
      at="day-starts"
      :name="words.dayStartTime"
      :detail="words.dayStartTimeDetail"
    >
      <TimeField
        :model-value="installation.dayStartTime.value"
        :max="installation.latestDayStartTime.value"
        :aria-labelledby="labelledBy"
        class="settings__number"
        @commit="onDayStartTimeChange"
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
