<script setup lang="ts">
/**
 * The editor of the query of the note, inside the tab: a code editor of the search language with
 * highlighting, completion of the names of the catalog, and the problems of the query underlined at
 * their place with their message on hover, in a framed box with a label and a Format button above and,
 * while the query has problems, a list of the first ten below; a click on one puts the cursor there. The
 * query is saved once typing pauses.
 */
import { computed, onMounted, useTemplateRef } from 'vue'
import { Button } from '@numen/ui'
import { WORDS as words } from '../../words'
import { getProblemText } from '../lib/problemText'
import { FormatIcon } from '../model/icons'
import type { QueryFunction, QueryProblem } from '../lib/queryLanguage'
import { useQueryEditor } from '../model/queryEditor'
import type { SearchChange } from '../model/tab'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  id: string
  query: string
  problems: readonly QueryProblem[]
  functions: readonly QueryFunction[]
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'save', change: SearchChange): void
}>()

/* --------------------------------- State ---------------------------------- */
const SHOWN_PROBLEMS = 10
const host = useTemplateRef<HTMLDivElement>('host')
const placedProblems = computed(() => props.problems.filter((problem) => problem.line >= 1))
const shownProblems = computed(() => placedProblems.value.slice(0, SHOWN_PROBLEMS))
const hiddenProblemCount = computed(() => placedProblems.value.length - shownProblems.value.length)
const editor = useQueryEditor({
  host,
  query: props.query,
  getProblems: () => props.problems,
  getFunctions: () => props.functions,
  onChange,
})

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  editor.focus()
})

/* -------------------------------- Handlers -------------------------------- */
function onChange(query: string) {
  emit('save', { query })
}

function onFormat() {
  editor.format()
}

function onPickProblem(problem: QueryProblem) {
  editor.moveTo(problem.line, problem.column)
}
</script>

<template>
  <section :id="id" class="saved-search__query" :aria-label="words.editQuery">
    <div class="saved-search__query-head">
      <span class="saved-search__query-label" aria-hidden="true">{{ words.queryField }}</span>
      <Button
        variant="ghost"
        size="icon-small"
        class="saved-search__query-format"
        :aria-label="words.formatQuery"
        :title="words.formatQueryHint"
        @click="onFormat"
      >
        <FormatIcon :size="14" aria-hidden="true" />
      </Button>
    </div>
    <div ref="host" class="saved-search__query-field" />
    <ul
      v-if="placedProblems.length > 0"
      class="saved-search__query-problems"
      :aria-label="words.queryProblems"
    >
      <li v-for="(problem, index) in shownProblems" :key="index">
        <button type="button" class="saved-search__query-problem" @click="onPickProblem(problem)">
          <span class="saved-search__query-place">{{ problem.line }}:{{ problem.column }}</span>
          <span class="saved-search__query-message">{{ getProblemText(problem.message) }}</span>
        </button>
      </li>
      <li v-if="hiddenProblemCount > 0" class="saved-search__query-more">
        {{ words.moreProblems(hiddenProblemCount) }}
      </li>
    </ul>
  </section>
</template>
