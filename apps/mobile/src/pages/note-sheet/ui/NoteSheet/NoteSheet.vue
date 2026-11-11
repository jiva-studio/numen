<script setup lang="ts">
/**
 * One note on the phone, as plain text until the note editor view is mounted here.
 */
import { onMounted, useTemplateRef } from 'vue'
import type { Core } from '@/entities/core'
import { useNoteSheet } from '../../model/useNoteSheet'
import NoteBar from '../NoteBar/NoteBar.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ core: Core; path: string }>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'error', message: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const editor = useTemplateRef<HTMLTextAreaElement>('editor')
const { prose, isReading, actions } = useNoteSheet({
  core: props.core,
  path: props.path,
  focusEditor: () => editor.value?.focus(),
  emitError: (message) => emit('error', message),
  emitClose: () => emit('close'),
})

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => actions.open())

/* -------------------------------- Handlers -------------------------------- */
function onKeep() {
  actions.save()
}
</script>

<template>
  <div class="note">
    <NoteBar :path="path" :is-reading="isReading" @close="emit('close')" @keep="onKeep" />
    <div class="note__prose">
      <textarea
        ref="editor"
        v-model="prose"
        class="note__text"
        aria-label="Note"
        :readonly="isReading"
        data-testid="editor"
      />
    </div>
  </div>
</template>

<style scoped>
/* The note covers the underlying screen and is a column: the toolbar, then the prose. */
.note {
  position: fixed;
  inset: 0;
  z-index: var(--numen-lift-sheet);
  display: flex;
  flex-direction: column;
  background: var(--numen-surface);
}

.note__prose {
  flex: 1;
  min-block-size: 0;
}

.note__text {
  box-sizing: border-box;
  inline-size: 100%;
  block-size: 100%;
  padding: var(--numen-gutter);
  border: none;
  resize: none;
  background: transparent;
  color: var(--numen-ink);
  font: inherit;
  font-size: var(--numen-reading-size);
  line-height: 1.6;
}

.note__text:focus {
  outline: none;
}
</style>
