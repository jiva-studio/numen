<script setup lang="ts">
import { onMounted, ref, useTemplateRef } from 'vue'
import { Editor } from '@numen/ui'
import { formatErrorCodeMessage } from '@numen/wire'
import { NoteBar } from './note-bar'
import type { ReadNoteResponse } from '@numen/protocol'
import type { Core } from '../core'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ core: Core; path: string }>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'error', message: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const prose = ref('')
const loadedNote = ref<{ prose: string; at: NonNullable<ReadNoteResponse['at']> } | null>(null)
const isReading = ref(true)
const editor = useTemplateRef<InstanceType<typeof Editor>>('editor')

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
      <Editor ref="editor" v-model="prose" data-testid="editor" />
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
</style>
