<script setup lang="ts">
/**
 * The bar above a note: the path it is filed under, the way back, and the way
 * to keep it. The progress bar is up while the note is still being read.
 */
import { IonButton, IonButtons, IonHeader, IonProgressBar, IonTitle, IonToolbar } from '@ionic/vue'

defineProps<{ path: string; reading: boolean }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'keep'): void
}>()
</script>

<template>
  <IonHeader>
    <IonToolbar>
      <IonButtons slot="start">
        <IonButton data-testid="leave" @click="emit('close')">Back</IonButton>
      </IonButtons>
      <IonTitle size="small">{{ path }}</IonTitle>
      <IonButtons slot="end">
        <IonButton data-testid="keep" @click="emit('keep')">Keep</IonButton>
      </IonButtons>
    </IonToolbar>
    <IonProgressBar v-if="reading" type="indeterminate" />
  </IonHeader>
</template>
