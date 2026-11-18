<script setup lang="ts">
/**
 * The Logs tab view: a top bar with level filter pills and message search input, and the list of
 * structured log rows.
 */
import type { LogsView } from '../model/logs'
import type { LevelFilter } from '../types'
import { WORDS as words } from '../words'
import LogEntryRow from './LogEntryRow.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  state: LogsView
}>()

/* --------------------------------- State ---------------------------------- */
const levels: readonly { readonly id: LevelFilter; readonly label: string }[] = [
  { id: 'all', label: words.levelAll },
  { id: 'debug', label: words.levelDebug },
  { id: 'info', label: words.levelInfo },
  { id: 'warning', label: words.levelWarning },
  { id: 'error', label: words.levelError },
]

/* -------------------------------- Handlers -------------------------------- */
function onSelectLevel(level: LevelFilter) {
  props.state.setLevelFilter(level)
}

function onSearchInput(event: Event) {
  const target = event.target as HTMLInputElement | null
  props.state.setSearchQuery(target?.value ?? '')
}

function onToggleRow(id: number) {
  props.state.toggleExpanded(id)
}
</script>

<template>
  <div class="logs-tab">
    <header class="logs-tab__toolbar">
      <div class="logs-tab__levels" role="group" :aria-label="words.title">
        <button
          v-for="item in levels"
          :key="item.id"
          type="button"
          class="logs-tab__level-btn"
          :class="{ 'logs-tab__level-btn--active': state.levelFilter.value === item.id }"
          :aria-pressed="state.levelFilter.value === item.id"
          @click="onSelectLevel(item.id)"
        >
          {{ item.label }}
        </button>
      </div>

      <input
        type="search"
        class="logs-tab__search"
        :value="state.searchQuery.value"
        :placeholder="words.searchPlaceholder"
        :aria-label="words.searchLabel"
        @input="onSearchInput"
      />
    </header>

    <p v-if="state.phase.value === 'unavailable'" class="logs-tab__notice">
      {{ words.unavailable }}
    </p>
    <p v-else-if="state.phase.value === 'loading'" class="logs-tab__notice">
      {{ words.loading }}
    </p>
    <p v-else-if="state.visibleRows.value.length === 0" class="logs-tab__notice">
      {{ words.empty }}
    </p>
    <ul v-else class="logs-tab__list">
      <LogEntryRow
        v-for="row in state.visibleRows.value"
        :key="row.id"
        :row="row"
        :is-expanded="state.expandedIds.value.has(row.id)"
        @toggle="onToggleRow"
      />
    </ul>
  </div>
</template>
