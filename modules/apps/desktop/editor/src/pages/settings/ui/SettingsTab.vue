<script setup lang="ts">
/**
 * The settings tab: everything in numen.json a person can change, grouped by
 * the part of the application it governs.
 */
import { computed } from 'vue'
import { Button } from '@numen/ui'
import type { SettingsTabState } from '../types'
import WindowSettingsSection from './WindowSettingsSection.vue'
import MediaSettingsSection from './MediaSettingsSection.vue'
import AgentSettingsSection from './AgentSettingsSection.vue'
import { WORDS as words } from '../words'

// --- Props & Emits ---
const props = defineProps<{ state: SettingsTabState }>()

// --- State ---
const installation = computed(() => props.state.installation)

// --- Handlers ---
function onOpenFile() {
  installation.value.openFile()
}

// --- Helpers ---
</script>

<template>
  <div class="settings">
    <div class="settings__page">
      <div class="settings__where">
        <p class="settings__file">{{ installation.file.value || words.file }}</p>
        <Button variant="outline" size="small" @click="onOpenFile">
          {{ words.opens }}
        </Button>
      </div>

      <WindowSettingsSection :state="props.state" />
      <MediaSettingsSection :state="props.state" />
      <AgentSettingsSection :state="props.state" />
    </div>
  </div>
</template>

<style scoped>
.settings {
  --settings-measure: 54rem;
  --settings-apart: 1.75rem;
  --settings-near: 0.375rem;
  --settings-value: 6rem;
  --settings-choice: 18rem;
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
}

.settings__page {
  flex: 1;
  min-block-size: 0;
  overflow-y: auto;
  padding: var(--numen-gutter);
}

.settings__where {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--numen-panel-gap);
  max-inline-size: var(--settings-measure);
  margin: 0 auto var(--settings-apart);
}

.settings__file {
  min-inline-size: 0;
  margin: 0;
  color: var(--numen-hushed);
  overflow-wrap: anywhere;
}
</style>
