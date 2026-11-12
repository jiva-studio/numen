<script setup lang="ts">
/**
 * Registry list of plugins with search filtering and category prefix, plus navigation to the
 * plugin pages and the system settings.
 */
import { computed } from 'vue'
import PluginRow from './PluginRow.vue'
import SystemRow from './SystemRow.vue'
import {
  getFilteredPluginItems,
  matchesConnectionsRow,
  matchesSystemRow,
  parsePluginQuery,
  type PluginFilterContext,
} from './filter'
import type { ManagedPlugin } from './plugins'
import { PLUGIN_WORDS as words } from './words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  plugins: readonly ManagedPlugin[]
  enabled: Readonly<Record<string, boolean>>
}>()

const searchQuery = defineModel<string>('query', { default: '' })

const emit = defineEmits<{
  (e: 'select-plugin', pluginId: string): void
  (e: 'toggle-enabled', pluginId: string, isEnabled: boolean): void
}>()

/* --------------------------------- State ---------------------------------- */
const filterContext: PluginFilterContext = { isEnabled: isPluginEnabled }

const parsedQuery = computed(() => parsePluginQuery(searchQuery.value))

const showsSystem = computed(() => matchesSystemRow(parsedQuery.value))

const showsConnections = computed(() => matchesConnectionsRow(parsedQuery.value))

const flatPlugins = computed(() =>
  getFilteredPluginItems(props.plugins, parsedQuery.value, filterContext),
)

/* -------------------------------- Handlers -------------------------------- */
function onSelectPlugin(pluginId: string): void {
  emit('select-plugin', pluginId)
}

function onSelectSystem(): void {
  emit('select-plugin', 'system')
}

function onSelectConnections(): void {
  emit('select-plugin', 'connections')
}

function onTogglePlugin(pluginId: string, isEnabled: boolean): void {
  emit('toggle-enabled', pluginId, isEnabled)
}

/* -------------------------------- Helpers --------------------------------- */
function isPluginEnabled(pluginId: string): boolean {
  return props.enabled[pluginId] ?? true
}
</script>

<template>
  <div class="plugins-list settings__plugins">
    <div class="plugins-list__search">
      <input
        v-model="searchQuery"
        type="search"
        :placeholder="words.searchPlugins"
        class="plugins-list__search-input"
        aria-label="Search plugins"
      />
    </div>

    <div class="plugins-list__items">
      <SystemRow v-if="showsSystem" @select="onSelectSystem" />
      <SystemRow
        v-if="showsConnections"
        class="connections-row"
        :name="words.connections"
        :detail="words.connectionsDetail"
        @select="onSelectConnections"
      />
      <PluginRow
        v-for="item in flatPlugins"
        :key="item.plugin.id"
        :plugin="item.plugin"
        :category-prefix="item.categoryLabel"
        :is-enabled="isPluginEnabled(item.plugin.id)"
        @select-plugin="onSelectPlugin(item.plugin.id)"
        @toggle-enabled="(isEnabled) => onTogglePlugin(item.plugin.id, isEnabled)"
      />
    </div>

    <div v-if="props.plugins.length === 0" class="plugins-list__empty">
      {{ words.noPlugins }}
    </div>
    <div
      v-else-if="flatPlugins.length === 0 && !showsSystem && !showsConnections"
      class="plugins-list__empty"
    >
      {{ words.noPluginsFound }}
    </div>
  </div>
</template>

<style scoped>
.plugins-list {
  --list-gap: 0.5rem;
  --list-spacing: 1rem;
  --search-pad-block: 0.5rem;
  --search-pad-inline: 0.75rem;
  --items-gap: 0.125rem;
  display: flex;
  flex-direction: column;
}

.plugins-list__search {
  display: flex;
  gap: var(--list-gap);
  margin-block-end: var(--list-spacing);
}

.plugins-list__search-input {
  box-sizing: border-box;
  flex: 1 1 auto;
  min-inline-size: 0;
  padding: var(--search-pad-block) var(--search-pad-inline);
  background-color: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius-tight);
  color: var(--numen-ink);
  font-family: inherit;
  font-size: var(--numen-text-2);
}

.plugins-list__search-input:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.plugins-list__items {
  display: flex;
  flex-direction: column;
  gap: var(--items-gap);
}

.plugins-list__empty {
  padding-block: var(--numen-gutter);
  color: var(--numen-hushed);
  font-style: italic;
}
</style>
