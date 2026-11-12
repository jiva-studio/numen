<script setup lang="ts">
/**
 * The bar above a note: the path it is filed under, the way back, and the way
 * to keep it. The progress bar is up while the note is still being read.
 */
import { defineComponent, h } from 'vue'
import {
  IonButton,
  IonButtons,
  IonHeader,
  IonProgressBar,
  IonTitle,
  IonToolbar as BaseToolbar,
} from '@ionic/vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ path: string; isReading: boolean }>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'keep'): void
}>()

/* -------------------------------- Helpers --------------------------------- */
const IonToolbar = defineComponent({
  name: 'IonToolbar',
  setup(_, { slots }) {
    return () =>
      h(BaseToolbar, null, {
        default: () => [
          slots.start ? h(IonButtons, { slot: 'start' }, slots.start) : null,
          slots.default?.(),
          slots.end ? h(IonButtons, { slot: 'end' }, slots.end) : null,
        ],
      })
  },
})
</script>

<template>
  <IonHeader>
    <IonToolbar>
      <template #start>
        <IonButton data-testid="leave" @click="emit('close')">Back</IonButton>
      </template>
      <IonTitle size="small">{{ path }}</IonTitle>
      <template #end>
        <IonButton data-testid="keep" @click="emit('keep')">Keep</IonButton>
      </template>
    </IonToolbar>
    <IonProgressBar v-if="isReading" type="indeterminate" />
  </IonHeader>
</template>
