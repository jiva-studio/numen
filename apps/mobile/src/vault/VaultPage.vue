<script setup lang="ts">
/**
 * The vault the core opened, and the note it opens on.
 *
 * The page starts the core and opens the seeded note. Closing the note leaves
 * a row naming it, and tapping the row opens it again.
 */
import { onMounted, ref } from 'vue'
import { IonContent, IonItem, IonList, IonPage, IonProgressBar, IonToast } from '@ionic/vue'
import { formatErrorMessage } from '@numen/wire'
import NoteSheet from '../note/NoteSheet.vue'
import { startCore, type Core } from '../core'

/** The note a seeded vault opens on. The Go side writes it and names it. */
const SEED_NOTE_PATH = 'Physics.md'

/* --------------------------------- State ---------------------------------- */
const core = ref<Core | null>(null)
const errorMessage = ref('')

/** The note being written, and nothing while none is. */
const editingPath = ref<string | null>(null)

/* --------------------------------- Hooks ---------------------------------- */
onMounted(async () => {
  try {
    core.value = await startCore()
    editingPath.value = SEED_NOTE_PATH
  } catch (why) {
    errorMessage.value = formatErrorMessage(why)
  }
})
</script>

<template>
  <IonPage>
    <IonContent fullscreen>
      <IonProgressBar v-if="!core && !errorMessage" type="indeterminate" />
      <IonList v-if="core">
        <IonItem button data-testid="note" @click="editingPath = SEED_NOTE_PATH">
          {{ SEED_NOTE_PATH }}
        </IonItem>
      </IonList>
      <IonToast
        :is-open="!!errorMessage"
        :message="errorMessage"
        color="danger"
        :duration="6000"
        data-testid="error"
        @did-dismiss="errorMessage = ''"
      />
    </IonContent>
  </IonPage>
  <!-- The note is put under the body: an editor keeps whichever root it was
       built in, and every wrapper of the framework has one of its own. -->
  <Teleport to="body">
    <NoteSheet
      v-if="core && editingPath"
      :core="core"
      :path="editingPath"
      @close="editingPath = null"
      @error="(message: string) => (errorMessage = message)"
    />
  </Teleport>
</template>
