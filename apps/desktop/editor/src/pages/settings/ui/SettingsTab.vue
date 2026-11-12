<script setup lang="ts">
/**
 * The settings tab: every numen.json setting the user can change, grouped by
 * the part of the application it governs.
 */
import { computed } from 'vue'
import type { SettingsTabState } from '../types'
import { groupManagedHealth } from '../lib/pluginHealth'
import { useSettingsTab } from '../model/useSettingsTab'
import SystemDetail from './SystemDetail.vue'
import ConnectionsPage from './connections/ConnectionsPage.vue'
import { AvailablePluginDetail, PluginsList, PluginDetail } from './plugins'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ state: SettingsTabState }>()

/* --------------------------------- State ---------------------------------- */
const { installation, initialSearchQuery, markedSetting } = props.state
const health = computed(() =>
  groupManagedHealth(installation.plugin.health.items.value, installation.plugin.health.canRunCommand),
)
const {
  selectedPluginId,
  plugins,
  uninstalled,
  selectedPlugin,
  selectPlugin,
  updateQuery,
  goBack,
} = useSettingsTab(props.state)
</script>

<template>
  <div class="settings">
    <div class="settings__page">
      <div class="settings__main">
        <SystemDetail
          v-if="selectedPluginId === 'system' || selectedPluginId === 'appearance'"
          :key="selectedPluginId"
          :section="selectedPluginId"
          :settings="installation.settings"
          :manager="installation.plugin"
          :connections="installation.connections"
          @back="goBack"
          @open-connections="selectPlugin('connections')"
        />
        <ConnectionsPage
          v-else-if="selectedPluginId === 'connections'"
          :connections="installation.connections"
          @back="goBack"
        />
        <PluginDetail
          v-else-if="selectedPlugin"
          :key="selectedPlugin.id"
          :settings="installation.settings"
          :manager="installation.plugin"
          :connections="installation.connections"
          :plugin="selectedPlugin"
          :marked-setting="markedSetting"
          @back="goBack"
          @open-connections="selectPlugin('connections')"
        />
        <AvailablePluginDetail v-else-if="uninstalled" :plugin="uninstalled" @back="goBack" />
        <PluginsList
          v-else
          :settings="installation.settings"
          :catalog="installation.plugin.catalog.value"
          :query="initialSearchQuery"
          :plugins="plugins"
          :health="health"
          @select-plugin="selectPlugin"
          @update:query="updateQuery"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings {
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

.settings__main {
  container: settings / inline-size;
  inline-size: 100%;
  max-inline-size: var(--settings-measure);
  margin-inline: auto;
}
</style>
