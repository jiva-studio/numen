<script setup lang="ts">
/**
 * The tab of a saved search note: the title, the status line and a failed last run; the accordion that
 * edits the query; the query and the rerank text; the problems, the warnings and the results.
 */
import { computed } from 'vue'
import { WORDS as words } from '../../words'
import type { SavedSearchTabState, SearchChange } from '../model/tab'
import QueryEditor from './QueryEditor.vue'
import QueryPanel from './QueryPanel.vue'
import ResultList from './ResultList.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  state: SavedSearchTabState
}>()

/* --------------------------------- State ---------------------------------- */
const view = computed(() => props.state.view.value)
const failure = computed(() => props.state.failure.value)

/* -------------------------------- Handlers -------------------------------- */
function onSave(change: SearchChange) {
  props.state.actions.saveSearch(change).catch(() => undefined)
}

function onToggle() {
  props.state.actions.setEditing(!props.state.isEditing.value)
}

function onOpen(index: number) {
  props.state.actions.openResult(index)
}
</script>

<template>
  <section class="saved-search">
    <template v-if="view">
      <header class="saved-search__header">
        <h2 class="saved-search__title">{{ view.title }}</h2>
        <p role="status" class="saved-search__status">{{ view.status }}</p>
        <p v-if="view.failedRun" class="saved-search__failed">{{ view.failedRun }}</p>
      </header>
      <QueryEditor
        :is-open="state.isEditing.value"
        :query="view.query"
        :rerank="view.rerank"
        :limit="view.limit"
        :problems="state.saveProblems.value"
        :is-saving="state.isSaving.value"
        @toggle="onToggle"
        @save="onSave"
      />
      <QueryPanel :query="view.query" :rerank="view.rerank" :rerank-label="words.rerankLabel" />
      <p v-for="problem in view.problems" :key="problem" role="alert" class="saved-search__problem">
        {{ problem }}
      </p>
      <p
        v-for="warning in view.warnings"
        :key="warning"
        role="status"
        class="saved-search__warning"
      >
        {{ warning }}
      </p>
      <ResultList
        v-if="view.items.length > 0"
        :items="view.items"
        :label="words.results"
        @open="onOpen"
      />
      <p v-else-if="view.emptyText" class="saved-search__empty">{{ view.emptyText }}</p>
    </template>
    <p v-if="failure" role="alert" class="saved-search__failure">{{ failure }}</p>
  </section>
</template>
