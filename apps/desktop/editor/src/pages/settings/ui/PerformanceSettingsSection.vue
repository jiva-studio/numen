<script setup lang="ts">
/**
 * The performance profile: how much of the machine the core's CPU-bound work takes.
 */
import { computed } from 'vue'
import { SegmentedControl } from '@numen/ui'
import SettingRow from './setting-row/SettingRow.vue'
import type { SettingsTabState } from '../types'
import paths from '../paths.json'
import { createSettingFields } from '@/entities/settings'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ state: SettingsTabState }>()

/* --------------------------------- State ---------------------------------- */
const fields = createSettingFields(() => props.state.installation)

const profiles = [
  { id: 'eco', text: words.eco },
  { id: 'balanced', text: words.balanced },
  { id: 'maximum', text: words.maximum },
]

const currentProfile = computed<string>(() => {
  const profile = fields.getString(paths.performanceProfile)
  return profiles.some((one) => one.id === profile) ? profile : 'balanced'
})

const profileDetail = computed<string>(() => {
  switch (currentProfile.value) {
    case 'eco':
      return words.ecoDetail
    case 'maximum':
      return words.maximumDetail
    default:
      return words.balancedDetail
  }
})

/* -------------------------------- Handlers -------------------------------- */
function onProfileChange(choice: string): void {
  fields.setValue(paths.performanceProfile, choice)
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
</style>
