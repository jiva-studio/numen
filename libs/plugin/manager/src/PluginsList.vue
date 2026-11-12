<script setup lang="ts">
/**
 * The list of plugins: the System and Connections pages as two tiles, a card for each stopped
 * plugin, then the search field and the plugin rows that match it.
 */
import { computed, useTemplateRef } from 'vue'
import { ChevronRight, Search, TriangleAlert, X } from '@lucide/vue'
import NavigationRow from './NavigationRow.vue'
import PluginRow from './PluginRow.vue'
import { getFilteredPluginItems, parsePluginQuery, type PluginFilterContext } from './filter'
import type { PluginHealth } from './pluginInfo'
import type { ManagedPlugin } from './plugins'
import { getStartItem } from './state'
import { PLUGIN_WORDS as words } from './words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  plugins: readonly ManagedPlugin[]
  enabled: Readonly<Record<string, boolean>>
  health?: Readonly<Record<string, readonly PluginHealth[]>>
}>()

const searchQuery = defineModel<string>('query', { default: '' })

const emit = defineEmits<{
  (e: 'select-plugin', pluginId: string): void
  (e: 'toggle-enabled', pluginId: string, isEnabled: boolean): void
}>()

/* --------------------------------- State ---------------------------------- */
const searchInput = useTemplateRef<HTMLInputElement>('searchInput')
const filterContext: PluginFilterContext = { isEnabled: isPluginEnabled }
const parsedQuery = computed(() => parsePluginQuery(searchQuery.value))
const filteredPlugins = computed(() =>
  getFilteredPluginItems(props.plugins, parsedQuery.value, filterContext),
)
const stoppedPlugins = computed(() =>
  props.plugins.flatMap((plugin) => {
    const item = getStartItem(props.health?.[plugin.id] ?? [])
    return item ? [{ plugin, item }] : []
  }),
)
const TILES = [
  { page: 'system', name: words.systemSettings, detail: words.systemSettingsDetail },
  { page: 'connections', name: words.connections, detail: words.connectionsDetail },
]

/* -------------------------------- Handlers -------------------------------- */
function onClearSearch(): void {
  searchQuery.value = ''
  searchInput.value?.focus()
}

function onSelectPlugin(pluginId: string): void {
  emit('select-plugin', pluginId)
}

function onTogglePlugin(pluginId: string, isEnabled: boolean): void {
  emit('toggle-enabled', pluginId, isEnabled)
}

/* -------------------------------- Helpers --------------------------------- */
function isPluginEnabled(pluginId: string): boolean {
  return props.enabled[pluginId] ?? true
}

function getReasonId(pluginId: string): string {
  return `plugins-list-stopped-${pluginId.replaceAll('.', '-')}`
}
</script>

<template>
  <div class="plugins-list">
    <div class="plugins-list__tiles">
      <div v-for="tile in TILES" :key="tile.page" class="settings-card plugins-list__tile">
        <NavigationRow
          :name="tile.name"
          :detail="tile.detail"
          :category="tile.page"
          @select="onSelectPlugin(tile.page)"
        />
      </div>
    </div>

    <button
      v-for="{ plugin, item } in stoppedPlugins"
      :key="`stopped-${plugin.id}`"
      type="button"
      class="settings-card plugins-list__alarm"
      :aria-label="words.pluginStoppedBanner(plugin.name)"
      :aria-describedby="getReasonId(plugin.id)"
      @click="onSelectPlugin(plugin.id)"
    >
      <TriangleAlert class="plugins-list__alarm-icon" :size="20" aria-hidden="true" />
      <span class="plugins-list__alarm-text">
        <span class="plugins-list__alarm-name">{{ words.pluginStoppedBanner(plugin.name) }}</span>
        <span :id="getReasonId(plugin.id)">{{ item.text }}</span>
      </span>
      <ChevronRight class="plugins-list__chevron" :size="16" aria-hidden="true" />
    </button>

    <span v-if="props.plugins.length > 0" class="plugins-list__field">
      <Search class="plugins-list__search-icon" :size="16" aria-hidden="true" />
      <input
        ref="searchInput"
        v-model="searchQuery"
        type="search"
        :placeholder="words.searchPlugins"
        class="plugins-list__search"
        :aria-label="words.searchPluginsLabel"
        @keydown.esc="onClearSearch"
      />
      <button
        v-if="searchQuery !== ''"
        type="button"
        class="plugins-list__clear"
        :aria-label="words.clearSearch"
        :title="words.clearSearch"
        @click="onClearSearch"
      >
        <X :size="14" aria-hidden="true" />
      </button>
    </span>

    <div v-if="filteredPlugins.length > 0" class="settings-card plugins-list__box">
      <PluginRow
        v-for="item in filteredPlugins"
        :key="item.plugin.id"
        :plugin="item.plugin"
        :is-enabled="isPluginEnabled(item.plugin.id)"
        :health="props.health?.[item.plugin.id] ?? []"
        @select-plugin="onSelectPlugin(item.plugin.id)"
        @toggle-enabled="(isEnabled) => onTogglePlugin(item.plugin.id, isEnabled)"
      />
    </div>

    <p v-if="props.plugins.length === 0" class="plugins-list__empty">{{ words.noPlugins }}</p>
    <div v-else-if="filteredPlugins.length === 0" class="settings-card plugins-list__none">
      <Search class="plugins-list__none-icon" :size="24" :stroke-width="1.5" aria-hidden="true" />
      <span class="plugins-list__none-title">{{ words.noPluginsMatch(searchQuery.trim()) }}</span>
      <span class="plugins-list__none-hint">{{ words.noPluginsHint }}</span>
    </div>
  </div>
</template>

<style scoped>
.plugins-list {
  display: flex;
  flex-direction: column;
  gap: 0.625rem;
}

.plugins-list__tiles {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.5rem;
}

.plugins-list__alarm {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  inline-size: 100%;
  padding: var(--settings-card-pad);
  color: var(--numen-ink);
  font-family: inherit;
  font-size: var(--numen-text-2);
  text-align: start;
  cursor: pointer;
  transition: background-color var(--numen-motion-hover) ease;
}

.plugins-list__alarm:hover {
  background-color: var(--settings-hover-bg);
}

.plugins-list__alarm:focus-visible {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.plugins-list__alarm-icon {
  flex: none;
  color: var(--numen-alarm);
}

.plugins-list__alarm-text {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-inline-size: 0;
  color: var(--numen-hushed);
}

.plugins-list__alarm-name {
  color: var(--numen-ink);
  font-weight: 500;
}

.plugins-list__chevron {
  flex: none;
  color: var(--numen-hushed);
}

.plugins-list__field {
  position: relative;
  display: flex;
  align-items: center;
  margin-block-start: 0.875rem;
}

.plugins-list__search-icon {
  position: absolute;
  inset-inline-start: 0.75rem;
  color: var(--numen-hushed);
  pointer-events: none;
}

.plugins-list__search {
  box-sizing: border-box;
  appearance: none;
  inline-size: 100%;
  block-size: var(--numen-action-size);
  padding: 0 2.25rem;
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
  background-color: var(--numen-field-bg);
  color: var(--numen-ink);
  font-family: inherit;
  font-size: var(--numen-text-2);
}

.plugins-list__search::-webkit-search-cancel-button {
  appearance: none;
}

.plugins-list__search:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.plugins-list__clear {
  position: absolute;
  inset-inline-end: 0.375rem;
  display: inline-grid;
  place-items: center;
  inline-size: 1.5rem;
  block-size: 1.5rem;
  padding: 0;
  border: none;
  border-radius: var(--numen-radius-tight);
  background: none;
  color: var(--numen-hushed);
  cursor: pointer;
}

.plugins-list__clear:hover {
  background-color: var(--settings-hover-bg);
  color: var(--numen-ink);
}

.plugins-list__tile,
.plugins-list__box {
  padding: 0;
  overflow: hidden;
}

.plugins-list__none {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.375rem;
  padding: 2.5rem var(--settings-card-pad);
  text-align: center;
}

.plugins-list__none-icon {
  margin-block-end: 0.375rem;
  color: var(--numen-hushed);
}

.plugins-list__none-title {
  color: var(--numen-ink);
  font-weight: 500;
  overflow-wrap: anywhere;
}

.plugins-list__none-hint {
  max-inline-size: 40ch;
  color: var(--numen-hushed);
}

.plugins-list__empty {
  margin: 0;
  padding-block: var(--numen-gutter);
  color: var(--numen-hushed);
}

@container settings (max-width: 34rem) {
  .plugins-list__tiles {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
