<script setup lang="ts">
/**
 * One note on the phone, as plain text until the note editor view is mounted here.
 */
import { onMounted, ref, useTemplateRef } from 'vue'
import { formatErrorCodeMessage } from '@numen/plugin-sdk'
import { NoteBar } from './note-bar'
import type { Core } from '../core'

type NoteBaselineAt = NonNullable<Awaited<ReturnType<Core['notes']['readNote']>>['at']>

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ core: Core; path: string }>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'error', message: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const prose = ref('')
const loadedNote = ref<{ prose: string; at: NoteBaselineAt } | null>(null)
const isReading = ref(true)
const editor = useTemplateRef<HTMLTextAreaElement>('editor')

/* --------------------------------- Hooks ---------------------------------- */
onMounted(async () => {
  try {
    const answer = await props.core.notes.readNote({ path: props.path })
    if (answer.error) {
      emit('error', formatErrorCodeMessage(answer.error))
      emit('close')
      return
    }
    prose.value = answer.body
    loadedNote.value = answer.at ? { prose: answer.body, at: answer.at } : null
  } finally {
    isReading.value = false
  }
  editor.value?.focus()
})

/* -------------------------------- Handlers -------------------------------- */
async function onKeep() {
  const answer = await props.core.notes.writeNote({
    path: props.path,
    body: prose.value,
    baseline: loadedNote.value ?? undefined,
  })
  if (answer.error) {
    emit('error', formatErrorCodeMessage(answer.error))
    return
  }
  emit('close')
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
/* The note covers the picture and is a column: the toolbar, then the prose. */
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
