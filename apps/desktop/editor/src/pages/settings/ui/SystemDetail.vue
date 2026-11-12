<script setup lang="ts">
/**
 * Dedicated settings view for system settings: every setting the core declares for the owner `app`,
 * drawn by the renderer of the plugins, and the path of the numen.json configuration file.
 */
import { computed } from 'vue'
import { ChevronLeft, ChevronRight } from '@lucide/vue'
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
    <div class="system-detail__bar">
      <button
        type="button"
        class="system-detail__back"
        :aria-label="words.backToPlugins"
        @click="onBack"
      >
        <ChevronLeft :size="16" aria-hidden="true" />
      </button>
      <nav class="system-detail__crumbs">
        <button type="button" class="system-detail__crumb" @click="onBack">
          {{ words.settings }}
        </button>
        <ChevronRight :size="12" aria-hidden="true" />
        <span>{{ words.systemSettings }}</span>
      </nav>
    </div>

    <p class="system-detail__lede">{{ words.systemLede }}</p>

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
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 1rem;
}

.system-detail__bar {
  position: sticky;
  inset-block-start: calc(-1 * var(--numen-gutter));
  z-index: 5;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-block: calc(-1 * var(--numen-gutter)) 0;
  padding-block: var(--numen-gutter) 0.5rem;
  background-color: var(--numen-surface);
}

.system-detail__back {
  display: grid;
  place-items: center;
  inline-size: 1.75rem;
  block-size: 1.75rem;
  border: none;
  border-radius: var(--numen-radius);
  background: none;
  color: var(--numen-hushed);
  cursor: pointer;
}

.system-detail__back:hover,
.system-detail__crumb:hover {
  background-color: var(--numen-bubble-bg);
  color: var(--numen-ink);
}

.system-detail__crumbs {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  color: var(--numen-hushed);
}

.system-detail__crumb {
  padding: 0.125rem 0.25rem;
  border: none;
  border-radius: var(--numen-radius-tight);
  background: none;
  color: inherit;
  font-family: inherit;
  font-size: var(--numen-text-2);
  cursor: pointer;
}

.system-detail__lede {
  margin: 0;
  padding-inline: var(--settings-card-pad);
  color: var(--numen-hushed);
}

.system-detail__where {
  padding-inline: var(--settings-card-pad);
}

.system-detail__file {
  min-inline-size: 0;
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  overflow-wrap: anywhere;
}
</style>
