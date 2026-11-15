<script setup lang="ts">
/**
 * The editor of the query of the note, inside the tab: a code editor of the search language with
 * highlighting, completion of the names of the catalog, and the problems of the last save underlined at
 * their place with their message on hover and under the cursor. The query is saved once typing pauses.
 */
import { onMounted, useTemplateRef } from 'vue'
import { WORDS as words } from '../../words'
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
const host = useTemplateRef<HTMLDivElement>('host')
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
</script>

<template>
  <section :id="id" class="saved-search__query" :aria-label="words.editQuery">
    <div ref="host" />
  </section>
</template>
