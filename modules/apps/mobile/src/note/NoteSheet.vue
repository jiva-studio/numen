<script setup lang="ts">
/**
 * One note, written where it is read. The prose is what the file holds below
 * its frontmatter, and what was read is carried back with the write, so a note
 * changed elsewhere meanwhile is refused.
 *
 * The editor stands in the page's own root, under the layer this draws.
 */
import { onMounted, ref, useTemplateRef } from 'vue'
import {
  IonButton,
  IonButtons,
  IonHeader,
  IonProgressBar,
  IonTitle,
  IonToolbar,
} from '@ionic/vue'
import { Editor } from '@numen/ui'
import { formatErrorCodeMessage } from '@numen/wire'
import type { ReadNoteResponse } from '@numen/protocol'
import type { Core } from '../core'

const props = defineProps<{ core: Core; path: string }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'trouble', said: string): void
}>()

const prose = ref('')
const seen = ref<{ prose: string; at: NonNullable<ReadNoteResponse['at']> } | null>(null)
const reading = ref(true)
const editor = useTemplateRef<InstanceType<typeof Editor>>('editor')

onMounted(async () => {
  try {
    const said = await props.core.notes.readNote({ path: props.path })
    if (said.error) {
      emit('trouble', formatErrorCodeMessage(said.error))
      emit('close')
      return
    }
    prose.value = said.body
    seen.value = said.at ? { prose: said.body, at: said.at } : null
  } finally {
    reading.value = false
  }
  editor.value?.focus()
})

async function keep() {
  const said = await props.core.notes.writeNote({
    path: props.path,
    body: prose.value,
    seen: seen.value ?? undefined,
  })
  if (said.error) {
    emit('trouble', formatErrorCodeMessage(said.error))
    return
  }
  emit('close')
}
</script>

<template>
  <div class="note">
    <IonHeader>
      <IonToolbar>
        <IonButtons slot="start">
          <IonButton data-testid="leave" @click="emit('close')">Back</IonButton>
        </IonButtons>
        <IonTitle size="small">{{ path }}</IonTitle>
        <IonButtons slot="end">
          <IonButton data-testid="keep" @click="keep">Keep</IonButton>
        </IonButtons>
      </IonToolbar>
      <IonProgressBar v-if="reading" type="indeterminate" />
    </IonHeader>
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
