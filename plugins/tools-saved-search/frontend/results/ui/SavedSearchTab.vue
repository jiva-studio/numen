<script setup lang="ts">
/**
 * The tab of a saved search note: the title, the status line, a failed last run and the button that
 * opens the editor of the query; the problems and the warnings; the filter and the sources of the
 * results; and the results.
 */
import { computed } from 'vue'
import { Button, createNamedIcon } from '@numen/ui'
import { WORDS as words } from '../../words'
import type { SavedSearchTabState, SearchChange } from '../model/tab'
import QueryDialog from './QueryDialog.vue'
import ResultList from './ResultList.vue'
import ResultsFilter from './ResultsFilter.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  state: SavedSearchTabState
}>()

/* --------------------------------- State ---------------------------------- */
const view = computed(() => props.state.view.value)
const items = computed(() => props.state.items.value)
const failure = computed(() => props.state.failure.value)
const EditIcon = createNamedIcon('pencil')

/* -------------------------------- Handlers -------------------------------- */
function onSave(change: SearchChange) {
  props.state.actions.saveSearch(change).catch(() => undefined)
}

function onEdit() {
  props.state.actions.setEditing(true)
}

function onCancel() {
  props.state.actions.setEditing(false)
}

function onOpen(index: number) {
  props.state.actions.openResult(index)
}

function onFilter(filter: string) {
  props.state.actions.setFilter(filter)
}

function onToggleSource(path: string, isShown: boolean) {
  props.state.actions.setSourceShown(path, isShown)
}
</script>

<template>
  <section class="saved-search">
    <template v-if="view">
      <header class="saved-search__header">
        <h2 class="saved-search__title">{{ view.title }}</h2>
        <p role="status" class="saved-search__status">{{ view.status }}</p>
        <p v-if="view.failedRun" class="saved-search__failed">{{ view.failedRun }}</p>
        <Button
          variant="ghost"
          size="icon-small"
          class="saved-search__edit"
          :aria-label="words.editQuery"
          aria-haspopup="dialog"
          @click="onEdit"
        >
          <EditIcon aria-hidden="true" />
        </Button>
      </header>
      <QueryDialog
        v-if="state.isEditing.value"
        :query="view.query"
        :problems="state.saveProblems.value"
        :functions="state.functions.value"
        :is-saving="state.isSaving.value"
        @save="onSave"
        @cancel="onCancel"
      />
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
      <ResultsFilter
        v-if="view.items.length > 0"
        :filter="state.filter.value"
        :sources="view.sources"
        :hidden="state.hiddenSources.value"
        @filter="onFilter"
        @toggle="onToggleSource"
      />
      <ResultList v-if="items.length > 0" :items="items" :label="words.results" @open="onOpen" />
      <p v-else-if="view.items.length > 0" class="saved-search__empty">{{ words.noFilterMatch }}</p>
      <p v-else-if="view.emptyText" class="saved-search__empty">{{ view.emptyText }}</p>
    </template>
    <p v-if="failure" role="alert" class="saved-search__failure">{{ failure }}</p>
  </section>
</template>
