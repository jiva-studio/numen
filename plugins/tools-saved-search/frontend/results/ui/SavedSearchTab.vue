<script setup lang="ts">
/**
 * The tab of a saved search note: the title, the dot of the state of the run and the button that shows
 * or hides the editor of the query below them; the filter and the sources of the results;
 * and the results, or why there is none with the button that brings them back.
 */
import { computed, useId } from 'vue'
import { Button } from '@numen/ui'
import { WORDS as words } from '../../words'
import { EditIcon } from '../model/icons'
import { getFilterEmpty, type EmptyReason } from '../lib/view'
import type { SavedSearchTabState, SearchChange } from '../model/tab'
import EmptyResults from './EmptyResults.vue'
import QueryEditor from './QueryEditor.vue'
import ResultList from './ResultList.vue'
import ResultsFilter from './ResultsFilter.vue'
import StatusDot from './StatusDot.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  state: SavedSearchTabState
}>()

/* --------------------------------- State ---------------------------------- */
const view = computed(() => props.state.view.value)
const items = computed(() => props.state.items.value)
const failure = computed(() => props.state.failure.value)
const filterEmpty = computed(() =>
  view.value && view.value.items.length > 0 && items.value.length === 0
    ? getFilterEmpty(
        view.value.sources,
        props.state.hiddenSources.value,
        props.state.filter.value,
        words,
      )
    : null,
)
const editorId = useId()

/* -------------------------------- Handlers -------------------------------- */
function onSave(change: SearchChange) {
  props.state.actions.saveSearch(change).catch(() => undefined)
}

function onToggleEditor() {
  props.state.actions.setEditing(!props.state.isEditing.value)
}

function onOpen(index: number) {
  props.state.actions.openResult(index)
}

function onFilter(filter: string) {
  props.state.actions.setFilter(filter)
}

function onBringBack(reason: EmptyReason) {
  if (reason === 'sources') props.state.actions.showAllSources()
  else props.state.actions.setFilter('')
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
        <StatusDot :status="view.status" />
        <Button
          variant="ghost"
          size="icon-small"
          class="saved-search__edit"
          :aria-label="words.editQuery"
          :aria-expanded="state.isEditing.value"
          :aria-controls="state.isEditing.value ? editorId : undefined"
          @click="onToggleEditor"
        >
          <EditIcon :size="16" aria-hidden="true" />
        </Button>
      </header>
      <QueryEditor
        v-if="state.isEditing.value"
        :id="editorId"
        :query="view.query"
        :problems="state.queryProblems.value"
        :functions="state.functions.value"
        @save="onSave"
      />
      <ResultsFilter
        v-if="view.items.length > 0"
        :filter="state.filter.value"
        :sources="view.sources"
        :hidden="state.hiddenSources.value"
        @filter="onFilter"
        @toggle="onToggleSource"
      />
      <ResultList v-if="items.length > 0" :items="items" :label="words.results" @open="onOpen" />
      <EmptyResults
        v-else-if="filterEmpty"
        :text="filterEmpty.text"
        :action-text="filterEmpty.actionText"
        @act="onBringBack(filterEmpty.reason)"
      />
      <EmptyResults v-else-if="view.emptyText" :text="view.emptyText" />
    </template>
    <p v-if="failure" role="alert" class="saved-search__failure">{{ failure }}</p>
  </section>
</template>
