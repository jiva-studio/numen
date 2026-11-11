<script setup lang="ts">
/**
 * The settings tab: every numen.json setting the user can change, grouped by
 * the part of the application it governs.
 */
import type { SettingsTabState } from '../types'
import { useSettingsTab } from '../model/useSettingsTab'
import SystemDetail from './SystemDetail.vue'
import SecretsPage from './secrets/SecretsPage.vue'
import { AvailablePluginDetail, PluginsList, PluginDetail } from './plugins'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ state: SettingsTabState }>()

/* --------------------------------- State ---------------------------------- */
const { installation, initialSearchQuery } = props.state
const { selectedPluginId, plugins, uninstalled, selectedPlugin, selectPlugin, updateQuery, goBack } =
  useSettingsTab(props.state)
</script>

<template>
  <div class="settings">
    <div class="settings__page">
      <div class="settings__main">
        <SystemDetail
          v-if="selectedPluginId === 'system'"
          :settings="installation.settings"
          :manager="installation.plugin"
          :choices="installation.secrets"
          @back="goBack"
          @open-secrets="selectPlugin('secrets')"
        />
        <SecretsPage
          v-else-if="selectedPluginId === 'secrets'"
          :secrets="installation.secrets"
          :plugins="plugins"
          @back="goBack"
        />
        <PluginDetail
          v-else-if="selectedPlugin"
          :key="selectedPlugin.id"
          :settings="installation.settings"
          :manager="installation.plugin"
          :choices="installation.secrets"
          :plugin="selectedPlugin"
          @back="goBack"
          @open-secrets="selectPlugin('secrets')"
        />
        <AvailablePluginDetail v-else-if="uninstalled" :plugin="uninstalled" @back="goBack" />
        <PluginsList
          v-else
          :settings="installation.settings"
          :catalog="installation.plugin.catalog.value"
          :query="initialSearchQuery"
          :plugins="plugins"
          @select-plugin="selectPlugin"
          @update:query="updateQuery"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings {
  --settings-measure: 54rem;
  --settings-apart: 1.75rem;
  --settings-near: 0.375rem;
  --settings-value: 6rem;
  --settings-choice: 16rem;
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
  container-type: inline-size;
  inline-size: 100%;
  max-inline-size: var(--settings-measure);
  margin-inline: auto;
}
</style>
