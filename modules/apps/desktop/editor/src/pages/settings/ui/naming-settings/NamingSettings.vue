<script setup lang="ts">
/**
 * Settings for how notes are named.
 */
import { computed } from 'vue'
import { Switch } from '@numen/ui'
import SettingRow from '../setting-row/SettingRow.vue'
import type { SettingsTabState } from '../../types'
import { WORDS as words } from '../../words'

// --- Props & Emits ---
const props = defineProps<{ state: SettingsTabState }>()

// --- State ---
const installation = computed(() => props.state.installation)
</script>

<template>
  <section class="settings__group" :aria-label="words.naming">
    <h2 class="settings__heading">{{ words.naming }}</h2>

    <SettingRow
      v-slot="{ labelledBy }"
      at="syncing"
      :name="words.syncing"
      :detail="words.syncingDetail"
    >
      <Switch v-model="installation.isSyncing.value" :aria-labelledby="labelledBy" />
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
