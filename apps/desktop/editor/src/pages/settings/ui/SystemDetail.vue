<script setup lang="ts">
/**
 * Dedicated settings view for system settings: every setting the core declares for the owner `app`,
 * drawn by the renderer of the plugins, and the path of the numen.json configuration file.
 */
import { computed } from 'vue'
import { Button } from '@numen/ui'
import PluginFields from './plugins/PluginFields.vue'
import type { PluginInfo } from '@/entities/settings'
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

/** The owner the core declares the settings of the shell under. */
const owner = computed<PluginInfo>(() => ({
  id: 'app',
  name: 'app',
  version: '',
  description: '',
  isActive: true,
  capabilities: [],
  secrets: [],
  hasFrontend: false,
  hasBackend: false,
  settings: installation.value.app?.value ?? [],
}))

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

    <PluginFields :state="props.state" :plugin="owner" />
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
