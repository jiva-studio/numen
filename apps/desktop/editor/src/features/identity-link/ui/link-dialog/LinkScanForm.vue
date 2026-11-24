<script setup lang="ts">
/**
 * Takes the text of the QR code of the other device: from the camera where the machine has one, and always from a
 * field to paste it into.
 */
import { ref } from 'vue'
import { Button, hasCamera, QrScanner, type ScannerCamera } from '@numen/ui'
import { useLinkScan } from '../../model/useLinkScan'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = withDefaults(
  defineProps<{ isBusy: boolean; findCamera?: () => Promise<boolean>; camera?: ScannerCamera }>(),
  { findCamera: hasCamera },
)

const emit = defineEmits<{ (e: 'submit', text: string): void }>()

/* --------------------------------- State ---------------------------------- */
const pasted = ref('')
const { phase, notice, setUnavailable } = useLinkScan(props.findCamera)
</script>

<template>
  <form class="link-scan" @submit.prevent="emit('submit', pasted)">
    <QrScanner
      v-if="phase === 'camera'"
      v-bind="camera === undefined ? {} : { camera }"
      @scanned="emit('submit', $event)"
      @unavailable="setUnavailable"
    />
    <p v-if="notice !== ''" role="status" class="link-scan__notice">{{ notice }}</p>
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

.link-scan__notice {
  margin: 0;
}
</style>
