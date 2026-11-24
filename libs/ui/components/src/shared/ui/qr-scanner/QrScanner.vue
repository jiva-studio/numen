<script setup lang="ts">
/**
 * Reads QR codes from the camera at eight frames a second and reports the text of the first valid one it finds.
 * It asks the machine for the camera when it mounts, so a dialog that holds it asks once it is open. The picture
 * of the camera fills the width it is given, at 4:3. The stream stops after the first scan; with `continuous` it
 * keeps reading and reports each new code (the same code held in front of the camera once) until the scanner unmounts.
 * The decoder is plain JavaScript, so no worker and no wasm is needed.
 */
import { useTemplateRef } from 'vue'
import { openMachineCamera, type ScannerCamera } from '@/shared/api/camera'
import { Spinner } from '../spinner'
import { useQrScanner } from './useQrScanner'

/* ----------------------------- Props & Emits ------------------------------ */
const props = withDefaults(
  defineProps<{
    camera?: ScannerCamera
    isAccepted?: (text: string) => boolean
    continuous?: boolean
  }>(),
  {
    camera: openMachineCamera,
    isAccepted: () => true,
    continuous: false,
  },
)

const emit = defineEmits<{
  (e: 'scanned', text: string): void
  (e: 'failed', reason: string): void
  (e: 'unavailable', reason: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const video = useTemplateRef<HTMLVideoElement>('video')
const { status, problem } = useQrScanner(video, props, emit)
</script>

<template>
  <div class="qr-scanner">
    <div class="qr-scanner__stage">
      <video ref="video" class="qr-scanner__video" autoplay muted playsinline />
    </div>
    <p v-if="status === 'waiting'" class="qr-scanner__note" role="status">
      <Spinner />
      Waiting for the camera.
    </p>
    <p v-if="problem !== ''" role="alert" class="qr-scanner__note">{{ problem }}</p>
  </div>
</template>

<style scoped>
.qr-scanner {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.qr-scanner__stage {
  inline-size: 100%;
  aspect-ratio: 4 / 3;
  overflow: hidden;
  border-radius: var(--numen-radius);
  background: var(--numen-field-bg);
}

.qr-scanner__video {
  display: block;
  inline-size: 100%;
  block-size: 100%;
  object-fit: cover;
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
