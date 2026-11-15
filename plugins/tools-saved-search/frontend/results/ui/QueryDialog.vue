<script setup lang="ts">
/**
 * The dialog that edits the query of the note in a code editor of the search language: highlighting,
 * completion of the names of the catalog, the problems of the last save underlined at their place and
 * listed below, and Save and Cancel.
 */
import { computed, onMounted, useTemplateRef } from 'vue'
import { Button } from '@numen/ui'
import { WORDS as words } from '../../words'
import type { QueryFunction, QueryProblem } from '../lib/queryLanguage'
import { formatProblem } from '../lib/view'
import { useQueryEditor } from '../model/queryEditor'
import type { SearchChange } from '../model/tab'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  query: string
  problems: readonly QueryProblem[]
  functions: readonly QueryFunction[]
  isSaving: boolean
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'save', change: SearchChange): void
  (event: 'cancel'): void
}>()

/* --------------------------------- State ---------------------------------- */
const dialog = useTemplateRef<HTMLDialogElement>('dialog')
const host = useTemplateRef<HTMLDivElement>('host')
const problemTexts = computed(() => props.problems.map(formatProblem))
const editor = useQueryEditor({
  host,
  query: props.query,
  getProblems: () => props.problems,
  getFunctions: () => props.functions,
  onSave,
})

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  dialog.value?.showModal()
  editor.focus()
})

/* -------------------------------- Handlers -------------------------------- */
function onSave() {
  if (props.isSaving) return
  emit('save', { query: editor.getQuery() })
}

function onCancel() {
  emit('cancel')
}
</script>

<template>
  <dialog
    ref="dialog"
    class="saved-search__dialog"
    :aria-label="words.editQuery"
    @cancel.prevent="onCancel"
  >
    <form class="saved-search__dialog-form" @submit.prevent="onSave">
      <div ref="host" class="saved-search__query-editor" />
      <p v-for="problem in problemTexts" :key="problem" role="alert" class="saved-search__problem">
        {{ problem }}
      </p>
      <div class="saved-search__dialog-actions">
        <Button type="button" variant="outline" @click="onCancel">{{ words.cancel }}</Button>
        <Button type="submit" :disabled="isSaving">{{ words.save }}</Button>
      </div>
    </form>
  </dialog>
</template>
