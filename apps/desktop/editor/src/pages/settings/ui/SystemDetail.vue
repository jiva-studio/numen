<script setup lang="ts">
/**
 * Dedicated settings view for system settings: window appearance, theme, scale,
 * filename syncing, and the path of the numen.json configuration file.
 */
import { computed } from 'vue'
import { Button } from '@numen/ui'
import WindowSettingsSection from './WindowSettingsSection.vue'
import type { SettingsTabState } from '../types'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  state: SettingsTabState
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (e: 'back'): void
}>()

/* --------------------------------- State ---------------------------------- */
const installation = computed(() => props.state.installation)

/* -------------------------------- Handlers -------------------------------- */
function onBack(): void {
  emit('back')
}
</script>

<template>
  <div class="system-detail">
    <Button variant="ghost" size="small" class="system-detail__back" @click="onBack">
      &larr; {{ words.backToPlugins }}
    </Button>

    <div class="system-detail__where">
      <p class="system-detail__file">{{ installation.file.value || words.file }}</p>
    </div>

    <WindowSettingsSection :state="props.state" />
  </div>
</template>

<style scoped>
.system-detail {
  display: flex;
  flex-direction: column;
  inline-size: 100%;
  max-inline-size: var(--settings-measure);
  margin-inline: auto;
}

.system-detail__back {
  align-self: flex-start;
  margin-block-end: 0.75rem;
  margin-inline-start: -0.5rem;
  color: var(--numen-hushed);
}

.system-detail__where {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--numen-panel-gap);
  margin-block-end: var(--settings-apart);
  padding-block-end: 1rem;
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
}

.system-detail__file {
  min-inline-size: 0;
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  overflow-wrap: anywhere;
}
</style>
