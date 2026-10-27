<script setup lang="ts">
/**
 * The vault the core opened, and the note it opens on.
 *
 * The page starts the core and opens the vault's opening note. Closing the note
 * leaves a row naming it, and tapping the row opens it again. A vault the core
 * has no opening note for shows no row.
 */
import { onMounted, ref } from 'vue'
import { IonContent, IonItem, IonList, IonPage, IonProgressBar, IonToast } from '@ionic/vue'
import { formatErrorMessage } from '@numen/plugin-sdk'
import NoteSheet from '../note/NoteSheet.vue'
import { startCore, type Core } from '../core'

/* --------------------------------- State ---------------------------------- */
const core = ref<Core | null>(null)
const errorMessage = ref('')

/** The note the vault opens on, and nothing while it has none. */
const openingPath = ref<string | null>(null)

/** The note being written, and nothing while none is. */
const editingPath = ref<string | null>(null)

/* --------------------------------- Hooks ---------------------------------- */
onMounted(async () => {
  try {
    const started = await startCore()
    core.value = started
    const { note } = await started.notes.getOpeningNote({})
    openingPath.value = note?.path ?? null
    editingPath.value = openingPath.value
  } catch (why) {
    errorMessage.value = formatErrorMessage(why)
  }
})
</script>

<template>
  <IonPage>
    <IonContent fullscreen>
      <IonProgressBar v-if="!core && !errorMessage" type="indeterminate" />
      <IonList v-if="core && openingPath">
        <IonItem button data-testid="note" @click="editingPath = openingPath">
          {{ openingPath }}
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
