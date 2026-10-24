<script setup lang="ts">
/**
 * Dedicated settings view for system settings: window appearance, theme, scale,
 * filename syncing, and direct access to numen.json configuration file.
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

function onOpenFile(): void {
  installation.value.openFile()
}
</script>

<template>
  <div class="system-detail">
    <div class="system-detail__navigation">
      <Button variant="outline" size="small" class="system-detail__back" @click="onBack">
        &larr; {{ words.backToPlugins }}
      </Button>
    </div>

    <div class="system-detail__where">
      <p class="system-detail__file">{{ installation.file.value || words.file }}</p>
      <Button variant="outline" size="small" @click="onOpenFile">
        {{ words.opens }}
      </Button>
    </div>

    <WindowSettingsSection :state="props.state" />
  </div>
</template>

<style scoped>
.system-detail {
  display: flex;
  flex-direction: column;
  inline-size: 100%;
  max-inline-size: var(--settings-measure, 54rem);
  margin-inline: auto;
}

.system-detail__navigation {
  margin-block-end: 1rem;
}

.system-detail__where {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--numen-panel-gap, 0.75rem);
  margin-block-end: var(--settings-apart, 1.75rem);
  padding-block-end: 1rem;
  border-block-end: var(--numen-stroke, 1px) solid var(--numen-rule);
}

.system-detail__file {
  min-inline-size: 0;
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  overflow-wrap: anywhere;
}
</style>
