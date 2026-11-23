<script setup lang="ts">
/**
 * Takes the text of the QR code of the other device: from the camera where the machine has one, and always from a
 * field to paste it into.
 */
import { ref } from 'vue'
import { Button, hasCamera, QrScanner } from '@numen/ui'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
defineProps<{ isBusy: boolean }>()

const emit = defineEmits<{ (e: 'submit', text: string): void }>()

/* --------------------------------- State ---------------------------------- */
const pasted = ref('')
</script>

<template>
  <form class="link-scan" @submit.prevent="emit('submit', pasted)">
    <QrScanner v-if="hasCamera()" @scanned="emit('submit', $event)" />
    <label class="link-scan__field">
      <span>{{ words.linkPaste }}</span>
      <input v-model="pasted" type="text" autocomplete="off" />
    </label>
    <Button type="submit" variant="solid" :disabled="isBusy">{{ words.linkUse }}</Button>
  </form>
</template>

<style scoped>
.link-scan,
.link-scan__field {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}
</style>
