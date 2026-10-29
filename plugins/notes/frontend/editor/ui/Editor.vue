<script setup lang="ts">
/**
 * Markdown, written and read in the same place.
 *
 * The text in the editor is the text of the file, character for character;
 * nothing here rewrites typed content. Only the rendering of a construct
 * changes: away from the caret it is rendered formatted, and at the caret its
 * markup characters are shown.
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
  softWrap: true,
  lineNumbers: false,
  tabSize: 2,
})

const text = defineModel<string>({ default: '' })

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /** A rendered link was followed. */
  (event: 'open', address: string): void
  /** The user pressed `Ctrl+S` to save the text now. */
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
  softWrap: () => props.softWrap,
  lineNumbers: () => props.lineNumbers,
  tabSize: () => props.tabSize,
})

/* --------------------------------- Helpers -------------------------------- */
defineExpose({
  /** Take the keyboard. False while there is no editor yet to take it. */
  focus,
  /**
   * Remeasure the editor. An editor rendered while hidden has no measurements;
   * the caller signals when it is visible.
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
/* The prose being edited is selectable; the caret and the clipboard operate
   on it. */
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
