<script setup lang="ts">
/**
 * Registry list of installed plugins with search filtering and category prefix,
 * plus navigation to detail views and system settings.
 */
import { computed, ref, watch } from 'vue'
import PluginRow from './PluginRow.vue'
import SystemRow from './SystemRow.vue'
import type { SettingsTabState } from '../../types'
import {
  formatCategoryLabel,
  getFilteredPluginItems,
  getPluginCategory,
  matchesSystemRow,
  parsePluginQuery,
  type PluginFilterContext,
} from '../../lib/pluginFilter'
import { write, type PluginInfo } from '@/entities/settings'
import { WORDS as words } from '../../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  state: SettingsTabState
  plugins: readonly PluginInfo[]
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (e: 'select-plugin', pluginId: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const searchQuery = ref(props.state.initialSearchQuery?.value ?? '')

watch(
  () => props.state.initialSearchQuery?.value,
  (newQuery) => {
    if (newQuery !== undefined) {
      searchQuery.value = newQuery
    }
  },
)

const categoryLabels: Readonly<Record<string, string>> = {
  agents: words.categoryAgents,
  editors: words.categoryEditors,
  engines: words.categoryEngines,
  readers: words.categoryReaders,
  tools: words.categoryTools,
  views: words.categoryViews,
}

const filterContext: PluginFilterContext = {
  isEnabled: isPluginEnabled,
  getCategory: getPluginCategory,
  getCategoryLabel: (category) => formatCategoryLabel(category, categoryLabels),
}

const parsedQuery = computed(() => parsePluginQuery(searchQuery.value))

const showsSystem = computed(() => matchesSystemRow(parsedQuery.value))

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

function onTogglePlugin(pluginId: string, enabled: boolean): void {
  props.state.installation.write([{ at: ['plugins', pluginId, 'enabled'], value: write(enabled) }])
}

/* -------------------------------- Helpers --------------------------------- */
function isPluginEnabled(pluginId: string): boolean {
  const val = props.state.installation.getSetting(['plugins', pluginId, 'enabled'])
  return typeof val === 'boolean' ? val : true
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
      <PluginRow
        v-for="item in flatPlugins"
        :key="item.plugin.id"
        :plugin="item.plugin"
        :category-prefix="item.categoryLabel"
        :is-enabled="isPluginEnabled(item.plugin.id)"
        @select-plugin="onSelectPlugin(item.plugin.id)"
        @toggle-enabled="(val) => onTogglePlugin(item.plugin.id, val)"
      />
    </div>

    <div v-if="props.plugins.length === 0" class="plugins-list__empty">
      {{ words.noPlugins }}
    </div>
    <div v-else-if="flatPlugins.length === 0 && !showsSystem" class="plugins-list__empty">
      {{ words.noPluginsFound }}
    </div>
  </div>
</template>

<style scoped>
.plugins-list {
  display: flex;
  flex-direction: column;
}

.plugins-list__search {
  margin-block-end: 1rem;
}

.plugins-list__search-input {
  box-sizing: border-box;
  inline-size: 100%;
  padding: 0.5rem 0.75rem;
  background-color: var(--numen-field-bg, var(--numen-surface, transparent));
  border: var(--numen-stroke, 1px) solid var(--numen-rule);
  border-radius: var(--numen-radius-small, 4px);
  color: var(--numen-ink);
  font-family: inherit;
  font-size: var(--numen-text-2);
}

.plugins-list__search-input:focus {
  outline: var(--numen-stroke, 1px) solid var(--numen-accent, currentColor);
}

.plugins-list__items {
  display: flex;
  flex-direction: column;
}

.plugins-list__empty {
  padding-block: var(--numen-gutter, 1rem);
  color: var(--numen-hushed);
  font-style: italic;
}
</style>
