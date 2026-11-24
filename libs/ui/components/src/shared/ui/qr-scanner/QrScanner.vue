<script setup lang="ts">
/**
 * Reads QR codes from the camera at eight frames a second and reports the text of the first valid one it finds.
 * It asks the machine for the camera when it mounts, so a dialog that holds it asks once it is open. The picture
 * of the camera is drawn in a frame that marks where to hold the code. The stream stops when the scanner unmounts.
 * The decoder is plain JavaScript, so no worker and no wasm is needed.
 */
import { useTemplateRef } from 'vue'
import { openMachineCamera, type ScannerCamera } from '@/shared/api/camera'
import { Button } from '../button'
import { Spinner } from '../spinner'
import { useQrScanner } from './useQrScanner'

/* ----------------------------- Props & Emits ------------------------------ */
const props = withDefaults(
  defineProps<{ camera?: ScannerCamera; isAccepted?: (text: string) => boolean }>(),
  {
    camera: openMachineCamera,
    isAccepted: () => true,
  },
)

const emit = defineEmits<{
  (e: 'scanned', text: string): void
  (e: 'failed', reason: string): void
  (e: 'unavailable', reason: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const video = useTemplateRef<HTMLVideoElement>('video')
const { status, problem, retry } = useQrScanner(video, props, emit)
</script>

<template>
  <div class="qr-scanner">
    <div class="qr-scanner__stage">
      <video ref="video" class="qr-scanner__video" autoplay muted playsinline />
      <span class="qr-scanner__frame" aria-hidden="true" />
    </div>
    <p v-if="status === 'waiting'" class="qr-scanner__note" role="status">
      <Spinner />
      Waiting for the camera.
    </p>
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

.qr-scanner__stage {
  position: relative;
  inline-size: 100%;
  max-inline-size: 22rem;
  aspect-ratio: 4 / 3;
  overflow: hidden;
  border-radius: var(--numen-radius);
  background: var(--numen-field-bg);
}

.qr-scanner__video {
  inline-size: 100%;
  block-size: 100%;
  object-fit: cover;
}

.qr-scanner__frame {
  position: absolute;
  inset-block: 18%;
  inset-inline: 30%;
  border: 0.1875rem solid var(--numen-accent);
  border-radius: var(--numen-radius);
  pointer-events: none;
}

.qr-scanner__note {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
}
</style>
