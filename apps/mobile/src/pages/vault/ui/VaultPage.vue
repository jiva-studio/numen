<script setup lang="ts">
/**
 * The vault the core opened, and the note it opens on.
 *
 * The page starts the core and opens the vault's opening note. Closing the note
 * leaves a row naming it, and tapping the row opens it again. A vault the core
 * has no opening note for shows no row. The `note` slot draws the note and receives
 * the core, the path, and the two ways back from it: `close` and `report`. The `tree`
 * slot draws the files of the vault and receives `open`, which opens the note at a path.
 */
import { IonContent, IonItem, IonList, IonPage, IonProgressBar, IonToast } from '@ionic/vue'
import { useVault } from '../model/useVault'

const { core, errorMessage, openingPath, editingPath } = useVault()
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
      <slot v-if="core" name="tree" :open="(path: string) => (editingPath = path)" />
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
    <slot
      v-if="core && editingPath"
      name="note"
      :core="core"
      :path="editingPath"
      :close="() => (editingPath = null)"
      :report="(message: string) => (errorMessage = message)"
    />
  </Teleport>
</template>
