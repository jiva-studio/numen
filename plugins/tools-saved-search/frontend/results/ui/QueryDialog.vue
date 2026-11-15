<script setup lang="ts">
/**
 * The dialog that edits the query of the note in a code editor of the search language: highlighting,
 * completion of the names of the catalog, the problems of the last save underlined at their place and
 * listed below, and Save and Cancel.
 */
import { computed, onBeforeUnmount, onMounted, useTemplateRef, watch } from 'vue'
import { Button } from '@numen/ui'
import { autocompletion, completionKeymap } from '@codemirror/autocomplete'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { syntaxHighlighting } from '@codemirror/language'
import { setDiagnostics } from '@codemirror/lint'
import { EditorView, keymap } from '@codemirror/view'
import { WORDS as words } from '../../words'
import {
  createQueryCompletion,
  getQueryDiagnostics,
  queryHighlighting,
  queryLanguage,
  type QueryFunction,
  type QueryProblem,
} from '../lib/queryLanguage'
import { formatProblem } from '../lib/view'
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
let view: EditorView | null = null

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  dialog.value?.showModal()
  if (!host.value) return
  view = createEditor(host.value)
  view.focus()
})

onBeforeUnmount(() => {
  view?.destroy()
  view = null
})

watch(
  () => props.problems,
  (problems) => {
    if (!view) return
    view.dispatch(setDiagnostics(view.state, getQueryDiagnostics(view.state.doc, problems)))
  },
)

/* -------------------------------- Handlers -------------------------------- */
function onSave() {
  if (!view || props.isSaving) return
  emit('save', { query: view.state.doc.toString() })
}

function onCancel() {
  emit('cancel')
}

/* --------------------------------- Helpers -------------------------------- */
function createEditor(parent: HTMLElement): EditorView {
  const saveKey = {
    key: 'Mod-Enter',
    run: () => {
      onSave()
      return true
    },
  }
  return new EditorView({
    parent,
    doc: props.query,
    extensions: [
      history(),
      queryLanguage,
      syntaxHighlighting(queryHighlighting),
      autocompletion({ override: [createQueryCompletion(() => props.functions)] }),
      keymap.of([saveKey, ...completionKeymap, ...historyKeymap, ...defaultKeymap]),
      EditorView.lineWrapping,
      EditorView.contentAttributes.of({ 'aria-label': words.queryField }),
    ],
  })
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
