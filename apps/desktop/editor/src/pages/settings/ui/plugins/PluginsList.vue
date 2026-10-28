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
  matchesPluginQuery,
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

const CATEGORY_ORDER = ['agents', 'editors', 'engines', 'readers', 'tools', 'views']

interface FlatPluginItem {
  category: string
  categoryLabel: string
  plugin: PluginInfo
}

const parsedQuery = computed(() => parsePluginQuery(searchQuery.value))

const filterContext: PluginFilterContext = {
  isEnabled: isPluginEnabled,
  getCategory: getPluginCategory,
  getCategoryLabel: formatCategoryLabel,
}

const showsSystem = computed(() => matchesSystemRow(parsedQuery.value))

const flatPlugins = computed<readonly FlatPluginItem[]>(() => {
  const query = parsedQuery.value
  const matching =
    query.qualifiers.length > 0 || query.terms.length > 0
      ? props.plugins.filter((plugin) => matchesPluginQuery(plugin, query, filterContext))
      : props.plugins

  const items: FlatPluginItem[] = matching.map((plugin) => {
    const category = getPluginCategory(plugin)
    return {
      category,
      categoryLabel: formatCategoryLabel(category),
      plugin,
    }
  })

  return items.sort((a, b) => {
    const idxA = CATEGORY_ORDER.indexOf(a.category)
    const idxB = CATEGORY_ORDER.indexOf(b.category)
    if (idxA >= 0 && idxB >= 0 && idxA !== idxB) return idxA - idxB
    if (idxA >= 0 && idxB < 0) return -1
    if (idxB >= 0 && idxA < 0) return 1
    const catComp = a.category.localeCompare(b.category)
    if (catComp !== 0) return catComp
    return a.plugin.name.localeCompare(b.plugin.name)
  })
})

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
function getPluginCategory(plugin: PluginInfo): string {
  if ('category' in plugin && typeof plugin.category === 'string' && plugin.category.length > 0) {
    return plugin.category.toLowerCase()
  }
  const parts = plugin.id.split('.')
  if (parts.length >= 3 && parts[0] === 'md' && parts[1] === 'numen' && parts[2]) {
    return parts[2].toLowerCase()
  }
  if (parts.length >= 2 && parts[0]) {
    return parts[0].toLowerCase()
  }
  return 'other'
}

function formatCategoryLabel(category: string): string {
  const map: Record<string, string> = {
    agents: words.categoryAgents,
    editors: words.categoryEditors,
    engines: words.categoryEngines,
    readers: words.categoryReaders,
    tools: words.categoryTools,
    views: words.categoryViews,
  }
  return map[category] ?? category.charAt(0).toUpperCase() + category.slice(1)
}

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
