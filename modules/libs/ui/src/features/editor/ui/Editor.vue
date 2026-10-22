<script setup lang="ts">
/**
 * Markdown, written and read in the same place.
 *
 * The text in the editor is the text of the file, mark for mark; nothing here
 * rewrites what was typed. What changes is how a construct is drawn: away
 * from the caret it is drawn as it reads, and where the caret stands the
 * marks come back.
 */
import { useId, useTemplateRef } from 'vue'
import { useEditor } from '../model/editor'
import type { EditorProps } from './Editor.types'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<EditorProps>(), {
  isLive: true,
  language: '',
  isReadonly: false,
  placeholder: 'Write',
  name: 'Editor',
  keys: 'Tab indents. Press Escape, then Tab, to leave the editor.',
  change: null,
  extensions: () => [],
})

const text = defineModel<string>({ default: '' })

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /** A drawn link was followed. */
  (event: 'open', address: string): void
  /** The person asked, with `Ctrl+S`, for the text to be kept now. */
  (event: 'save'): void
}>()

/* ---------------------------------- State --------------------------------- */
const host = useTemplateRef<HTMLElement>('host')

/** What the words about the keyboard are addressed by, this editor's alone. */
const keysId = `${useId()}-keys`

const { focus, measure, reveal } = useEditor({
  host,
  props,
  text,
  keysId,
  emit,
})

/* --------------------------------- Helpers -------------------------------- */
defineExpose({
  /** Take the keyboard. False while there is no editor yet to take it. */
  focus,
  /**
   * Take the editor's measurements again. An editor drawn while it is hidden
   * has none to take. The caller says when it is on screen.
   */
  measure,
  /**
   * Put the caret on one line of the prose and bring it into sight. Lines are
   * counted from the first line of the prose, and one past the end lands on the
   * last line there is.
   */
  reveal,
})
</script>

<template>
  <div ref="host" class="editor numen text-ink h-full min-h-0 overflow-auto font-sans text-base">
    <!-- Read out as the keyboard arrives, which is the one moment a person
         needs to know how to get away again. -->
    <span :id="keysId" class="sr-only">{{ keys }}</span>
  </div>
</template>

<style scoped>
/* The prose a person is writing takes a selection, and the caret and the
   clipboard work over it. */
.editor {
  user-select: text;
  -webkit-user-select: text;
}

/* A splitter above is being dragged, and the text under the pointer is left
   alone for as long as it is. */
[data-resizing] .editor :deep(.cm-content) {
  user-select: none;
  -webkit-user-select: none;
}
</style>
