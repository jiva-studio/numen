<script setup lang="ts">
/**
 * Reads QR codes from the camera at eight frames a second and reports the text of the first valid one it finds.
 * The stream stops when the scanner unmounts. The decoder is plain JavaScript, so no worker and no wasm is needed.
 */
import { useTemplateRef } from 'vue'
import { openMachineCamera, type ScannerCamera } from '@/shared/api/camera'
import { Button } from '../button'
import { useQrScanner } from './useQrScanner'

/* ----------------------------- Props & Emits ------------------------------ */
const props = withDefaults(
  defineProps<{ camera?: ScannerCamera; accepts?: (text: string) => boolean }>(),
  {
    camera: openMachineCamera,
    accepts: () => true,
  },
)

const emit = defineEmits<{
  (e: 'scanned', text: string): void
  (e: 'failed', reason: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const video = useTemplateRef<HTMLVideoElement>('video')
const { status, problem, retry } = useQrScanner(video, props, emit)
</script>

<template>
  <div class="qr-scanner">
    <video ref="video" class="qr-scanner__video" muted playsinline />
    <p v-if="status === 'waiting'" class="qr-scanner__note">Waiting for the camera.</p>
    <p v-if="problem !== ''" role="alert" class="qr-scanner__note">{{ problem }}</p>
    <Button v-if="status === 'failed'" variant="outline" size="small" @click="retry">
      Try again
    </Button>
  </div>
</template>

<style scoped>
.qr-scanner {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  align-items: center;
}

.qr-scanner__video {
  width: 100%;
  max-width: 22rem;
  border-radius: 0.5rem;
  background: var(--numen-raised);
}

.qr-scanner__note {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}
</style>
