<script setup lang="ts">
/**
 * One note, written where it is read.
 *
 * The prose is what the file holds below its frontmatter. What was read is
 * carried back with the write, so a note changed elsewhere meanwhile is
 * refused rather than overwritten.
 *
 * The sheet is a layer over the page and not an overlay of the framework's.
 * An overlay holds its content in a shadow root while it comes up and moves it
 * out afterwards; an editor built in there paints itself into a root it then
 * leaves behind, and is drawn with none of its own styles.
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
import type { ReadResponse } from '@numen/protocol'
import type { Reached } from '../core'

const props = defineProps<{ core: Reached; path: string }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'trouble', said: string): void
}>()

const prose = ref('')
const seen = ref<{ prose: string; at: NonNullable<ReadResponse['at']> } | null>(null)
const reading = ref(true)
const editor = useTemplateRef<InstanceType<typeof Editor>>('editor')

onMounted(async () => {
  try {
    const said = await props.core.vault.read({ path: props.path })
    if (said.refusal) {
      emit('trouble', `the note was not read: ${JSON.stringify(said.refusal)}`)
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
  const said = await props.core.vault.write({
    path: props.path,
    body: prose.value,
    seen: seen.value ?? undefined,
  })
  if (said.refusal) {
    emit('trouble', `nothing was written: ${JSON.stringify(said.refusal)}`)
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
  z-index: 10;
  display: flex;
  flex-direction: column;
  background: var(--numen-surface);
}

.note__prose {
  flex: 1;
  min-height: 0;
}
</style>
