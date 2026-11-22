<script setup lang="ts">
/**
 * The System, the Appearance or the Editor page: the toolbar and the settings the core declares for
 * the owner `app` that belong to the page, drawn by the renderer of the plugins, in the islands of
 * the groups the core declares. Appearance holds
 * the `appearance.*` settings and Editor the `editor.*` ones; System holds the rest, the settings of connection rows below them as
 * the plugin pages draw them, and the path of the numen.json configuration file.
 */
import { computed } from 'vue'
import { SettingsPage } from '@numen/plugin-manager'
import PluginSettings from './plugins/PluginSettings.vue'
import PluginConnectionSections from './plugins/PluginConnectionSections.vue'
import type { PluginInfo } from '@/entities/settings'
import type { InstallationConnections, InstallationPlugin, InstallationSettings } from '../types'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  settings: InstallationSettings
  manager: InstallationPlugin
  connections: InstallationConnections
  section: 'system' | 'appearance' | 'editor' | 'search'
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (e: 'back'): void
  (e: 'open-connections'): void
}>()

/* --------------------------------- State ---------------------------------- */
const SECTION_PREFIXES = {
  appearance: 'appearance.',
  editor: 'editor.',
  search: 'search.',
} as const

const getSectionOf = (key: string): 'system' | 'appearance' | 'editor' | 'search' =>
  key.startsWith(SECTION_PREFIXES.appearance)
    ? 'appearance'
    : key.startsWith(SECTION_PREFIXES.editor)
      ? 'editor'
      : key.startsWith(SECTION_PREFIXES.search)
        ? 'search'
        : 'system'

const TITLES = {
  system: words.systemSettings,
  appearance: words.appearanceSettings,
  editor: words.editorSettings,
  search: words.searchSettings,
} as const

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
  settings: props.manager.app.value.filter((one) => getSectionOf(one.key) === props.section),
  settingGroups: props.manager.appGroups.value,
}))

/* -------------------------------- Handlers -------------------------------- */
function onBack(): void {
  emit('back')
}
</script>

<template>
  <SettingsPage :path="[words.settings, TITLES[props.section]]" @back="onBack" @open="onBack">
    <div class="system-detail">
      <PluginConnectionSections
        v-if="props.section === 'search'"
        :settings="props.settings"
        :host="props.manager"
        :connections="props.connections"
        :plugin="owner"
        @open-connections="emit('open-connections')"
      />

      <PluginSettings
        :settings="props.settings"
        :host="props.manager"
        :plugin="owner"
        :marked="[]"
        :focused="null"
      />

      <PluginConnectionSections
        v-if="props.section === 'system'"
        :settings="props.settings"
        :host="props.manager"
        :connections="props.connections"
        :plugin="owner"
        @open-connections="emit('open-connections')"
      />

      <p v-if="props.section === 'system'" class="system-detail__file">
        {{ props.settings.file.value || words.file }}
      </p>
    </div>
  </SettingsPage>
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
