<script setup lang="ts">
/**
 * Dedicated settings view for system settings: every setting the core declares for the owner `app`,
 * drawn by the renderer of the plugins, and the path of the numen.json configuration file.
 */
import { computed } from 'vue'
import { Button } from '@numen/ui'
import { PluginSettingsCard } from '@numen/plugin-manager'
import PluginFields from './plugins/PluginFields.vue'
import type { PluginInfo } from '@/entities/settings'
import type { InstallationConnections, InstallationPlugin, InstallationSettings } from '../types'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  settings: InstallationSettings
  manager: InstallationPlugin
  connections: InstallationConnections
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (e: 'back'): void
  (e: 'open-connections'): void
}>()

/* --------------------------------- State ---------------------------------- */
/** The owner the core declares the settings of the shell under. */
const owner = computed<PluginInfo>(() => ({
  id: 'app',
  name: 'app',
  version: '',
  description: '',
  isActive: true,
  capabilities: [],
  hasFrontend: false,
  hasBackend: false,
  settings: props.manager.app.value,
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

    <PluginSettingsCard>
      <PluginFields
        :settings="props.settings"
        :host="props.manager"
        :connections="props.connections"
        :plugin="owner"
        @open-connections="emit('open-connections')"
      />
    </PluginSettingsCard>

    <div class="system-detail__where">
      <p class="system-detail__file">{{ props.settings.file.value || words.file }}</p>
    </div>
  </div>
</template>

<style scoped>
.system-detail {
  --max-width: 54rem;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  max-inline-size: var(--max-width);
  margin-inline: auto;
}

.system-detail__back {
  --margin-end: 0.75rem;
  align-self: flex-start;
  margin-block-end: var(--margin-end);
}

.system-detail__where {
  --margin-start: 0.75rem;
  margin-block-start: var(--margin-start);
}

.system-detail__file {
  min-inline-size: 0;
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  overflow-wrap: anywhere;
}
</style>
