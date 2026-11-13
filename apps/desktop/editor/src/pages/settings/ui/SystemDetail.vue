<script setup lang="ts">
/**
 * The System or the Appearance page: the breadcrumb bar and the settings the core declares for the
 * owner `app` that belong to the page, drawn by the renderer of the plugins. Appearance holds the
 * `appearance.*` settings; System holds the rest and the path of the numen.json configuration file.
 */
import { computed } from 'vue'
import { BreadcrumbBar, PluginSettingsCard } from '@numen/plugin-manager'
import PluginFields from './plugins/PluginFields.vue'
import type { PluginInfo } from '@/entities/settings'
import type { InstallationPlugin, InstallationSettings } from '../types'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  settings: InstallationSettings
  manager: InstallationPlugin
  section: 'system' | 'appearance'
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (e: 'back'): void
}>()

/* --------------------------------- State ---------------------------------- */
const APPEARANCE = 'appearance.'

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
  settings: props.manager.app.value.filter(
    (one) => one.key.startsWith(APPEARANCE) === (props.section === 'appearance'),
  ),
}))

/* -------------------------------- Handlers -------------------------------- */
function onBack(): void {
  emit('back')
}
</script>

<template>
  <div class="system-detail">
    <BreadcrumbBar
      :current="props.section === 'appearance' ? words.appearanceSettings : words.systemSettings"
      @back="onBack"
    />

    <PluginSettingsCard>
      <PluginFields
        :settings="props.settings"
        :host="props.manager"
        :plugin="owner"
      />
    </PluginSettingsCard>

    <p v-if="props.section === 'system'" class="system-detail__file">{{ props.settings.file.value || words.file }}</p>
  </div>
</template>

<style scoped>
.system-detail {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 1rem;
}


.system-detail__file {
  min-inline-size: 0;
  margin: 0;
  padding-inline: var(--settings-card-pad);
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  overflow-wrap: anywhere;
}
</style>
