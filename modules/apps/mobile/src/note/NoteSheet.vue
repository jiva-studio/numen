<script setup lang="ts">
/**
 * One note, written where it is read. The prose is what the file holds below
 * its frontmatter, and what was read is carried back with the write, so a note
 * changed elsewhere meanwhile is refused.
 *
 * The editor stands in the page's own root, under the layer this draws.
 */
import { onMounted, ref, useTemplateRef } from 'vue'
import { Editor } from '@numen/ui'
import { formatErrorCodeMessage } from '@numen/wire'
import { NoteBar } from './note-bar'
import type { ReadNoteResponse } from '@numen/protocol'
import type { Core } from '../core'

const props = defineProps<{ core: Core; path: string }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'error', message: string): void
}>()

const prose = ref('')
const seen = ref<{ prose: string; at: NonNullable<ReadNoteResponse['at']> } | null>(null)
const reading = ref(true)
const editor = useTemplateRef<InstanceType<typeof Editor>>('editor')

onMounted(async () => {
  try {
    const answer = await props.core.notes.readNote({ path: props.path })
    if (answer.error) {
      emit('error', formatErrorCodeMessage(answer.error))
      emit('close')
      return
    }
    prose.value = answer.body
    seen.value = answer.at ? { prose: answer.body, at: answer.at } : null
  } finally {
    reading.value = false
  }
  editor.value?.focus()
})

async function keep() {
  const answer = await props.core.notes.writeNote({
    path: props.path,
    body: prose.value,
    seen: seen.value ?? undefined,
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
    <NoteBar :path="path" :reading="reading" @close="emit('close')" @keep="keep" />
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
