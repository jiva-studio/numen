<script setup lang="ts">
/**
 * Registry list of plugins: the System and Connections pages as two tiles, a banner for
 * each stopped plugin, then the plugin search and the plugin rows.
 */
import { computed, ref } from 'vue'
import { ChevronRight, Search, TriangleAlert, X } from '@lucide/vue'
import { getCategoryIcon } from './categoryIcons'
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
const searchInput = ref<HTMLInputElement | null>(null)
const filterContext: PluginFilterContext = { isEnabled: isPluginEnabled }

const parsedQuery = computed(() => parsePluginQuery(searchQuery.value))

const tiles = [
  {
    page: 'system',
    name: words.systemSettings,
    detail: words.systemSettingsDetail,
  },
  {
    page: 'connections',
    name: words.connections,
    detail: words.connectionsDetail,
  },
]

const shownPlugins = computed(() =>
  getFilteredPluginItems(props.plugins, parsedQuery.value, filterContext),
)

const stoppedPlugins = computed(() =>
  props.plugins.flatMap((plugin) => {
    const item = getStartItem(props.health?.[plugin.id] ?? [])
    return item ? [{ plugin, item }] : []
  }),
)

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
</script>

<template>
  <div class="plugins-list settings__plugins">
    <div class="plugins-list__tiles">
      <button
        v-for="tile in tiles"
        :key="tile.page"
        type="button"
        class="plugins-list__tile"
        :class="`${tile.page}-row`"
        @click="onSelectPlugin(tile.page)"
      >
        <span class="plugins-list__glyph" aria-hidden="true">
          <component :is="getCategoryIcon(tile.page)" :size="20" :stroke-width="1.75" />
        </span>
        <span class="plugins-list__tile-text">
          <span class="plugins-list__tile-name">{{ tile.name }}</span>
          <span class="plugins-list__tile-detail">{{ tile.detail }}</span>
        </span>
      </button>
    </div>

    <div
      v-for="{ plugin, item } in stoppedPlugins"
      :key="`stopped-${plugin.id}`"
      class="plugins-list__alarm"
      role="button"
      tabindex="0"
      :aria-label="words.pluginStoppedBanner(plugin.name)"
      @click="onSelectPlugin(plugin.id)"
      @keydown.enter.prevent="onSelectPlugin(plugin.id)"
      @keydown.space.prevent="onSelectPlugin(plugin.id)"
    >
      <TriangleAlert
        class="plugins-list__alarm-icon"
        :size="20"
        :stroke-width="1.75"
        aria-hidden="true"
      />
      <span class="plugins-list__alarm-text">
        <span class="plugins-list__alarm-name">{{ words.pluginStoppedBanner(plugin.name) }}</span>
        {{ item.text }}
      </span>
      <ChevronRight class="plugins-list__alarm-chevron" :size="16" aria-hidden="true" />
    </div>

    <div v-if="props.plugins.length > 0" class="plugins-list__toolbar">
      <span class="plugins-list__field">
        <Search class="plugins-list__search-icon" :size="16" aria-hidden="true" />
        <input
          ref="searchInput"
          v-model="searchQuery"
          type="search"
          :placeholder="words.searchPlugins"
          class="plugins-list__search"
          aria-label="Search plugins"
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
    </div>

    <div v-if="shownPlugins.length > 0" class="plugins-list__box">
      <PluginRow
        v-for="item in shownPlugins"
        :key="item.plugin.id"
        :plugin="item.plugin"
        :is-enabled="isPluginEnabled(item.plugin.id)"
        :health="props.health?.[item.plugin.id] ?? []"
        @select-plugin="onSelectPlugin(item.plugin.id)"
        @toggle-enabled="(isEnabled) => onTogglePlugin(item.plugin.id, isEnabled)"
      />
    </div>

    <div v-if="props.plugins.length === 0" class="plugins-list__empty">
      {{ words.noPlugins }}
    </div>
    <div v-else-if="shownPlugins.length === 0" class="plugins-list__box plugins-list__none">
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

.plugins-list__toolbar {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-block-start: calc(1.5rem - 0.625rem);
}

.plugins-list__field {
  position: relative;
  flex: 1;
  min-inline-size: 0;
  display: flex;
  align-items: center;
}

.plugins-list__search-icon {
  position: absolute;
  inset-inline-start: 0.75rem;
  color: var(--numen-hushed);
  pointer-events: none;
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
  background-color: color-mix(in oklab, var(--numen-field-bg), var(--numen-ink) 8%);
  color: var(--numen-ink);
}

.plugins-list__search::-webkit-search-cancel-button {
  appearance: none;
}

.plugins-list__search {
  box-sizing: border-box;
  appearance: none;
  inline-size: 100%;
  block-size: var(--numen-action-size);
  padding: 0 2.25rem;
  background-color: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
  font-family: inherit;
  font-size: var(--numen-text-2);
}

.plugins-list__search:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.plugins-list__tiles {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.5rem;
}

@container (max-width: 34rem) {
  .plugins-list__tiles {
    grid-template-columns: minmax(0, 1fr);
  }
}

.plugins-list__tile {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: var(--settings-card-pad, 1rem);
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
  background-color: var(--numen-raised);
  color: var(--numen-ink);
  font-family: inherit;
  font-size: var(--numen-text-2);
  text-align: start;
  cursor: pointer;
  transition: border-color var(--numen-motion-hover) ease;
}

.plugins-list__tile:hover {
  border-color: color-mix(in oklab, var(--numen-rule), var(--numen-ink) 25%);
}

.plugins-list__tile:focus-visible {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.plugins-list__glyph {
  display: grid;
  place-items: center;
  flex: none;
  color: var(--numen-hushed);
}

.plugins-list__tile-text {
  display: flex;
  flex-direction: column;
  min-inline-size: 0;
}

.plugins-list__tile-name {
  font-weight: 500;
}

.plugins-list__tile-detail {
  color: var(--numen-hushed);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.plugins-list__alarm {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  box-sizing: border-box;
  min-block-size: var(--settings-row-min, 3.25rem);
  padding: var(--settings-card-pad, 1rem);
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
  background-color: var(--numen-raised);
  cursor: pointer;
  user-select: none;
  transition: background-color var(--numen-motion-hover) ease;
}

.plugins-list__alarm:hover {
  background-color: color-mix(in oklab, var(--numen-raised), var(--numen-ink) 5%);
}

.plugins-list__alarm:focus-visible {
  outline: var(--numen-stroke) solid var(--numen-accent);
  outline-offset: calc(-1 * var(--numen-stroke));
}

.plugins-list__alarm-chevron {
  flex: none;
  color: var(--numen-hushed);
}

.plugins-list__alarm-icon {
  flex: none;
  color: var(--numen-alarm);
}

.plugins-list__alarm-text {
  flex: 1;
  min-inline-size: 0;
  color: var(--numen-hushed);
}

.plugins-list__alarm-name {
  color: var(--numen-ink);
  font-weight: 500;
}

.plugins-list__box {
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
  background-color: var(--numen-raised);
  overflow: hidden;
}

.plugins-list__none {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.375rem;
  padding: 2.5rem var(--settings-card-pad, 1rem);
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
  padding-block: var(--numen-gutter);
  color: var(--numen-hushed);
}

@container (max-width: 34rem) {
  .plugins-list__toolbar {
    flex-wrap: wrap;
  }

  .plugins-list__field {
    flex-basis: 100%;
  }

}
</style>
